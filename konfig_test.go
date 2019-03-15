package konfig

import (
	"testing"
)

func TestParseSecretReference(t *testing.T) {
	r := "$SecretKeyRef:/projects/hightowerlabs/locations/us-central1/clusters/api/namespaces/default/secrets/app/keys/foo"

	_, err := parseReference(r)
	if err != nil {
		t.Errorf(err.Error())
	}
}
