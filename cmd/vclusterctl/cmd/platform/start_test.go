package platform

import (
	"testing"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/start"
)

func TestNewStartCmd_InsecureFlag(t *testing.T) {
	globalFlags := &flags.GlobalFlags{}
	cmd := NewStartCmd(globalFlags)

	// Verify --insecure flag exists and defaults to false.
	f := cmd.Flags().Lookup("insecure")
	if f == nil {
		t.Fatal("--insecure flag not registered on start command")
	}
	if f.DefValue != "false" {
		t.Errorf("expected --insecure default to be 'false', got %q", f.DefValue)
	}

	// Simulate passing --insecure on the command line.
	if err := cmd.Flags().Set("insecure", "true"); err != nil {
		t.Fatalf("failed to set --insecure flag: %v", err)
	}
	if f.Value.String() != "true" {
		t.Errorf("expected --insecure value to be 'true' after set, got %q", f.Value.String())
	}
}

func TestPlatformUsesNewActivationFlow(t *testing.T) {
	testCases := []struct {
		version  string
		expected bool
	}{
		{"", false},
		{"dev", false},
		{"4.5.0", false},
		{"v4.5.0", false},
		{"4.5.1", false},
		{"4.6.0-alpha.5", false},
		{"4.6.0-rc.7", false},
		{"4.6.0-rc.8", true},
		{"4.6.0-rc.9", true},
		{"4.6.0", true},
		{"v4.6.0", true},
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
