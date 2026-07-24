package standalone

import (
	"errors"
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

// TestIsServiceActive verifies the is-active probe: an affirmative systemctl
// exit reports active, any failure (stopped unit, unknown unit, no systemd)
// reports not-active. isServiceActive has no GOOS gate, so this runs anywhere.
func TestIsServiceActive(t *testing.T) {
	var gotArgs []string
	active := isServiceActive(func(args ...string) error {
		gotArgs = args
		return nil
	})
	if !active {
		t.Fatal("expected active when systemctl is-active succeeds")
	}
	wantArgs := []string{"is-active", "--quiet", constants.VClusterStandaloneSystemdServiceName}
	if !slices.Equal(gotArgs, wantArgs) {
		t.Fatalf("systemctl args = %v, want %v", gotArgs, wantArgs)
	}

	if isServiceActive(func(...string) error { return errors.New("inactive") }) {
		t.Fatal("expected not-active when systemctl is-active fails")
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
