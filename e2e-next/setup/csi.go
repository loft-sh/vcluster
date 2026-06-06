package setup

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	csiDriverHostPathVersion   = "v1.17.0"
	externalSnapshotterVersion = "v8.4.0"
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
		snapshotterBase := "https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/" + externalSnapshotterVersion
		if err := kubectlApply(ctx, kubeContext,
			snapshotterBase+"/client/config/crd/snapshot.storage.k8s.io_volumesnapshotclasses.yaml",
			snapshotterBase+"/client/config/crd/snapshot.storage.k8s.io_volumesnapshotcontents.yaml",
			snapshotterBase+"/client/config/crd/snapshot.storage.k8s.io_volumesnapshots.yaml",
		); err != nil {
			return fmt.Errorf("install snapshot CRDs: %w", err)
		}

		// Install snapshot-controller
		if err := kubectlApplyKustomize(ctx, kubeContext,
			"https://github.com/kubernetes-csi/external-snapshotter/deploy/kubernetes/snapshot-controller?ref="+externalSnapshotterVersion); err != nil {
			return fmt.Errorf("install snapshot-controller: %w", err)
		}

		// Clone and deploy CSI hostpath driver
		tmpDir, err := os.MkdirTemp("", "csi-driver-host-path-*")
		if err != nil {
			return fmt.Errorf("create temp dir: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		cloneCmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "--branch", csiDriverHostPathVersion,
			"https://github.com/kubernetes-csi/csi-driver-host-path.git", tmpDir)
		if out, err := cloneCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("clone csi-driver-host-path@%s: %s: %w", csiDriverHostPathVersion, string(out), err)
		}

		// Remove the testing manifest (socat proxy) - it is only needed for
		// csi-sanity/csc testing and its StatefulSet is flaky on Kind.
		_ = os.Remove(filepath.Join(tmpDir, "deploy", "kubernetes-latest", "hostpath", "csi-hostpath-testing.yaml"))

		// Set the kubectl context before running deploy.sh so the CSI driver
		// is installed into the correct cluster even when multiple contexts exist.
		setCtxCmd := exec.CommandContext(ctx, "kubectl", "config", "use-context", kubeContext)
		if out, err := setCtxCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("set kubectl context to %s: %s: %w", kubeContext, string(out), err)
		}

		deployScript := tmpDir + "/deploy/kubernetes-latest/deploy.sh"
		deployCmd := exec.CommandContext(ctx, "bash", deployScript)
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
	return kubectlApplyWithOptions(ctx, kubeContext, nil, files...)
}

func kubectlApplyWithOptions(ctx context.Context, kubeContext string, options []string, files ...string) error {
	for _, f := range files {
		args := []string{"apply"}
		args = append(args, options...)
		args = append(args, "-f", f, "--context", kubeContext)
		cmd := exec.CommandContext(ctx, "kubectl", args...)
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
