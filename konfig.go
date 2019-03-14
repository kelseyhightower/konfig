package konfig

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudfunctions/v1"
	"google.golang.org/api/container/v1"
)

type SecretReference struct {
	Cluster   string
	Namespace string
	Name      string
	Key       string
	TempFile  *os.File
}

type ConfigMapReference struct {
	Cluster   string
	Namespace string
	Name      string
	Key       string
	TempFile  *os.File
}

type Secret struct {
	ApiVersion string            `json:"apiVersion"`
	Data       map[string]string `json:"data"`
	Kind       string            `json:"kind"`
}

type RuntimeEnvironment string

const (
	CloudFunctionsRuntime = RuntimeEnvironment("cloudfunctions")
	CloudRunRuntime       = RuntimeEnvironment("cloudrun")
	UnknownRuntime        = RuntimeEnvironment("unknown")
)

const runEndpoint = "https://us-central1-run.googleapis.com/apis/serving.knative.dev/v1alpha1/%s"

func init() {
	parse()
}

func parse() {
	runtimeEnvironment := detectRuntimeEnvironment()
	if runtimeEnvironment == UnknownRuntime {
		log.Println("konfig: unknown runtime environment")
		return
	}

	environmentVariables, err := getEnvironmentVariables(runtimeEnvironment)
	if err != nil {
		log.Println(err)
		return
	}

	if len(environmentVariables) == 0 {
		return
	}

	// Setup the GKE HTTP client.
	httpClient, err := google.DefaultClient(oauth2.NoContext,
		"https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		log.Println(err)
		return
	}

	containerService, err := container.New(httpClient)
	if err != nil {
		log.Println(err)
		return
	}

	// Process the environment variable with secret references.
	for k, v := range environmentVariables {
		if !isSecretReference(v) {
			continue
		}

		secretReference, err := parseSecretReference(v)
		if err != nil {
			log.Println(err)
			return
		}

		cluster := strings.TrimPrefix(secretReference.Cluster, "/")

		resp, err := containerService.Projects.Locations.Clusters.Get(cluster).Context(context.Background()).Do()
		if err != nil {
			log.Println(err)
			return
		}

		kUrl := fmt.Sprintf("https://%s/api/v1/namespaces/%s/secrets/%s/", resp.Endpoint,
			secretReference.Namespace, secretReference.Name)

		caCert, err := base64.StdEncoding.DecodeString(resp.MasterAuth.ClusterCaCertificate)
		if err != nil {
			log.Println(err)
			return
		}

		roots := x509.NewCertPool()
		roots.AppendCertsFromPEM(caCert)

		tr := &http.Transport{
			MaxIdleConns:    10,
			IdleConnTimeout: 30 * time.Second,
			TLSClientConfig: &tls.Config{
				RootCAs: roots,
			},
		}

		ts, err := google.DefaultTokenSource(context.TODO(), "https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			log.Println(err)
			return
		}

		oauthTransport := &oauth2.Transport{
			Base:   tr,
			Source: ts,
		}

		kubernetesClient := &http.Client{Transport: oauthTransport}

		kResp, err := kubernetesClient.Get(kUrl)
		data, err := ioutil.ReadAll(kResp.Body)
		if err != nil {
			log.Println(err)
			return
		}

		defer kResp.Body.Close()

		if kResp.StatusCode != 200 {
			log.Printf("kconfig: unable to get secret %s from Kubernetes status code %v", k, kResp.StatusCode)
			continue
		}

		var secret Secret
		err = json.Unmarshal(data, &secret)
		if err != nil {
			log.Println(err)
			return
		}

		envData, err := base64.StdEncoding.DecodeString(secret.Data[secretReference.Key])
		if err != nil {
			log.Println(err)
			return
		}

		if secretReference.TempFile != nil {
			err = secretReference.TempFile.Chmod(600)
			if err != nil {
				log.Println(err)
				return
			}

			_, err = secretReference.TempFile.Write(envData)
			if err != nil {
				log.Println(err)
				return
			}

			err = secretReference.TempFile.Close()
			if err != nil {
				log.Println(err)
				return
			}

			os.Setenv(k, secretReference.TempFile.Name())

			continue
		}

		os.Setenv(k, string(envData))
	}
}

func detectRuntimeEnvironment() RuntimeEnvironment {
	if os.Getenv("FUNCTION_NAME") != "" {
		return CloudFunctionsRuntime
	}

	if os.Getenv("K_SERVICE") != "" {
		return CloudRunRuntime
	}

	return UnknownRuntime
}

func getEnvironmentVariables(e RuntimeEnvironment) (map[string]string, error) {
	switch e {
	case CloudRunRuntime:
		return getCloudRunEnvironmentVariables()
	case CloudFunctionsRuntime:
		return getCloudFunctionsEnvironmentVariables()
	}

	return nil, errors.New("unknown runtime environment")
}

func getCloudFunctionsEnvironmentVariables() (map[string]string, error) {
	oauthHttpClient, err := google.DefaultClient(oauth2.NoContext,
		"https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, err
	}

	client, err := cloudfunctions.New(oauthHttpClient)
	if err != nil {
		return nil, err
	}

	cloudFunction, err := client.Projects.Locations.Functions.Get(functionName()).Do()
	if err != nil {
		return nil, err
	}

	return cloudFunction.EnvironmentVariables, nil
}

func getCloudRunEnvironmentVariables() (map[string]string, error) {
	httpClient, err := google.DefaultClient(oauth2.NoContext,
		"https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, err
	}

	runEndPointUrl := fmt.Sprintf(runEndpoint, serviceName())

	resp, err := httpClient.Get(runEndPointUrl)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var s Service
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}

	environmentVariables := make(map[string]string)
	for _, env := range s.Spec.RunLatest.Configuration.RevisionTemplate.Spec.Container.Env {
		environmentVariables[env.Name] = env.Value
	}

	return environmentVariables, nil
}

func serviceName() string {
	service := os.Getenv("K_SERVICE")
	project := os.Getenv("GOOGLE_CLOUD_PROJECT")
	return fmt.Sprintf("namespaces/%s/services/%s", project, service)
}

func functionName() string {
	name := os.Getenv("FUNCTION_NAME")
	project := os.Getenv("GCP_PROJECT")
	region := os.Getenv("FUNCTION_REGION")

	return fmt.Sprintf("projects/%s/locations/%s/functions/%s", project, region, name)
}

func isSecretReference(s string) bool {
	if !strings.HasPrefix(s, "$SecretKeyRef:") {
		return false
	}
	return true
}

func parseSecretReference(r string) (*SecretReference, error) {
	if !strings.HasPrefix(r, "$SecretKeyRef:") {
		return nil, errors.New("missing secret key reference prefix")
	}

	path := strings.TrimPrefix(r, "$SecretKeyRef:")

	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	ss := strings.SplitN(u.Path, "/", 13)

	var tempFile *os.File
	if u.Query().Get("tempFile") != "" {
		tempFile, err = ioutil.TempFile("", "")
		if err != nil {
			return nil, err
		}
	}

	sr := &SecretReference{
		Cluster:   strings.Join(ss[0:7], "/"),
		Namespace: ss[8],
		Name:      ss[10],
		Key:       ss[12],
		TempFile:  tempFile,
	}

	return sr, nil
}
