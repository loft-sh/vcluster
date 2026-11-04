package standalone

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/loft-sh/vcluster/pkg/constants"
	"k8s.io/klog/v2"
)

// ServiceManager abstracts stopping and starting the standalone vCluster process.
type ServiceManager interface {
	Stop() error
	Start() error
}

// systemctlRunner is a function that runs a systemctl subcommand and returns any error.
type systemctlRunner func(args ...string) error

// systemctlOutputRunner runs a systemctl subcommand and returns its stdout and any error.
type systemctlOutputRunner func(args ...string) (string, error)

type SystemdServiceManager struct {
	name string
	run  systemctlRunner
}

func (s *SystemdServiceManager) Stop() error {
	return s.run("stop", s.name)
}

func (s *SystemdServiceManager) Start() error {
	return s.run("start", s.name)
}

func defaultSystemctlRunner(args ...string) error {
	return exec.Command("systemctl", args...).Run()
}

func defaultSystemctlOutputRunner(args ...string) (string, error) {
	out, err := exec.Command("systemctl", args...).Output()
	return string(out), err
}

// NewServiceManager returns a systemd-based service manager when on Linux with systemd
// available. Returns an error on other platforms or when the service unit is not found.
func NewServiceManager() (ServiceManager, error) {
	return newServiceManager(defaultSystemctlRunner)
}

// IsServiceActive reports whether the standalone vCluster systemd unit is
// currently active on this host, along with the unit's observed state (the
// systemctl is-active output) so callers can name it in an error. It treats
// activating, deactivating, and refreshing as active as well — in all of them a
// server process may still hold the backing store open — so callers can use it
// as a guard that refuses to run while a control plane is up. It reports false
// where systemd cannot answer (non-Linux, no systemctl binary, systemd
// unreachable, unit stopped/failed/not installed), so the in-pod restore path,
// which has no systemd, proceeds.
func IsServiceActive() (active bool, state string) {
	if runtime.GOOS != "linux" {
		return false, ""
	}
	return isServiceActive(defaultSystemctlOutputRunner)
}

func isServiceActive(run systemctlOutputRunner) (active bool, state string) {
	// Key the decision on the unit's reported state, not the systemctl exit
	// code: "systemctl is-active" exits non-zero for activating and
	// deactivating as well as for inactive/failed, but in the first two a
	// process may still be live against the store (a Type=notify daemon before
	// READY=1, or the RestartSec dwell of a crash-looping unit). reloading and
	// refreshing (a Type=notify-reload unit mid-reload) exit zero and are live
	// too. is-active prints the state on stdout in every case, so we read that.
	out, err := run("is-active", constants.VClusterStandaloneSystemdServiceName)
	state = strings.TrimSpace(out)
	switch state {
	case "active", "reloading", "refreshing", "activating", "deactivating":
		return true, state
	case "inactive", "failed":
		return false, state
	}
	// No usable state. Expected where systemd is absent (the in-pod restore
	// path: no systemctl binary), where we fail open so the restore proceeds.
	// Log only when systemctl exists but still could not answer (e.g. systemd
	// unreachable), so an operator can tell that from a confirmed-inactive unit.
	if err != nil && !errors.Is(err, exec.ErrNotFound) {
		klog.Warningf("standalone restore guard: could not determine %s.service state (%v); proceeding as not-active",
			constants.VClusterStandaloneSystemdServiceName, err)
	}
	return false, state
}

func newServiceManager(run systemctlRunner) (ServiceManager, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("systemd manager is only supported on Linux (current OS: %s)", runtime.GOOS)
	}

	if err := run("cat", constants.VClusterStandaloneSystemdServiceName); err != nil {
		return nil, fmt.Errorf("standalone vCluster service %q not found on this host", constants.VClusterStandaloneSystemdServiceName)
	}

	return &SystemdServiceManager{name: constants.VClusterStandaloneSystemdServiceName, run: run}, nil
}
