package coredns

import (
	"testing"

	"k8s.io/apimachinery/pkg/version"

	"gotest.tools/v3/assert"
)

func TestGetManifestVariablesImage(t *testing.T) {
	tests := []struct {
		name                 string
		major                string
		minor                string
		defaultImageRegistry string
		expectedImage        string
	}{
		{
			name:          "known version is fully qualified with docker.io",
			major:         "1",
			minor:         "34",
			expectedImage: "docker.io/coredns/coredns:1.12.1",
		},
		{
			name:          "unknown version falls back to fully qualified default image",
			major:         "1",
			minor:         "99",
			expectedImage: "docker.io/" + DefaultImage,
		},
		{
			name:                 "custom registry takes precedence over docker.io",
			major:                "1",
			minor:                "34",
			defaultImageRegistry: "my.registry:5000",
			expectedImage:        "my.registry:5000/coredns/coredns:1.12.1",
		},
		{
			name:                 "custom registry trailing slash is trimmed",
			major:                "1",
			minor:                "34",
			defaultImageRegistry: "my.registry:5000/",
			expectedImage:        "my.registry:5000/coredns/coredns:1.12.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := getManifestVariables(tt.defaultImageRegistry, &version.Info{Major: tt.major, Minor: tt.minor})
			assert.Equal(t, vars[VarImage], tt.expectedImage)
		})
	}
}
