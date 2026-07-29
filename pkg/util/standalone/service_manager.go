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

func newServiceManager(run systemctlRunner) (ServiceManager, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("systemd manager is only supported on Linux (current OS: %s)", runtime.GOOS)
	}

	if err := run("cat", constants.VClusterStandaloneSystemdServiceName); err != nil {
		return nil, fmt.Errorf("standalone vCluster service %q not found on this host", constants.VClusterStandaloneSystemdServiceName)
	}

	return &SystemdServiceManager{name: constants.VClusterStandaloneSystemdServiceName, run: run}, nil
}
