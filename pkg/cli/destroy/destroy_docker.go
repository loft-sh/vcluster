package destroy

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/loft-sh/log"
)

const (
	// PlatformContainerName is the name of the docker container for vCluster platform
	PlatformContainerName = "vcluster-platform"
	// PlatformVolumeName is the name of the docker volume for vCluster platform
	PlatformVolumeName = "vcluster-platform"
)

// ErrDockerPlatformNotFound is returned when no docker platform installation is found
var ErrDockerPlatformNotFound = fmt.Errorf("no vCluster platform docker installation found (no container or volume)")

// DestroyDocker stops and removes the vCluster platform docker container and volume.
// If ignoreNotFound is false and no container or volume is found, it returns ErrDockerPlatformNotFound.
func DestroyDocker(ctx context.Context, ignoreNotFound bool, log log.Logger) error {
	// check if docker is available
	_, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("docker is not installed or not available in PATH: %w", err)
	}

	// check if docker daemon is running
	output, err := exec.CommandContext(ctx, "docker", "ps").CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker daemon is not running or not accessible: %s", string(output))
	}

	// track if we found anything to destroy
	foundContainer := false
	foundVolume := false

	// check if container exists
	containerID, err := findContainer(ctx, PlatformContainerName)
	if err != nil {
		return fmt.Errorf("failed to find container: %w", err)
	}

	if containerID != "" {
		foundContainer = true
		// stop the container
		log.Infof("Stopping vCluster platform container %s...", PlatformContainerName)
		out, err := exec.CommandContext(ctx, "docker", "stop", containerID).CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to stop container: %w: %s", err, string(out))
		}

		// remove the container
		log.Infof("Removing vCluster platform container %s...", PlatformContainerName)
		out, err = exec.CommandContext(ctx, "docker", "rm", containerID).CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to remove container: %w: %s", err, string(out))
		}
	}

	// check if volume exists and remove it
	hasVolume, err := volumeExists(ctx, PlatformVolumeName)
	if err != nil {
		return fmt.Errorf("failed to check if volume exists: %w", err)
	}

	if hasVolume {
		foundVolume = true
		log.Infof("Removing vCluster platform volume %s...", PlatformVolumeName)
		out, err := exec.CommandContext(ctx, "docker", "volume", "rm", PlatformVolumeName).CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to remove volume: %w: %s", err, string(out))
		}
	}

	// check if anything was found
	if !foundContainer && !foundVolume {
		if ignoreNotFound {
			log.Info("No vCluster platform docker installation found")
			return nil
		}
		return ErrDockerPlatformNotFound
	}

	log.Info("Successfully destroyed vCluster platform docker installation")
	return nil
}

// findContainer finds a container by name and returns its ID
func findContainer(ctx context.Context, name string) (string, error) {
	out, err := exec.CommandContext(ctx, "docker", "ps", "-q", "-a", "-f", "name=^"+name+"$").CombinedOutput()
	if err != nil {
		return "", wrapCommandError(out, err)
	}

	containerID := strings.TrimSpace(string(out))
	return containerID, nil
}

// volumeExists checks if a docker volume exists
func volumeExists(ctx context.Context, name string) (bool, error) {
	out, err := exec.CommandContext(ctx, "docker", "volume", "ls", "-q", "-f", "name=^"+name+"$").CombinedOutput()
	if err != nil {
		return false, wrapCommandError(out, err)
	}

	return strings.TrimSpace(string(out)) != "", nil
}

func wrapCommandError(stdout []byte, err error) error {
	if err == nil {
		return nil
	}

	message := ""
	if len(stdout) > 0 {
		message += string(stdout) + "\n"
	}

	var exitError *exec.ExitError
	if errors.As(err, &exitError) && exitError != nil && len(exitError.Stderr) > 0 {
		message += string(exitError.Stderr) + "\n"
	}

	return fmt.Errorf("%s%w", message, err)
}
