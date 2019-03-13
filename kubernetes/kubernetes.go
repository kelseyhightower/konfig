package kubernetes

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
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

const runEndpoint = "https://%s-run.googleapis.com/apis/serving.knative.dev/v1alpha1/%s"

func Parse() error {
	region := os.Getenv("GOOGLE_CLOUD_REGION")
	service := serviceName()

	e := fmt.Sprintf(runEndpoint, region, service)

	httpClient, err := google.DefaultClient(oauth2.NoContext,
		"https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return err
	}

	containerService, err := container.New(httpClient)
	if err != nil {
		return err
	}

	resp, err := httpClient.Get(e)
	if err != nil {
		return err
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var s Service
	err = json.Unmarshal(data, &s)
	if err != nil {
		return err
	}

	for _, env := range s.Spec.RunLatest.Configuration.RevisionTemplate.Spec.Container.Env {
		if !IsSecretReferenc(env.Value) {
			continue
		}

		secretReference, err := ParseReference(env.Value)
		if err != nil {
			return err
		}
		cluster := strings.TrimPrefix(secretReference.Cluster, "/")
		resp, err := containerService.Projects.Locations.Clusters.Get(cluster).Context(context.Background()).Do()
		if err != nil {
			return err
		}

		kUrl := fmt.Sprintf("https://%s/api/v1/namespaces/%s/secrets/%s/", resp.Endpoint,
			secretReference.Namespace, secretReference.Name)

		caCert, err := base64.StdEncoding.DecodeString(resp.MasterAuth.ClusterCaCertificate)
		if err != nil {
			return err
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
			return err
		}

		oauthTransport := &oauth2.Transport{
			Base:   tr,
			Source: ts,
		}

		kubernetesClient := &http.Client{Transport: oauthTransport}

		kResp, err := kubernetesClient.Get(kUrl)
		data, err := ioutil.ReadAll(kResp.Body)
		if err != nil {
			return err
		}

		var secret Secret
		err = json.Unmarshal(data, &secret)
		if err != nil {
			return err
		}

		envData, err := base64.StdEncoding.DecodeString(secret.Data[secretReference.Key])
		if err != nil {
			return err
		}

		if secretReference.TempFile != nil {
			err = secretReference.TempFile.Chmod(600)
			if err != nil {
				return err
			}

			_, err = secretReference.TempFile.Write(envData)
			if err != nil {
				return err
			}

			err = secretReference.TempFile.Close()
			if err != nil {
				return err
			}

			os.Setenv(env.Name, secretReference.TempFile.Name())

			continue
		}

		os.Setenv(env.Name, string(envData))
	}

	return nil
}

func serviceName() string {
	service := os.Getenv("K_SERVICE")
	project := os.Getenv("GOOGLE_CLOUD_PROJECT")
	return fmt.Sprintf("namespaces/%s/services/%s", project, service)
}

func IsSecretReferenc(s string) bool {
	if !strings.HasPrefix(s, "$SecretKeyRef:") {
		return false
	}
	return true
}

func ParseReference(r string) (*SecretReference, error) {
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
		tempFile, err = ioutil.TempFile("", os.Getenv("K_SERVICE"))
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
