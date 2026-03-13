package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/loft-sh/log"
	"github.com/loft-sh/log/hash"
	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/platform"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"
)

type deleteDocker struct {
	*flags.GlobalFlags
	*DeleteOptions

	log log.Logger
}

func DeleteDocker(ctx context.Context, platformClient platform.Client, options *DeleteOptions, globalFlags *flags.GlobalFlags, vClusterName string, log log.Logger) error {
	cmd := &deleteDocker{
		GlobalFlags:   globalFlags,
		DeleteOptions: options,
		log:           log,
	}

	return cmd.delete(ctx, platformClient, vClusterName)
}

func (cmd *deleteDocker) delete(ctx context.Context, platformClient platform.Client, vClusterName string) error {
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
	nodes, err := findDockerContainer(ctx, constants.DockerNodePrefix+vClusterName+".")
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

	// delete the load balancers
	loadBalancers, err := findDockerContainer(ctx, constants.DockerLoadBalancerPrefix+vClusterName+".")
	if err != nil {
		return fmt.Errorf("failed to find vCluster load balancers: %w", err)
	}
	for _, loadBalancer := range loadBalancers {
		cmd.log.Infof("Removing vCluster load balancer %s...", loadBalancer.Name)
		err = stopContainer(ctx, constants.DockerLoadBalancerPrefix+vClusterName+"."+loadBalancer.Name)
		if err != nil {
			return fmt.Errorf("failed to stop vCluster load balancer: %w", err)
		}
		err = removeContainer(ctx, constants.DockerLoadBalancerPrefix+vClusterName+"."+loadBalancer.Name)
		if err != nil {
			return fmt.Errorf("failed to remove vCluster load balancer: %w", err)
		}
	}

	// clean up loopback aliases before deleting the network (macOS only)
	cleanupLoopbackAliases(ctx, cmd.GlobalFlags, vClusterName, cmd.log)

	// delete the network
	err = deleteNetwork(ctx, vClusterName, cmd.log)
	if err != nil {
		cmd.log.Warnf("Failed to delete network: %v", err)
	}

	// delete from platform
	if platformClient != nil {
		cmd.log.Debugf("deleting vcluster in platform")
		err = cmd.deleteVClusterInPlatform(ctx, platformClient, vClusterName)
		if err != nil {
			return fmt.Errorf("deleting vcluster in platform failed: %w", err)
		}
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

func (cmd *deleteDocker) deleteVClusterInPlatform(ctx context.Context, platformClient platform.Client, vClusterName string) error {
	managementClient, err := platformClient.Management()
	if err != nil {
		cmd.log.Debugf("Error creating management client: %v", err)
		return nil
	}

	joinToken, err := ensureVClusterJoinToken(cmd.GlobalFlags, vClusterName, false)
	if err != nil {
		if os.IsNotExist(err) {
			cmd.log.Debugf("Join token file not found, nothing to delete")
			return nil
		}

		return fmt.Errorf("failed to ensure join token: %w", err)
	}

	virtualClusterInstances, err := managementClient.Loft().ManagementV1().VirtualClusterInstances(corev1.NamespaceAll).List(ctx, metav1.ListOptions{
		LabelSelector: platform.CreatedByCLILabel + "=true," + joinTokenLabel + "=" + hash.String(joinToken)[:32],
	})
	if err != nil {
		cmd.log.Debugf("Error retrieving vcluster instances: %v", err)
		return nil
	}

	for _, virtualClusterInstance := range virtualClusterInstances.Items {
		cmd.log.Infof("Delete virtual cluster instance %s/%s in platform", virtualClusterInstance.Namespace, virtualClusterInstance.Name)
		err = managementClient.Loft().ManagementV1().VirtualClusterInstances(virtualClusterInstance.Namespace).Delete(ctx, virtualClusterInstance.Name, metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("delete virtual cluster instance %s/%s: %w", virtualClusterInstance.Namespace, virtualClusterInstance.Name, err)
		}
	}

	return nil
}

// cleanupLoopbackAliases removes loopback aliases that were added during vCluster creation
// on macOS. This must be called before the Docker network is deleted, since the network
// is needed to recompute which IPs were aliased.
func cleanupLoopbackAliases(ctx context.Context, globalFlags *flags.GlobalFlags, vClusterName string, log log.Logger) {
	if runtime.GOOS != "darwin" {
		return
	}

	// read the saved vcluster.yaml to check if load balancer aliases were created
	vClusterYAMLPath := filepath.Join(filepath.Dir(globalFlags.Config), "docker", "vclusters", vClusterName, "vcluster.yaml")
	data, err := os.ReadFile(vClusterYAMLPath)
	if err != nil {
		log.Debugf("Could not read vcluster config for alias cleanup: %v", err)
		return
	}

	vClusterConfig := &config.Config{}
	if err := yaml.Unmarshal(data, vClusterConfig); err != nil {
		log.Debugf("Could not parse vcluster config for alias cleanup: %v", err)
		return
	}

	if !vClusterConfig.Experimental.Docker.LoadBalancer.Enabled {
		return
	}

	// derive network name
	networkName := getNetworkName(vClusterName)
	if vClusterConfig.Experimental.Docker.Network != "" {
		networkName = vClusterConfig.Experimental.Docker.Network
	}

	// compute the same IPs that were aliased during creation
	ips, err := findTailIPs(ctx, networkName, 10)
	if err != nil {
		log.Debugf("Could not compute loopback alias IPs for cleanup: %v", err)
		return
	}

	useSudo := vClusterConfig.Experimental.Docker.LoadBalancer.UseSudo
	for _, ip := range ips {
		var cmd *exec.Cmd
		if useSudo {
			cmd = exec.CommandContext(ctx, "sudo", "-n", "ifconfig", "lo0", "-alias", ip)
		} else {
			cmd = exec.CommandContext(ctx, "ifconfig", "lo0", "-alias", ip)
		}
		if out, err := cmd.CombinedOutput(); err != nil {
			log.Debugf("Failed to remove loopback alias %s: %s: %v", ip, string(out), err)
		}
	}

	log.Debugf("Cleaned up loopback aliases for vCluster %s", vClusterName)
}
