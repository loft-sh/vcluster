package helmvalues

import (
	"os"
	"testing"

	"k8s.io/apimachinery/pkg/util/yaml"
)

var sourceFiles = map[string]string{
	"k0s": "../../../charts/k0s/values.yaml",
	"k3s": "../../../charts/k3s/values.yaml",
	"k8s": "../../../charts/k8s/values.yaml",
	"eks": "../../../charts/eks/values.yaml",
}

func TestHelmValues(t *testing.T) {
	for k, v := range sourceFiles {
		t.Logf("testing values for %s", k)
		sourceValuesRaw, err := os.ReadFile(v)
		if err != nil {
			t.Fatalf("error reading source values files %v", err)
		}

		var helmValues interface{}

		switch k {
		case "k0s":
			helmValues = &K0s{}
		case "k3s":
			helmValues = &K3s{}
		case "k8s", "eks":
			helmValues = &K8s{}
		}

		err = yaml.UnmarshalStrict(sourceValuesRaw, helmValues)
		if err != nil {
			t.Fatalf("error unmarshalling raw values %s", err)
		}
	}
}
