// Copyright 2019 The Konfig Authors. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

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
	"runtime"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudfunctions/v1"
	"google.golang.org/api/container/v1"
)

type Reference struct {
	Cluster   string
	Namespace string
	Name      string
	Key       string
	TempFile  *os.File
	Kind      string
}

type Secret struct {
	ApiVersion string            `json:"apiVersion"`
	Data       map[string]string `json:"data"`
	Kind       string            `json:"kind"`
}

type ConfigMap struct {
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

const runEndpoint = "https://%s-run.googleapis.com/apis/serving.knative.dev/v1/%s"

var (
	projectName    = "konfig"
	projectVersion = "0.1.0"
	projectURL     = "https://github.com/kelseyhightower/konfig"
	userAgent      = fmt.Sprintf("%s/%s (+%s; %s)",
		projectName, projectVersion, projectURL, runtime.Version())
)

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
	containerService.UserAgent = userAgent

	// Process the environment variable with secret references.
	for k, v := range environmentVariables {
		if !isReference(v) {
			continue
		}

		reference, err := parseReference(v)
		if err != nil {
			log.Println(err)
			continue
		}

		clusterID := strings.TrimPrefix(reference.Cluster, "/")

		cluster, err := containerService.Projects.Locations.Clusters.Get(clusterID).Context(context.Background()).Do()
		if err != nil {
			log.Println(err)
			continue
		}

		resourceURL := fmt.Sprintf("https://%s/api/v1/namespaces/%s/%ss/%s/", cluster.Endpoint,
			reference.Namespace, reference.Kind, reference.Name)

		caCert, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClusterCaCertificate)
		if err != nil {
			log.Println(err)
			continue
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
			continue
		}

		oauthTransport := &oauth2.Transport{
			Base:   tr,
			Source: ts,
		}

		kubernetesClient := &http.Client{Transport: oauthTransport}

		resp, err := kubernetesClient.Get(resourceURL)
		if err != nil {
			log.Println(err)
			continue
		}

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			log.Printf("kconfig: unable to get %s %s from Kubernetes status code %v",
				k, reference.Kind, resp.StatusCode)
			continue
		}

		var envData string

		if reference.Kind == "secret" {
			var secret Secret
			err := json.Unmarshal(data, &secret)
			if err != nil {
				log.Println(err)
				continue
			}

			d, err := base64.StdEncoding.DecodeString(secret.Data[reference.Key])
			if err != nil {
				log.Println(err)
				continue
			}
			envData = string(d)
		}

		if reference.Kind == "configmap" {
			var configmap ConfigMap

			err := json.Unmarshal(data, &configmap)
			if err != nil {
				log.Println(err)
				continue
			}

			envData = configmap.Data[reference.Key]
		}

		if reference.TempFile != nil {
			err = reference.TempFile.Chmod(600)
			if err != nil {
				log.Println(err)
				continue
			}

			_, err = reference.TempFile.WriteString(envData)
			if err != nil {
				log.Println(err)
				continue
			}

			err = reference.TempFile.Close()
			if err != nil {
				log.Println(err)
				continue
			}

			os.Setenv(k, reference.TempFile.Name())

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
	client.UserAgent = userAgent

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

	resp, err := httpClient.Get(runEndPointUrl())
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
	for _, container := range s.Spec.RevisionTemplate.Spec.Containers {
		for _, env := range container.Env {
			environmentVariables[env.Name] = env.Value
		}
	}

	return environmentVariables, nil
}

func runEndPointUrl() string {
	region := os.Getenv("REGION")
	if region == "" {
		region = "us-central1"
	}
	return fmt.Sprintf(runEndpoint, region, serviceName())
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

func isReference(s string) bool {
	if strings.HasPrefix(s, "$SecretKeyRef:") || strings.HasPrefix(s, "$ConfigMapKeyRef:") {
		return true
	}
	return false
}

func parseReference(s string) (*Reference, error) {
	var path string
	var kind string

	if strings.HasPrefix(s, "$ConfigMapKeyRef:") {
		path = strings.TrimPrefix(s, "$ConfigMapKeyRef:")
		kind = "configmap"
	}

	if strings.HasPrefix(s, "$SecretKeyRef:") {
		path = strings.TrimPrefix(s, "$SecretKeyRef:")
		kind = "secret"
	}

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

	r := &Reference{
		Cluster:   strings.Join(ss[0:7], "/"),
		Namespace: ss[8],
		Name:      ss[10],
		Key:       ss[12],
		Kind:      kind,
		TempFile:  tempFile,
	}

	return r, nil
}
