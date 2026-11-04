package cli

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

// The podman fallback and VM kernel module paths only execute on macOS and
// windows hosts, which CI doesn't run. These tests cover the platform branches
// and output parsing directly so the darwin podman machine case - the platform
// loft-sh/vind#5 was reported from - doesn't ship blind.

func TestPodmanSocketCommandArgs(t *testing.T) {
	t.Run("darwin resolves the podman machine's forwarded socket", func(t *testing.T) {
		args, err := podmanSocketCommandArgs("darwin")
		assert.NilError(t, err)
		assert.DeepEqual(t, args, []string{"machine", "inspect", "--format", "{{.ConnectionInfo.PodmanSocket.Path}}"})
	})

	t.Run("linux resolves the host socket", func(t *testing.T) {
		args, err := podmanSocketCommandArgs("linux")
		assert.NilError(t, err)
		assert.DeepEqual(t, args, []string{"info", "--format", "{{.Host.RemoteSocket.Path}}"})
	})

	t.Run("windows is unsupported and says how to proceed", func(t *testing.T) {
		_, err := podmanSocketCommandArgs("windows")
		assert.ErrorContains(t, err, "not supported on windows")
		assert.ErrorContains(t, err, "DOCKER_HOST")
	})
}

func TestParsePodmanSocketPath(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected string
		errPart  string
	}{
		{
			name:     "unix scheme is stripped",
			output:   "unix:///run/podman/podman.sock\n",
			expected: "/run/podman/podman.sock",
		},
		{
			name:     "bare path from podman machine inspect",
			output:   "/var/folders/5w/T/podman/podman-machine-default-api.sock\n",
			expected: "/var/folders/5w/T/podman/podman-machine-default-api.sock",
		},
		{
			name:    "nil template result means no machine is running",
			output:  "<nil>\n",
			errPart: "is the podman machine running",
		},
		{
			name:    "empty output means no machine is running",
			output:  "\n",
			errPart: "is the podman machine running",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := parsePodmanSocketPath(tt.output)
			if tt.errPart != "" {
				assert.ErrorContains(t, err, tt.errPart)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, path, tt.expected)
		})
	}
}

func TestHostNamespaceCommand(t *testing.T) {
	cmd := hostNamespaceCommand(t.Context(), "echo probe")

	// the command must enter every namespace of PID 1 through a privileged
	// container sharing the daemon host's PID namespace
	assert.DeepEqual(t, cmd.Args, []string{
		"docker", "run", "-q", "--rm", "--privileged", "--pid=host",
		"alpine", "nsenter", "-t", "1", "-m", "-p", "-u", "-i", "-n",
		"sh", "-c", "echo probe",
	})
}

func TestVMKernelModulesScript(t *testing.T) {
	// node join needs all three modules, and the sysctls must stay gated on a
	// module having been loaded so already configured machines are not modified
	for _, mod := range []string{"overlay", "bridge", "br_netfilter"} {
		assert.Assert(t, strings.Contains(vmKernelModulesScript, mod), "script must handle module %s", mod)
	}
	assert.Assert(t, strings.Contains(vmKernelModulesScript, `if [ -n "$loaded" ]`), "sysctls must be gated on a module having been loaded")
	assert.Assert(t, strings.Contains(vmKernelModulesScript, "net.bridge.bridge-nf-call-iptables=1"), "bridge netfilter sysctl must be set for fresh VMs")
}
