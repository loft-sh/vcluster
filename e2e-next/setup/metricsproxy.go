package setup

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/loft-sh/vcluster/e2e-next/constants"
)

// MetricsServerPreSetup returns a PreSetupFunc that installs metrics-server on
// the host cluster using Helm. This is required for the metrics proxy
// integration tests (integrations.metricsServer.enabled: true), which need a
// real metrics-server to be running on the host before the vCluster starts.
//
// Idempotent - skips installation if metrics-server is already present.
func MetricsServerPreSetup() func(ctx context.Context) error {
	return func(ctx context.Context) error {
		kubeContext := "kind-" + constants.GetHostClusterName()

		// Check if metrics-server is already installed - skip if so.
		checkCmd := exec.CommandContext(ctx, "helm", "status", "metrics-server",
			"-n", "kube-system",
			"--kube-context", kubeContext)
		if err := checkCmd.Run(); err == nil {
			return nil // already installed
		}

		// Add the Helm repo (idempotent).
		addRepoCmd := exec.CommandContext(ctx, "helm", "repo", "add",
			"metrics-server", "https://kubernetes-sigs.github.io/metrics-server/")
		if out, err := addRepoCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("helm repo add metrics-server: %s: %w", string(out), err)
		}

		updateRepoCmd := exec.CommandContext(ctx, "helm", "repo", "update")
		if out, err := updateRepoCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("helm repo update: %s: %w", string(out), err)
		}

		// Install metrics-server with --kubelet-insecure-tls (required in Kind).
		installCmd := exec.CommandContext(ctx, "helm", "upgrade", "--install",
			"metrics-server", "metrics-server/metrics-server",
			"--set", "args={--kubelet-insecure-tls}",
			"--set", "containerPort=4443",
			"-n", "kube-system",
			"--kube-context", kubeContext,
			"--wait",
			"--timeout", "120s",
		)
		if out, err := installCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("helm upgrade --install metrics-server: %s: %w", string(out), err)
		}

		return nil
	}
}
