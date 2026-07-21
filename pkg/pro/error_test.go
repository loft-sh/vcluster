package pro

import (
	"strings"
	"testing"

	"github.com/loft-sh/admin-apis/pkg/licenseapi"
)

func TestNewFeatureError(t *testing.T) {
	tests := []struct {
		name            string
		featureName     string
		expectedDisplay string
		shouldContain   []string
	}{
		{
			name:            "valid feature - vclusters",
			featureName:     "vclusters",
			expectedDisplay: "Virtual Cluster Management",
			shouldContain: []string{
				"you are trying to use a vCluster pro feature",
				"Virtual Cluster Management",
				"vclusters",
				"support@loft.sh",
			},
		},
		{
			name:            "valid feature - vcp-distro-embedded-etcd",
			featureName:     "vcp-distro-embedded-etcd",
			expectedDisplay: "Embedded etcd",
			shouldContain: []string{
				"you are trying to use a vCluster pro feature",
				"Embedded etcd",
				"vcp-distro-embedded-etcd",
				"support@loft.sh",
			},
		},
		{
			name:            "invalid feature name",
			featureName:     "non-existent-feature",
			expectedDisplay: "non-existent-feature",
			shouldContain: []string{
				"you are trying to use a vCluster pro feature",
				"non-existent-feature",
				"non-existent-feature",
				"support@loft.sh",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewFeatureError(licenseapi.FeatureName(tt.featureName))
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			errMsg := err.Error()

			// Check that all expected strings are present in the error message
			for _, expected := range tt.shouldContain {
				if !strings.Contains(errMsg, expected) {
					t.Errorf("error message should contain %q, got: %s", expected, errMsg)
				}
			}

			// Verify the display name is in the error
			if !strings.Contains(errMsg, tt.expectedDisplay) {
				t.Errorf("expected display name %q in error message, got: %s", tt.expectedDisplay, errMsg)
			}

			// Verify the feature name is in the error
			if !strings.Contains(errMsg, tt.featureName) {
				t.Errorf("expected feature name %q in error message, got: %s", tt.featureName, errMsg)
			}
		})
	}
}
