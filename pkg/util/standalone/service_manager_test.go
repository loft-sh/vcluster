package standalone

import (
	"errors"
	"os/exec"
	"runtime"
	"slices"
	"testing"

	"github.com/loft-sh/vcluster/pkg/constants"
)

func fakeRunner(catErr, stopErr, startErr error) systemctlRunner {
	return func(args ...string) error {
		switch args[0] {
		case "cat":
			return catErr
		case "stop":
			return stopErr
		case "start":
			return startErr
		}
		return nil
	}
}

func TestNewServiceManager_NonLinux(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("non-Linux path cannot be tested on Linux")
	}
	_, err := NewServiceManager()
	if err == nil {
		t.Fatal("expected error on non-Linux platform")
	}
}

func TestNewServiceManager_ServiceNotFound(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("systemd manager only supported on Linux")
	}
	_, err := newServiceManager(fakeRunner(errors.New("not found"), nil, nil))
	if err == nil {
		t.Fatal("expected error when service unit is missing")
	}
}

func TestNewServiceManager_ServiceExists(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("systemd manager only supported on Linux")
	}
	sm, err := newServiceManager(fakeRunner(nil, nil, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sm == nil {
		t.Fatal("expected non-nil ServiceManager")
	}
}

// TestServiceManager_StopStart verifies that Stop and Start delegate to the runner.
func TestServiceManager_StopStart(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("systemd manager only supported on Linux")
	}
	stopErr := errors.New("stop failed")
	startErr := errors.New("start failed")

	sm, err := newServiceManager(fakeRunner(nil, stopErr, startErr))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := sm.Stop(); !errors.Is(got, stopErr) {
		t.Errorf("Stop() = %v, want %v", got, stopErr)
	}
	if got := sm.Start(); !errors.Is(got, startErr) {
		t.Errorf("Start() = %v, want %v", got, startErr)
	}
}

// TestIsServiceActive verifies the probe keys on the unit's reported state, not
// the systemctl exit code: active/reloading/refreshing/activating/deactivating
// all mean a process may be live and count as active, while inactive/failed and
// an unanswerable probe count as not-active, and the observed state is reported
// back for the caller's error message. isServiceActive has no GOOS gate, so this
// runs anywhere.
func TestIsServiceActive(t *testing.T) {
	// is-active exits non-zero for activating/deactivating/inactive/failed but
	// zero for the live states, and always prints the state on stdout; mirror
	// that in the fake runner.
	stateRunner := func(state string) systemctlOutputRunner {
		return func(...string) (string, error) {
			if state == "active" || state == "reloading" || state == "refreshing" {
				return state + "\n", nil
			}
			return state + "\n", errors.New("exit status 3")
		}
	}

	for _, state := range []string{"active", "reloading", "refreshing", "activating", "deactivating"} {
		active, gotState := isServiceActive(stateRunner(state))
		if !active {
			t.Errorf("state %q: expected active", state)
		}
		if gotState != state {
			t.Errorf("state %q: reported state %q", state, gotState)
		}
	}
	for _, state := range []string{"inactive", "failed"} {
		active, gotState := isServiceActive(stateRunner(state))
		if active {
			t.Errorf("state %q: expected not-active", state)
		}
		if gotState != state {
			t.Errorf("state %q: reported state %q", state, gotState)
		}
	}

	// The probe passes "is-active <unit>" and reads stdout (no --quiet, so the
	// state is printed).
	var gotArgs []string
	isServiceActive(func(args ...string) (string, error) {
		gotArgs = args
		return "active\n", nil
	})
	wantArgs := []string{"is-active", constants.VClusterStandaloneSystemdServiceName}
	if !slices.Equal(gotArgs, wantArgs) {
		t.Fatalf("systemctl args = %v, want %v", gotArgs, wantArgs)
	}

	// A probe that cannot answer at all (no systemctl binary) must fail open.
	if active, _ := isServiceActive(func(...string) (string, error) {
		return "", &exec.Error{Name: "systemctl", Err: exec.ErrNotFound}
	}); active {
		t.Fatal("expected not-active when the probe cannot answer")
	}
}

// TestIsServiceActive_Exported exercises the exported IsServiceActive wiring:
// the GOOS gate and defaultSystemctlOutputRunner. No vcluster.service unit is
// installed in the test environment, so on Linux the probe reports the unit
// not-active and on other platforms the GOOS gate returns false. Either way the
// guard reports not-active — the fail-open behaviour the in-pod restore relies on.
func TestIsServiceActive_Exported(t *testing.T) {
	if active, _ := IsServiceActive(); active {
		t.Fatal("expected IsServiceActive() to report not-active with no vcluster.service installed")
	}
}

// TestNewServiceManager_ServiceDown verifies that NewServiceManager succeeds even
// when the service exists but is not active (the restore scenario).
func TestNewServiceManager_ServiceDown(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("systemd manager only supported on Linux")
	}
	// cat succeeds (unit exists), but the service is inactive — should not error.
	sm, err := newServiceManager(fakeRunner(nil, nil, nil))
	if err != nil {
		t.Fatalf("expected success for an inactive-but-existing service, got: %v", err)
	}
	if sm == nil {
		t.Fatal("expected non-nil ServiceManager")
	}
}
