package test_vind

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/blang/semver"
	"github.com/loft-sh/vcluster/pkg/constants"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var chartVersion string

func vclusterBinPath() string {
	return filepath.Join(os.Getenv("GOBIN"), "vcluster")
}

// getChartVersion returns the chart version to use for docker driver tests.
// Priority:
//  1. VCLUSTER_CHART_VERSION env var (manual override)
//  2. Latest git tag matching the CLI binary's major.minor version
//  3. Falls back to the binary's built-in version
func getChartVersion() string {
	if chartVersion != "" {
		return chartVersion
	}
	if v := os.Getenv("VCLUSTER_CHART_VERSION"); v != "" {
		chartVersion = v
		return chartVersion
	}
	if v, err := resolveLatestReleaseTag(); err == nil {
		chartVersion = v
		return chartVersion
	}
	// last resort: use the CLI binary's version directly (may fail for snapshot builds)
	if v, err := getVClusterCLIVersion(); err == nil {
		chartVersion = v
		return chartVersion
	}
	return ""
}

// getVClusterCLIVersion runs `vcluster --version` to determine the binary's version.
func getVClusterCLIVersion() (string, error) {
	out, err := exec.Command(vclusterBinPath(), "--version").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("vcluster --version: %w", err)
	}
	// output is like "vcluster version 0.33.0-next"
	parts := strings.Fields(strings.TrimSpace(string(out)))
	if len(parts) == 0 {
		return "", fmt.Errorf("empty version output")
	}
	return strings.TrimPrefix(parts[len(parts)-1], "v"), nil
}

// resolveLatestReleaseTag finds the latest git tag that shares the same
// major.minor version as the vcluster CLI binary. This allows snapshot builds
// to use a published release (e.g. 0.33.0-rc.1) whose pro image exists.
func resolveLatestReleaseTag() (string, error) {
	cliVersion, err := getVClusterCLIVersion()
	if err != nil {
		return "", err
	}

	// strip snapshot suffixes like "-next", "-dirty", etc. that aren't valid semver pre-release
	base := cliVersion
	if idx := strings.Index(base, "-"); idx != -1 {
		pre := base[idx+1:]
		// keep valid semver pre-release identifiers (alpha, beta, rc); strip others
		if !strings.HasPrefix(pre, "alpha") && !strings.HasPrefix(pre, "beta") && !strings.HasPrefix(pre, "rc") {
			base = base[:idx]
		}
	}

	sv, err := semver.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parse CLI version %q: %w", cliVersion, err)
	}

	// list git tags matching vX.Y.*
	prefix := fmt.Sprintf("v%d.%d.", sv.Major, sv.Minor)
	out, err := exec.Command("git", "tag", "-l", prefix+"*").Output()
	if err != nil {
		return "", fmt.Errorf("git tag: %w", err)
	}

	var versions []semver.Version
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		tag := strings.TrimSpace(strings.TrimPrefix(line, "v"))
		if tag == "" {
			continue
		}
		v, err := semver.Parse(tag)
		if err != nil {
			continue
		}
		versions = append(versions, v)
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("no tags found matching %s*", prefix)
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].GT(versions[j])
	})

	return versions[0].String(), nil
}

func runVCluster(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, vclusterBinPath(), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("vcluster %s failed: %w\noutput: %s", strings.Join(args, " "), err, string(out))
	}
	return string(out), nil
}

func dockerContainerRunning(ctx context.Context, containerName string) bool {
	out, err := exec.CommandContext(ctx, "docker", "inspect", "--type", "container", "--format", "{{.State.Running}}", containerName).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

func dockerContainerExists(ctx context.Context, containerName string) bool {
	err := exec.CommandContext(ctx, "docker", "inspect", "--type", "container", containerName).Run()
	return err == nil
}

func dockerNetworkExists(ctx context.Context, networkName string) bool {
	err := exec.CommandContext(ctx, "docker", "network", "inspect", networkName).Run()
	return err == nil
}

func dockerVolumesExist(ctx context.Context, prefix string) bool {
	out, err := exec.CommandContext(ctx, "docker", "volume", "ls", "--filter", "name=^"+prefix, "--format", "{{.Name}}").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}

func controlPlaneContainerName(vClusterName string) string {
	return constants.DockerControlPlanePrefix + vClusterName
}

func networkName(vClusterName string) string {
	return constants.DockerNetworkPrefix + vClusterName
}

func controlPlaneVolumePrefix(vClusterName string) string {
	return constants.DockerControlPlanePrefix + vClusterName + "."
}

func kubeClientFromKubeConfig(raw []byte) (kubernetes.Interface, error) {
	cfg, err := clientcmd.RESTConfigFromKubeConfig(raw)
	if err != nil {
		return nil, fmt.Errorf("build rest config from kubeconfig: %w", err)
	}
	return kubernetes.NewForConfig(cfg)
}
