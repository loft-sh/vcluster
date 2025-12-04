package platform

import (
	"testing"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/start"
)

func TestPlatformUsesNewActivationFlow(t *testing.T) {
	testCases := []struct {
		version  string
		expected bool
	}{
		{"", false},
		{"dev", false},
		{"4.5.0", false},
		{"4.5.1", false},
		{"4.6.0-alpha.5", false},
		{"4.6.0-next.2", false},
		{"4.6.0-next.internal.0", false},
		{"4.6.0-next.internal.1", true},
		{"4.6.0-next.internal.2", true},
		{"4.6.0", true},
	}

	globalFlags := &flags.GlobalFlags{}
	startCmd := &StartCmd{
		StartOptions: start.StartOptions{
			Options: start.Options{
				CommandName: "start",
				GlobalFlags: globalFlags,
				Log:         log.GetInstance(),
			},
		},
	}

	for _, testCase := range testCases {
		result := startCmd.platformUsesNewActivationFlow(testCase.version)
		if result != testCase.expected {
			t.Errorf("Expected %v, got %v for platform version %s", testCase.expected, result, testCase.version)
		}
	}
}
