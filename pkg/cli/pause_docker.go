package cli

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/constants"
)

func PauseDocker(ctx context.Context, globalFlags *flags.GlobalFlags, vClusterName string, log log.Logger) error {
	containerName := getControlPlaneContainerName(vClusterName)

	// check if container exists
	exists, running, err := checkDockerContainerState(ctx, containerName)
	if err != nil {
		return fmt.Errorf("failed to check container state: %w", err)
	}

	if !exists {
		return fmt.Errorf("vCluster container %s not found", containerName)
	}

	if !running {
		log.Infof("vCluster %s is already paused (container stopped)", vClusterName)
		return nil
	}

	// stop the container
	log.Infof("Pausing vCluster %s...", vClusterName)
	err = stopDockerContainer(ctx, containerName)
	if err != nil {
		return fmt.Errorf("failed to pause vCluster: %w", err)
	}

	// stop the nodes
	nodes, err := findDockerContainer(ctx, constants.DockerNodePrefix+vClusterName+".")
	if err != nil {
		return fmt.Errorf("failed to find vCluster nodes: %w", err)
	}
	for _, node := range nodes {
		log.Infof("Stopping node %s from vCluster %s...", node.Name, vClusterName)
		err = stopDockerContainer(ctx, getWorkerContainerName(vClusterName, node.Name))
		if err != nil {
			return fmt.Errorf("failed to stop vCluster node: %w", err)
		}
	}

	// stop the load balancers
	loadBalancers, err := findDockerContainer(ctx, constants.DockerLoadBalancerPrefix+vClusterName+".")
	if err != nil {
		return fmt.Errorf("failed to find vCluster load balancers: %w", err)
	}
	for _, loadBalancer := range loadBalancers {
		log.Infof("Stopping load balancer %s from vCluster %s...", loadBalancer.Name, vClusterName)
		err = stopDockerContainer(ctx, constants.DockerLoadBalancerPrefix+vClusterName+"."+loadBalancer.Name)
		if err != nil {
			return fmt.Errorf("failed to stop vCluster load balancer: %w", err)
		}
	}

	log.Donef("Successfully paused vCluster %s", vClusterName)
	return nil
}

func checkDockerContainerState(ctx context.Context, containerName string) (exists bool, running bool, err error) {
	args := []string{"inspect", "--type", "container", "--format", "{{.State.Running}}", containerName}
	out, err := exec.CommandContext(ctx, "docker", args...).Output()
	if err != nil {
		// container doesn't exist
		return false, false, nil
	}

	// check if running
	stateStr := string(out)
	if stateStr == "true\n" || stateStr == "true" {
		return true, true, nil
	}

	return true, false, nil
}

func stopDockerContainer(ctx context.Context, containerName string) error {
	args := []string{"stop", containerName}
	output, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker stop failed: %w, output: %s", err, string(output))
	}
	return nil
}
