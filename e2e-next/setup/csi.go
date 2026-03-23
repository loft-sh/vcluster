package setup

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// InstallCSIHostpath returns a PreSetupFunc that installs the CSI hostpath
// driver, snapshot CRDs, snapshot-controller, StorageClass, and
// VolumeSnapshotClass on the host cluster. This is required for snapshot tests
// that use PVCs with StorageClassName "csi-hostpath-sc".
//
// Idempotent - skips components that already exist.
func InstallCSIHostpath(kubeContext string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		// Check if csi-hostpath-sc already exists - skip everything if so
		checkCmd := exec.CommandContext(ctx, "kubectl", "get", "sc", "csi-hostpath-sc", "--context", kubeContext)
		if err := checkCmd.Run(); err == nil {
			return nil // already installed
		}

		// Install snapshot CRDs
		if err := kubectlApply(ctx, kubeContext,
			"https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/master/client/config/crd/snapshot.storage.k8s.io_volumesnapshotclasses.yaml",
			"https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/master/client/config/crd/snapshot.storage.k8s.io_volumesnapshotcontents.yaml",
			"https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/master/client/config/crd/snapshot.storage.k8s.io_volumesnapshots.yaml",
		); err != nil {
			return fmt.Errorf("install snapshot CRDs: %w", err)
		}

		// Install snapshot-controller
		if err := kubectlApplyKustomize(ctx, kubeContext,
			"https://github.com/kubernetes-csi/external-snapshotter/deploy/kubernetes/snapshot-controller"); err != nil {
			return fmt.Errorf("install snapshot-controller: %w", err)
		}

		// Clone and deploy CSI hostpath driver
		tmpDir, err := os.MkdirTemp("", "csi-driver-host-path-*")
		if err != nil {
			return fmt.Errorf("create temp dir: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		cloneCmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1",
			"https://github.com/kubernetes-csi/csi-driver-host-path.git", tmpDir)
		if out, err := cloneCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("clone csi-driver-host-path: %s: %w", string(out), err)
		}

		deployScript := tmpDir + "/deploy/kubernetes-latest/deploy.sh"
		deployCmd := exec.CommandContext(ctx, "bash", deployScript)
		deployCmd.Env = append(os.Environ(), "KUBECONFIG="+os.Getenv("KUBECONFIG"))
		if out, err := deployCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("deploy CSI hostpath driver: %s: %w", string(out), err)
		}

		// Create StorageClass and VolumeSnapshotClass
		if err := kubectlApply(ctx, kubeContext,
			tmpDir+"/examples/csi-storageclass.yaml",
			tmpDir+"/examples/csi-volumesnapshotclass.yaml",
		); err != nil {
			return fmt.Errorf("create StorageClass/VolumeSnapshotClass: %w", err)
		}

		// Annotate VolumeSnapshotClass as default
		annotateCmd := exec.CommandContext(ctx, "kubectl", "annotate", "volumesnapshotclass",
			"csi-hostpath-snapclass", "snapshot.storage.kubernetes.io/is-default-class=true",
			"--overwrite", "--context", kubeContext)
		if out, err := annotateCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("annotate VolumeSnapshotClass: %s: %w", string(out), err)
		}

		// Wait for snapshot-controller
		waitCmd := exec.CommandContext(ctx, "kubectl", "wait", "--for=condition=ready", "pod",
			"-l", "app.kubernetes.io/name=snapshot-controller",
			"-n", "kube-system", "--timeout=120s", "--context", kubeContext)
		if out, err := waitCmd.CombinedOutput(); err != nil {
			// Try alternative label used by some versions
			waitCmd2 := exec.CommandContext(ctx, "kubectl", "wait", "--for=condition=ready", "pod",
				"-l", "app=snapshot-controller",
				"-n", "kube-system", "--timeout=120s", "--context", kubeContext)
			if out2, err2 := waitCmd2.CombinedOutput(); err2 != nil {
				return fmt.Errorf("wait for snapshot-controller: %s / %s: %w", string(out), string(out2), err)
			}
		}

		return nil
	}
}

func kubectlApply(ctx context.Context, kubeContext string, files ...string) error {
	for _, f := range files {
		cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", f, "--context", kubeContext)
		if out, err := cmd.CombinedOutput(); err != nil {
			if !strings.Contains(string(out), "already exists") {
				return fmt.Errorf("kubectl apply -f %s: %s: %w", f, string(out), err)
			}
		}
	}
	return nil
}

func kubectlApplyKustomize(ctx context.Context, kubeContext, url string) error {
	// Pipe kustomize output to kubectl apply
	kustomizeCmd := exec.CommandContext(ctx, "kubectl", "kustomize", url)
	kustomizeOut, err := kustomizeCmd.Output()
	if err != nil {
		return fmt.Errorf("kubectl kustomize %s: %w", url, err)
	}

	applyCmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", "-", "--context", kubeContext)
	applyCmd.Stdin = strings.NewReader(string(kustomizeOut))
	if out, err := applyCmd.CombinedOutput(); err != nil {
		if !strings.Contains(string(out), "already exists") {
			return fmt.Errorf("kubectl apply kustomize output: %s: %w", string(out), err)
		}
	}
	return nil
}
