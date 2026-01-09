package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"k8s.io/client-go/tools/clientcmd"
)

type deleteDocker struct {
	*flags.GlobalFlags
	*DeleteOptions

	log log.Logger
}

func DeleteDocker(ctx context.Context, options *DeleteOptions, globalFlags *flags.GlobalFlags, vClusterName string, log log.Logger) error {
	cmd := &deleteDocker{
		GlobalFlags:   globalFlags,
		DeleteOptions: options,
		log:           log,
	}

	return cmd.delete(ctx, vClusterName)
}

func (cmd *deleteDocker) delete(ctx context.Context, vClusterName string) error {
	containerName := getControlPlaneContainerName(vClusterName)

	// check if container exists
	exists, err := containerExists(ctx, containerName)
	if err != nil {
		return fmt.Errorf("failed to check if container exists: %w", err)
	}

	if !exists {
		if cmd.IgnoreNotFound {
			cmd.log.Infof("vCluster container %s not found, nothing to delete", containerName)
			return nil
		}
		return fmt.Errorf("vCluster container %s not found", containerName)
	}

	// stop & delete the container
	cmd.log.Infof("Removing vCluster container %s...", containerName)
	err = stopContainer(ctx, containerName)
	if err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}
	err = removeContainer(ctx, containerName)
	if err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}
	for volumeName := range containerVolumes {
		err = removeVolume(ctx, getControlPlaneVolumeName(vClusterName, volumeName))
		if err != nil {
			cmd.log.Warnf("Failed to delete volume %s: %v", getControlPlaneVolumeName(vClusterName, volumeName), err)
		}
	}

	// delete the nodes
	nodes, err := findDockerVClusterNodes(ctx, vClusterName)
	if err != nil {
		return fmt.Errorf("failed to find vCluster nodes: %w", err)
	}
	for _, node := range nodes {
		cmd.log.Infof("Removing vCluster node %s...", node.Name)
		err = stopContainer(ctx, getWorkerContainerName(vClusterName, node.Name))
		if err != nil {
			return fmt.Errorf("failed to stop vCluster node: %w", err)
		}
		err = removeContainer(ctx, getWorkerContainerName(vClusterName, node.Name))
		if err != nil {
			return fmt.Errorf("failed to remove vCluster node: %w", err)
		}
		for volumeName := range containerVolumes {
			err = removeVolume(ctx, getWorkerVolumeName(vClusterName, node.Name, volumeName))
			if err != nil {
				cmd.log.Warnf("Failed to delete volume %s: %v", getWorkerVolumeName(vClusterName, node.Name, volumeName), err)
			}
		}
	}

	// delete the network
	err = deleteNetwork(ctx, vClusterName, cmd.log)
	if err != nil {
		cmd.log.Warnf("Failed to delete network: %v", err)
	}

	// delete context from kubeconfig if requested
	if cmd.DeleteContext {
		err = cmd.deleteKubeContext(vClusterName)
		if err != nil {
			cmd.log.Warnf("Failed to delete kube context: %v", err)
		}
	}

	// delete the vCluster directory
	err = os.RemoveAll(filepath.Join(filepath.Dir(cmd.GlobalFlags.Config), "docker", "vclusters", vClusterName))
	if err != nil {
		cmd.log.Warnf("Failed to delete vCluster directory: %v", err)
	}

	cmd.log.Donef("Successfully deleted virtual cluster %s", vClusterName)
	return nil
}

func containerExists(ctx context.Context, containerName string) (bool, error) {
	args := []string{"inspect", "--type", "container", containerName}
	err := exec.CommandContext(ctx, "docker", args...).Run()
	if err != nil {
		// container doesn't exist or docker command failed
		return false, nil
	}
	return true, nil
}

func stopContainer(ctx context.Context, containerName string) error {
	args := []string{"stop", "--timeout=1", containerName}
	output, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker stop failed: %w, output: %s", err, string(output))
	}
	return nil
}

func removeVolume(ctx context.Context, volumeName string) error {
	args := []string{"volume", "rm", volumeName}
	output, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker volume rm failed: %w, output: %s", err, string(output))
	}
	return nil
}

func removeContainer(ctx context.Context, containerName string) error {
	args := []string{"rm", containerName}
	output, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker rm failed: %w, output: %s", err, string(output))
	}
	return nil
}

func (cmd *deleteDocker) deleteKubeContext(vClusterName string) error {
	// The context name follows the pattern from connect_docker.go
	kubeContextName := "vcluster-docker_" + vClusterName

	// Load the kubeconfig
	kubeConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).RawConfig()
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	// Check if context exists
	if _, ok := kubeConfig.Contexts[kubeContextName]; !ok {
		cmd.log.Debugf("Kube context %s not found, nothing to delete", kubeContextName)
		return nil
	}

	// Delete context using the shared deleteContext function
	err = deleteContext(&kubeConfig, kubeContextName, "")
	if err != nil {
		return fmt.Errorf("failed to delete context: %w", err)
	}

	cmd.log.Infof("Deleted kube context %s", kubeContextName)
	return nil
}
