package konfig

import (
	"os"
	"testing"
)

func TestParseSecretReference(t *testing.T) {
	r := "$SecretKeyRef:/projects/hightowerlabs/locations/us-central1/clusters/api/namespaces/default/secrets/app/keys/foo"

	_, err := parseReference(r)
	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestRunEndPointUrl(t *testing.T) {
	os.Clearenv()
	os.Setenv("K_SERVICE", "env")
	os.Setenv("GOOGLE_CLOUD_PROJECT", "hightowerlabs")
	os.Setenv("REGION", "asia-northeast1")

	want := "https://asia-northeast1-run.googleapis.com/apis/serving.knative.dev/v1alpha1/namespaces/hightowerlabs/services/env"
	got := runEndPointUrl()

	if got != want {
		t.Errorf("expected %s, got %v", want, got)
	}
}

func TestRunEndPointUrlDefaultRegion(t *testing.T) {
	os.Clearenv()
	os.Setenv("K_SERVICE", "env")
	os.Setenv("GOOGLE_CLOUD_PROJECT", "hightowerlabs")

	want := "https://us-central1-run.googleapis.com/apis/serving.knative.dev/v1alpha1/namespaces/hightowerlabs/services/env"
	got := runEndPointUrl()

	if got != want {
		t.Errorf("expected %s, got %v", want, got)
	}
}
