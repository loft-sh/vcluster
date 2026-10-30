package synccontext

import (
	"testing"

	"k8s.io/apimachinery/pkg/version"
)

func TestParseClusterVersion(t *testing.T) {
	testCases := []struct {
		name            string
		info            *version.Info
		expectedVersion string
		expectError     bool
	}{
		{
			name:        "nil info is an error",
			info:        nil,
			expectError: true,
		},
		{
			name:            "plain upstream version",
			info:            &version.Info{GitVersion: "v1.34.0"},
			expectedVersion: "1.34.0",
		},
		{
			name:            "EKS vendor suffix",
			info:            &version.Info{GitVersion: "v1.34.5-eks-abc123"},
			expectedVersion: "1.34.5",
		},
		{
			name:            "k3s build metadata",
			info:            &version.Info{GitVersion: "v1.33.1+k3s1"},
			expectedVersion: "1.33.1",
		},
		{
			name:            "major.minor only",
			info:            &version.Info{GitVersion: "v1.34"},
			expectedVersion: "1.34",
		},
		{
			name:        "unparsable version is an error",
			info:        &version.Info{GitVersion: "vendor-build-xyz"},
			expectError: true,
		},
		{
			name:        "empty version is an error",
			info:        &version.Info{},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := ParseClusterVersion(tc.info)
			if tc.expectError {
				if err == nil {
					t.Fatalf("expected error, got version %s", parsed)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tc.expectedVersion == "" {
				if parsed != nil {
					t.Fatalf("expected nil version, got %s", parsed)
				}
				return
			}
			if parsed == nil {
				t.Fatalf("expected version %s, got nil", tc.expectedVersion)
			}
			if parsed.String() != tc.expectedVersion {
				t.Fatalf("expected version %s, got %s", tc.expectedVersion, parsed)
			}
		})
	}
}
