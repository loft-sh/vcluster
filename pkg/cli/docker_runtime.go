package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/loft-sh/log"
)

var (
	ensureDockerDaemonOnce sync.Once
	errEnsureDockerDaemon  error
)

// ensureDockerDaemon makes sure the docker CLI can reach a container daemon before
// any docker command is run. If the default docker daemon is not reachable, it falls
// back to Podman's Docker-compatible API socket by setting DOCKER_HOST for this
// process, which every subsequent docker command inherits.
func ensureDockerDaemon(ctx context.Context, log log.Logger) error {
	ensureDockerDaemonOnce.Do(func() {
		errEnsureDockerDaemon = findDockerDaemon(ctx, log)
	})
	return errEnsureDockerDaemon
}

func findDockerDaemon(ctx context.Context, log log.Logger) error {
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("couldn't find the docker CLI, please make sure docker (or podman plus the docker CLI) is installed: %w", err)
	}

	// check if the docker daemon is already reachable
	pingErr := pingDockerDaemon(ctx)
	if pingErr == nil {
		return nil
	}

	// if the user explicitly configured a docker endpoint, don't second-guess it
	if dockerHost := os.Getenv("DOCKER_HOST"); dockerHost != "" {
		return fmt.Errorf("docker daemon at DOCKER_HOST=%s is not reachable: %w", dockerHost, pingErr)
	}

	// the docker daemon is not reachable, try to fall back to podman's
	// docker-compatible API socket
	socketPath, err := findPodmanSocket(ctx)
	if err != nil {
		log.Debugf("Podman fallback not available: %v", err)
		return fmt.Errorf("docker daemon is not reachable, please make sure docker is running (if you are using podman, make sure the podman machine is running): %w", pingErr)
	}

	err = os.Setenv("DOCKER_HOST", "unix://"+socketPath)
	if err != nil {
		return fmt.Errorf("set DOCKER_HOST: %w", err)
	}

	err = pingDockerDaemon(ctx)
	if err != nil {
		return fmt.Errorf("podman socket %s is not reachable via the docker CLI, please make sure the podman machine is running and rootful: %w", socketPath, err)
	}

	log.Infof("Docker daemon not found, using Podman's Docker-compatible API at %s", socketPath)
	return nil
}

func pingDockerDaemon(ctx context.Context) error {
	out, err := exec.CommandContext(ctx, "docker", "version", "--format", "{{.Server.Version}}").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}

	return nil
}

// findPodmanSocket returns the host path of Podman's Docker-compatible API socket.
func findPodmanSocket(ctx context.Context) (string, error) {
	if _, err := exec.LookPath("podman"); err != nil {
		return "", fmt.Errorf("podman not found: %w", err)
	}

	// on linux podman serves the API directly on the host, on other platforms it
	// runs inside a podman machine that forwards a socket to the host
	args := []string{"machine", "inspect", "--format", "{{.ConnectionInfo.PodmanSocket.Path}}"}
	if runtime.GOOS == "linux" {
		args = []string{"info", "--format", "{{.Host.RemoteSocket.Path}}"}
	}

	out, err := exec.CommandContext(ctx, "podman", args...).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("podman %s: %s: %w", strings.Join(args, " "), strings.TrimSpace(string(exitErr.Stderr)), err)
		}
		return "", fmt.Errorf("podman %s: %w", strings.Join(args, " "), err)
	}

	socketPath := strings.TrimPrefix(strings.TrimSpace(string(out)), "unix://")
	if socketPath == "" || strings.Contains(socketPath, "<nil>") {
		return "", fmt.Errorf("podman did not return a socket path, is the podman machine running?")
	}

	return socketPath, nil
}

// ensureKernelModules loads the kernel modules required for node join and pod
// networking (overlay, bridge, br_netfilter). On linux hosts the modules are loaded
// directly, on other platforms the container daemon runs inside a VM, so they are
// loaded there through a privileged container. Failures are logged as warnings
// because the modules might be built into the kernel.
func ensureKernelModules(ctx context.Context, log log.Logger) {
	if runtime.GOOS == "linux" {
		ensureHostKernelModules(ctx, log)
		return
	}

	ensureVMKernelModules(ctx, log)
}

func ensureHostKernelModules(ctx context.Context, log log.Logger) {
	// only run modprobe for modules not already loaded (check via /proc/modules, no sudo)
	loaded := map[string]bool{}
	if data, err := os.ReadFile("/proc/modules"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				loaded[fields[0]] = true
			}
		}
	}

	for _, mod := range []string{"overlay", "bridge", "br_netfilter"} {
		if loaded[mod] {
			continue
		}
		if err := exec.CommandContext(ctx, "modprobe", mod).Run(); err != nil {
			log.Warnf("Could not load kernel module %s: %v. If node join fails, run: sudo modprobe overlay && sudo modprobe bridge && sudo modprobe br_netfilter", mod, err)
		}
	}
}

func ensureVMKernelModules(ctx context.Context, log log.Logger) {
	// Docker Desktop preloads these modules, but other VM based runtimes like
	// podman machine, colima or rancher desktop don't necessarily do so. Enter the
	// VM's namespaces through a privileged container and load whatever is missing.
	cmdStr := `
        for mod in overlay bridge br_netfilter; do
            if ! grep -q "^$mod " /proc/modules; then
                modprobe "$mod" 2>/dev/null || echo "could not load kernel module $mod"
            fi
        done
        sysctl -qw net.bridge.bridge-nf-call-iptables=1 net.bridge.bridge-nf-call-ip6tables=1 net.ipv4.ip_forward=1 2>/dev/null || true
    `

	out, err := exec.CommandContext(ctx, "docker", "run", "-q", "--rm", "--privileged", "--pid=host", "alpine", "nsenter", "-t", "1", "-m", "-p", "-u", "-i", "-n", "sh", "-c", cmdStr).CombinedOutput()
	if err != nil {
		log.Warnf("Could not ensure kernel modules inside the docker VM: %v: %s. If node join fails, load the overlay, bridge and br_netfilter kernel modules in the VM manually", err, strings.TrimSpace(string(out)))
		return
	}
	if output := strings.TrimSpace(string(out)); output != "" {
		log.Warnf("%s. If node join fails, load the overlay, bridge and br_netfilter kernel modules in the VM manually", output)
	}
}
