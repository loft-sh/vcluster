package snapshot

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/loft-sh/vcluster/pkg/constants"
)

// stubServiceActive stubs the standalone service probe for the duration of a test.
func stubServiceActive(t *testing.T, active bool) {
	t.Helper()
	orig := isServiceActive
	isServiceActive = func() (bool, string) {
		if active {
			return true, "active"
		}
		return false, "inactive"
	}
	t.Cleanup(func() { isServiceActive = orig })
}

func executeRestore(t *testing.T) error {
	t.Helper()
	cmd := NewRestoreCommand()
	cmd.SetArgs([]string{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	return cmd.ExecuteContext(context.Background())
}

// TestRestore_RefusedWhileServiceActive verifies the guard wiring: with the
// standalone unit active, RunE must refuse before touching the backing store,
// naming the unit and both remediations.
func TestRestore_RefusedWhileServiceActive(t *testing.T) {
	stubServiceActive(t, true)

	err := executeRestore(t)
	if err == nil {
		t.Fatal("expected restore to be refused while the service is active")
	}
	for _, want := range []string{
		constants.VClusterStandaloneSystemdServiceName + ".service",
		"vcluster restore --standalone",
		"systemctl stop " + constants.VClusterStandaloneSystemdServiceName,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal error %q does not mention %q", err, want)
		}
	}
}

// TestRestore_ProceedsWhileServiceInactive verifies that a not-active probe
// lets RunE continue past the guard into the restore path. VCLUSTER_STANDALONE
// is forced off so config loading takes the in-cluster path and does not depend
// on host systemd state; VCLUSTER_NAME is forced empty so the first step past
// the guard — config.LoadConfig("") — fails on the empty name (a runner that
// exports VCLUSTER_NAME would otherwise mask it). Asserting that exact error
// proves execution reached past the guard rather than being stopped by it.
func TestRestore_ProceedsWhileServiceInactive(t *testing.T) {
	stubServiceActive(t, false)
	t.Setenv(constants.VClusterStandaloneEnvVar, "false")
	t.Setenv("VCLUSTER_NAME", "")

	err := executeRestore(t)
	if err == nil {
		t.Fatal("expected the restore path to fail in the test environment")
	}
	if strings.Contains(err.Error(), "refusing to restore") {
		t.Fatalf("guard engaged despite inactive service: %v", err)
	}
	if !strings.Contains(err.Error(), "empty vCluster name") {
		t.Fatalf("expected the post-guard config load to fail on the empty vCluster name, got: %v", err)
	}
}
