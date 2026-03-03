package standalone

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"k8s.io/klog/v2"
)

// Reset uninstalls a vCluster Standalone node.
func Reset(ctx context.Context, config string) error {
	if err := preflightChecks(); err != nil {
		return err
	}

	configPath, err := findConfig(config, "/etc/vcluster/vcluster.yaml")
	if err != nil {
		return err
	}

	dataDir, err := lookupDataDir(configPath)
	if err != nil {
		return fmt.Errorf("failed to get data directory: %w", err)
	}

	if err := stopAndDisableService(ctx); err != nil {
		return err
	}

	if err := killAnyRenamingProcesses(ctx, dataDir); err != nil {
		return err
	}

	if err := deleteService(ctx); err != nil {
		return err
	}

	if err := deleteData(ctx, dataDir); err != nil {
		return err
	}

	return logUninstallCompleteMessage(ctx)
}

// deleteService removes the vcluster systemd service.
func deleteService(ctx context.Context) error {
	log := klog.FromContext(ctx)

	if err := os.Remove("/etc/systemd/system/vcluster.service"); err != nil {
		log.Error(err, "Failed to delete vcluster.service")
	}

	if err := exec.CommandContext(ctx, "systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to systemctl daemon-reload: %w", err)
	}

	return nil
}

// killAnyRenamingProcesses best-effort kill of any remaining vcluster processes.
func killAnyRenamingProcesses(ctx context.Context, dataDir string) error {
	log := klog.FromContext(ctx)

	if err := exec.CommandContext(ctx, "pkill", "-f", filepath.Join(dataDir, "bin", "vcluster")).Run(); err != nil {
		log.Error(err, "Failed to kill vcluster processes")
	}

	if err := exec.CommandContext(ctx, "pkill", "-x", "vcluster").Run(); err != nil {
		log.Error(err, "Failed to kill vcluster processes")
	}

	return nil
}

// deleteData removes vcluster and Kubernetes state that can cause reinstall issues.
func deleteData(ctx context.Context, dataDir string) error {
	log := klog.FromContext(ctx)

	dirs := []string{
		dataDir,
		"/etc/kubernetes",
		"/var/lib/kubelet",
		"/var/run/kubernetes",
		"/var/lib/etcd",
	}

	for _, dir := range dirs {
		log.Info("Deleting directory", "dir", dir)
		if err := os.RemoveAll(dir); err != nil {
			log.Error(err, "Failed to remove dir", "dir", dir)
		}
	}

	return nil
}

// logUninstallCompleteMessage logs a success message.
func logUninstallCompleteMessage(ctx context.Context) error {
	log := klog.FromContext(ctx)
	log.Info("Reset process complete")
	return nil
}
