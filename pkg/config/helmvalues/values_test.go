package helmvalues

import (
	"os"
	"testing"

	"github.com/loft-sh/vcluster-values/helmvalues"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestValues(t *testing.T) {
	testCases := map[string]string{
		"k0s": "../../../charts/k0s/values.yaml",
		"k3s": "../../../charts/k3s/values.yaml",
		"k8s": "../../../charts/k8s/values.yaml",
		"eks": "../../../charts/eks/values.yaml",
	}
	for name, path := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Logf("testing values for %s", name)
			sourceValuesRaw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("error reading source values files %v", err)
			}

			var helmValues interface{}

			switch name {
			case "k0s":
				helmValues = &helmvalues.K0s{}
			case "k3s":
				helmValues = &helmvalues.K3s{}
			case "k8s", "eks":
				helmValues = &helmvalues.K8s{}
			}

			err = yaml.UnmarshalStrict(sourceValuesRaw, helmValues)
			if err != nil {
				t.Fatalf("error unmarshalling raw values %s", err)
			}
		})
	}
}
