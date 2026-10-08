package debugshell

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const (
	DefaultTTLForEphemeralContainer = "60m"
	ShellBannerEnv                  = "SHELL_BANNER"
)

// CreateEphemeralContainer builds an ephemeral container spec that runs a debug shell.
func CreateEphemeralContainer(targetContainer *corev1.Container, envs []corev1.EnvVar, name string, bannerEnvName string, ttl string) corev1.EphemeralContainer {
	return corev1.EphemeralContainer{
		EphemeralContainerCommon: corev1.EphemeralContainerCommon{
			Name:  name,
			Image: targetContainer.Image,
			// Print the banner once, then keep the container alive for the TTL window.
			Command:                  []string{"sh", "-c", "echo \"$" + bannerEnvName + "\"; sleep " + ttl},
			Env:                      envs,
			TTY:                      true,
			Stdin:                    true,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
			SecurityContext: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Add: []corev1.Capability{"SYS_PTRACE"},
				},
			},
		},
		TargetContainerName: targetContainer.Name,
	}
}

// FindExistingDebugShell returns the matching debug container name and whether it is running.
func FindExistingDebugShell(pod *corev1.Pod, namePrefix string) (string, bool) {
	// Build a quick lookup to correlate spec entries with their current status.
	statusByName := map[string]corev1.ContainerStatus{}
	for _, containerStatus := range pod.Status.EphemeralContainerStatuses {
		statusByName[containerStatus.Name] = containerStatus
	}

	for _, container := range pod.Spec.EphemeralContainers {
		// Only consider debug shell containers created by our prefix.
		if !strings.HasPrefix(container.Name, namePrefix) {
			continue
		}
		containerStatus, ok := statusByName[container.Name]
		// If running, return immediately with the resolved target name.
		if ok && containerStatus.State.Running != nil {
			return container.Name, true
		}
		// If terminated, skip and allow the caller to check another container, if none is found then create a new shell container.
		if ok && containerStatus.State.Terminated != nil {
			continue
		}
		// Otherwise return the spec entry and let the caller wait for it to start.
		return container.Name, false
	}
	return "", false
}

// FindTargetContainer locates the named container in the pod spec.
func FindTargetContainer(pod *corev1.Pod, containerName string) *corev1.Container {
	for _, container := range pod.Spec.Containers {
		if container.Name == containerName {
			return &container
		}
	}
	return nil
}

func SelectTargetPod(pods []corev1.Pod, requestedPodName string) (*corev1.Pod, error) {
	if len(pods) == 0 {
		return nil, fmt.Errorf("no pods found for virtual cluster instance")
	}

	if requestedPodName != "" {
		for _, candidate := range pods {
			if candidate.Name == requestedPodName {
				if candidate.Status.Phase != corev1.PodRunning {
					return nil, fmt.Errorf("pod %q is not running", requestedPodName)
				}
				return &candidate, nil
			}
		}
		return nil, fmt.Errorf("pod %q not found for virtual cluster instance", requestedPodName)
	}

	runningPod, ok := SelectRunningPod(pods)
	if !ok {
		return nil, fmt.Errorf("no running pods found for virtual cluster instance")
	}
	return runningPod, nil
}

// SelectRunningPod returns the first running pod in the list.
func SelectRunningPod(pods []corev1.Pod) (*corev1.Pod, bool) {
	for i := range pods {
		if pods[i].Status.Phase == corev1.PodRunning {
			return &pods[i], true
		}
	}
	return nil, false
}

func BuildEnvs(podName, vClusterVersion string) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "PATH",
			Value: "/usr/bin:/bin:/usr/local/bin:/proc/1/root/binaries",
		},
		{
			Name: ShellBannerEnv,
			// Banner is shown once on shell start to guide the user.
			Value: "This is an ephemeral container attached to your virtual cluster pod (" + podName + ").\n\n" +
				"vCluster version: " + vClusterVersion + "\n\n" +
				"This debug shell will run for " + DefaultTTLForEphemeralContainer + " and then exit.\n\n" +
				"Please do not modify any files in /proc/1, as they may affect the virtual cluster runtime.\n" +
				"Your virtual cluster config is located at:\n" +
				"$ cat /proc/1/root/var/lib/vcluster/config.yaml\n\n",
		},
	}
}
