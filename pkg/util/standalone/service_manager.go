package standalone

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/loft-sh/vcluster/pkg/constants"
)

// serviceManager abstracts stopping and starting the standalone vCluster process.
type ServiceManager interface {
	Stop() error
	Start() error
}

type SystemdServiceManager struct {
	name string
}

func (s *SystemdServiceManager) Stop() error {
	return exec.Command("systemctl", "stop", s.name).Run()
}

func (s *SystemdServiceManager) Start() error {
	return exec.Command("systemctl", "start", s.name).Run()
}

// newServiceManager returns a systemd-based service manager when on Linux with systemd
// available. Returns an error on other platforms or when the service is not running.
func NewServiceManager() (ServiceManager, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("systemd manager is only supported on Linux (current OS: %s)", runtime.GOOS)
	}

	out, err := exec.Command("systemctl", "is-active", constants.VClusterStandaloneSystemdServiceName).Output()
	if err != nil {
		return nil, fmt.Errorf("standalone vCluster service %q is not active on this host", constants.VClusterStandaloneSystemdServiceName)
	}
	if strings.TrimSpace(string(out)) != "active" {
		return nil, fmt.Errorf("standalone vCluster service %q is not active (state: %s)", constants.VClusterStandaloneSystemdServiceName, strings.TrimSpace(string(out)))
	}

	return &SystemdServiceManager{name: constants.VClusterStandaloneSystemdServiceName}, nil
}
