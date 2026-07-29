package standalone

import (
	"errors"
	"runtime"
	"testing"
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
