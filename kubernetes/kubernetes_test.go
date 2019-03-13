package kubernetes

import (
	"os"
	"fmt"
	"testing"
)

func TestParse(t *testing.T) {
	err := Parse()
	if err != nil {
		t.Errorf(err.Error())
	}

	fmt.Println(os.Getenv("FOO"))
	fmt.Println(os.Getenv("CONFIG_FILE"))
}

func TestParseReference(t *testing.T) {
	r := "$SecretKeyRef:/projects/hightowerlabs/locations/us-central1/clusters/api/namespaces/default/secrets/app/keys/foo?mountPath=/etc/app/foo"

	_, err := ParseReference(r)
	if err != nil {
		t.Errorf(err.Error())
	}
}
