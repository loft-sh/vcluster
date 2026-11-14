package platform

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/cli/flags"
)

func TestPlatformLoginKeepsDeprecatedDockerLoginFlag(t *testing.T) {
	cmd := NewCobraLoginCmd(&flags.GlobalFlags{})
	if err := cmd.ParseFlags([]string{"--docker-login=false"}); err != nil {
		t.Fatalf("--docker-login=false no longer parses: %v", err)
	}
	if f := cmd.Flags().Lookup("docker-login"); f.Deprecated == "" || !f.Hidden {
		t.Errorf("--docker-login: deprecated %q, hidden %v; want deprecated and hidden", f.Deprecated, f.Hidden)
	}
}
