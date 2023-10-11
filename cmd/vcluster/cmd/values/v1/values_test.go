package values

import (
	"os"
	"testing"

	"k8s.io/apimachinery/pkg/util/yaml"
)

var sourceFiles = []string{
	"../../../../../charts/k3s/values.yaml",
}

func TestHelmValues(t *testing.T) {
	sourceValuesRaw, err := os.ReadFile(sourceFiles[0])
	if err != nil {
		t.Fatalf("error reading source values files %v", err)
	}

	var helmValues = &Helm{}
	err = yaml.UnmarshalStrict(sourceValuesRaw, helmValues)
	if err != nil {
		t.Fatalf("error unmarshalling raw values %s", err)
	}
}
