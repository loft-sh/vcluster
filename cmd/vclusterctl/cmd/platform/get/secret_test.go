package get

import (
	"errors"
	"testing"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform/set"
)

// Shared secrets were removed in vCluster Platform 5.0, so without a project there is nothing
// to read. Run must fail before it tries to reach the platform.
func TestRunRequiresProject(t *testing.T) {
	cmd := &SecretCmd{}
	err := cmd.Run(t.Context(), []string{"my-secret.key"})
	if !errors.Is(err, set.ErrProjectRequired) {
		t.Fatalf("expected ErrProjectRequired, got %v", err)
	}
}
