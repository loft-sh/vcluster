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

	// set by findDockerDaemon inside ensureDockerDaemonOnce, so it needs no extra synchronization
	isPodmanBackend bool
)

// EnsureDockerDaemon makes sure the docker CLI can reach a container daemon before
// any docker command is run. If the default docker daemon is not reachable, it falls
// back to Podman's Docker-compatible API socket by setting DOCKER_HOST for this
// process, which every subsequent docker command inherits.
func EnsureDockerDaemon(ctx context.Context, log log.Logger) error {
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

	isPodmanBackend = true

	// use Warnf so the notice goes to stderr and doesn't corrupt commands that
	// print machine readable output (e.g. vcluster list --output json)
	log.Warnf("Docker daemon not found, using Podman's Docker-compatible API at %s", socketPath)
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

	args, err := podmanSocketCommandArgs(runtime.GOOS)
	if err != nil {
		return "", err
	}

	out, err := exec.CommandContext(ctx, "podman", args...).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("podman %s: %s: %w", strings.Join(args, " "), strings.TrimSpace(string(exitErr.Stderr)), err)
		}
		return "", fmt.Errorf("podman %s: %w", strings.Join(args, " "), err)
	}

	return parsePodmanSocketPath(string(out))
}

// podmanSocketCommandArgs returns the podman arguments that resolve the
// Docker-compatible API socket on the given platform. Split out of
// findPodmanSocket so the platform branches can be unit tested on any OS.
func podmanSocketCommandArgs(goos string) ([]string, error) {
	switch goos {
	case "windows":
		// podman machine on windows exposes a named pipe instead of a unix
		// socket, which the fallback doesn't handle
		return nil, fmt.Errorf("automatic podman fallback is not supported on windows, please set DOCKER_HOST to the podman machine's docker API endpoint manually")
	case "linux":
		// podman serves the API directly on the host
		return []string{"info", "--format", "{{.Host.RemoteSocket.Path}}"}, nil
	default:
		// podman runs inside a podman machine that forwards a socket to the host
		return []string{"machine", "inspect", "--format", "{{.ConnectionInfo.PodmanSocket.Path}}"}, nil
	}
}

// parsePodmanSocketPath extracts the socket path from the output of the command
// built by podmanSocketCommandArgs, e.g. "unix:///run/podman/podman.sock" or a
// bare path. Empty and "<nil>" outputs are what podman prints when no machine is
// running, so they map to the error the user can act on.
func parsePodmanSocketPath(out string) (string, error) {
	socketPath := strings.TrimPrefix(strings.TrimSpace(out), "unix://")
	if socketPath == "" || strings.Contains(socketPath, "<nil>") {
		return "", fmt.Errorf("podman did not return a socket path, is the podman machine running?")
	}

	return socketPath, nil
}

// hostNamespaceCommand builds the command that executes script in the namespaces of
// PID 1, which is the container daemon's host: the host itself on linux and the
// daemon's VM everywhere else. Callers decide how to read the output, since some need
// stdout only and others also want stderr.
func hostNamespaceCommand(ctx context.Context, script string) *exec.Cmd {
	return exec.CommandContext(ctx, "docker", "run", "-q", "--rm", "--privileged", "--pid=host", "alpine", "nsenter", "-t", "1", "-m", "-p", "-u", "-i", "-n", "sh", "-c", script)
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

// vmKernelModulesScript loads the kernel modules required for node networking
// inside the container daemon's VM. The sysctls are only set when a module had to
// be loaded, so machines that already have everything in place are never modified.
// Package level so the script handed to hostNamespaceCommand can be unit tested.
const vmKernelModulesScript = `
        loaded=""
        for mod in overlay bridge br_netfilter; do
            if ! grep -q "^$mod " /proc/modules; then
                if modprobe "$mod" 2>/dev/null; then
                    loaded="$loaded $mod"
                else
                    echo "could not load kernel module $mod"
                fi
            fi
        done
        if [ -n "$loaded" ]; then
            sysctl -qw net.bridge.bridge-nf-call-iptables=1 net.bridge.bridge-nf-call-ip6tables=1 net.ipv4.ip_forward=1 2>/dev/null || true
        fi
    `

func ensureVMKernelModules(ctx context.Context, log log.Logger) {
	// Docker Desktop preloads these modules, but other VM based runtimes like
	// podman machine, colima or rancher desktop don't necessarily do so. Enter the
	// VM's namespaces through a privileged container and load whatever is missing.
	out, err := hostNamespaceCommand(ctx, vmKernelModulesScript).CombinedOutput()
	if err != nil {
		log.Warnf("Could not ensure kernel modules inside the docker VM: %v: %s. If node join fails, load the overlay, bridge and br_netfilter kernel modules in the VM manually", err, strings.TrimSpace(string(out)))
		return
	}
	if output := strings.TrimSpace(string(out)); output != "" {
		log.Warnf("%s. If node join fails, load the overlay, bridge and br_netfilter kernel modules in the VM manually", output)
	}
}
