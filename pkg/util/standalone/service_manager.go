package standalone

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/loft-sh/vcluster/pkg/constants"
)

// ServiceManager abstracts stopping and starting the standalone vCluster process.
type ServiceManager interface {
	Stop() error
	Start() error
}

// systemctlRunner is a function that runs a systemctl subcommand and returns any error.
type systemctlRunner func(args ...string) error

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

// NewServiceManager returns a systemd-based service manager when on Linux with systemd
// available. Returns an error on other platforms or when the service unit is not found.
func NewServiceManager() (ServiceManager, error) {
	return newServiceManager(defaultSystemctlRunner)
}

// IsServiceActive reports whether the standalone vCluster systemd unit is
// currently active on this host. It reports false wherever systemd cannot
// answer affirmatively (non-Linux, no systemctl binary, systemd not running,
// unit stopped or not installed), so callers can use it as a guard that only
// engages on a standalone host with a running control plane.
func IsServiceActive() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	return isServiceActive(defaultSystemctlRunner)
}

func isServiceActive(run systemctlRunner) bool {
	// "systemctl is-active" exits zero only when the unit is active; a missing
	// systemctl binary, an unreachable systemd, a stopped unit, and an unknown
	// unit all exit non-zero and therefore report not-active.
	return run("is-active", "--quiet", constants.VClusterStandaloneSystemdServiceName) == nil
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
