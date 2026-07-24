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
	isServiceActive = func() bool { return active }
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
// lets RunE continue past the guard into the restore path.
func TestRestore_ProceedsWhileServiceInactive(t *testing.T) {
	stubServiceActive(t, false)
	// Force the in-cluster config path so the result does not depend on host
	// systemd state, and poison the storage options env so the command can
	// never reach an actual restore in this environment.
	t.Setenv(constants.VClusterStandaloneEnvVar, "false")
	t.Setenv(constants.VClusterStorageOptionsEnv, "not-base64!")

	err := executeRestore(t)
	if err == nil {
		t.Fatal("expected the restore path to fail in the test environment")
	}
	if strings.Contains(err.Error(), "refusing to restore") {
		t.Fatalf("guard engaged despite inactive service: %v", err)
	}
}
