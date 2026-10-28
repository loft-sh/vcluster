package podhelper

import (
	"context"
	"fmt"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/util/clihelper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

func WaitForReadyPod(ctx context.Context, kubeClient kubernetes.Interface, namespace, name, container string, log log.Logger) error {
	now := time.Now()
	err := wait.PollUntilContextTimeout(ctx, time.Second*2, time.Minute*2, true, func(ctx context.Context) (bool, error) {
		pod, err := kubeClient.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			// this is a fatal
			return false, fmt.Errorf("error trying to retrieve pod %s/%s: %w", namespace, name, err)
		}

		found := false
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.State.Running != nil && containerStatus.Ready {
				if containerStatus.Name == container {
					found = true
				}

				continue
			} else if containerStatus.State.Terminated != nil || (containerStatus.State.Waiting != nil && clihelper.CriticalStatus[containerStatus.State.Waiting.Reason]) {
				// if the container is completed that is fine as well
				if containerStatus.State.Terminated != nil && containerStatus.State.Terminated.ExitCode == 0 {
					found = true
					continue
				}

				reason := ""
				message := ""
				if containerStatus.State.Terminated != nil {
					reason = containerStatus.State.Terminated.Reason
					message = containerStatus.State.Terminated.Message
				} else if containerStatus.State.Waiting != nil {
					reason = containerStatus.State.Waiting.Reason
					message = containerStatus.State.Waiting.Message
				}

				out, err := kubeClient.CoreV1().Pods(namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
					Container: container,
				}).Do(ctx).Raw()
				if err != nil {
					return false, fmt.Errorf("there seems to be an issue with pod %s/%s starting up: %s (%s)", namespace, name, message, reason)
				}

				return false, fmt.Errorf("there seems to be an issue with pod %s (%s - %s), logs:\n%s", name, message, reason, string(out))
			} else if containerStatus.State.Waiting != nil && time.Now().After(now.Add(time.Second*10)) {
				if containerStatus.State.Waiting.Message != "" {
					log.Infof("Please keep waiting, %s container is still starting up: %s (%s)", container, containerStatus.State.Waiting.Message, containerStatus.State.Waiting.Reason)
				} else if containerStatus.State.Waiting.Reason != "" {
					log.Infof("Please keep waiting, %s container is still starting up: %s", container, containerStatus.State.Waiting.Reason)
				} else {
					log.Infof("Please keep waiting, %s container is still starting up...", container)
				}

				now = time.Now()
			}

			return false, nil
		}

		return found, nil
	})
	if err != nil {
		return err
	}

	return nil
}

func WaitForCompletedPod(ctx context.Context, kubeClient *kubernetes.Clientset, namespace, name, container string, timeout time.Duration) (int32, error) {
	exitCode := int32(-1)
	err := wait.PollUntilContextTimeout(ctx, time.Second*2, timeout, true, func(ctx context.Context) (bool, error) {
		pod, err := kubeClient.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			// this is a fatal
			return false, fmt.Errorf("error trying to retrieve pod %s/%s: %w", namespace, name, err)
		}

		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.Name != container {
				continue
			}
			if containerStatus.State.Running != nil {
				return false, nil
			}
			if containerStatus.State.Terminated != nil {
				exitCode = containerStatus.State.Terminated.ExitCode
				return true, nil
			}
			if containerStatus.State.Waiting != nil {
				if containerStatus.State.Waiting.Message != "" {
					return false, fmt.Errorf("error: %s container is waiting: %s (%s)", container, containerStatus.State.Waiting.Message, containerStatus.State.Waiting.Reason)
				}
				if containerStatus.State.Waiting.Reason != "" {
					return false, fmt.Errorf("error: %s container is waiting: %s", container, containerStatus.State.Waiting.Reason)
				}

				return false, fmt.Errorf("error: %s container is waiting", container)
			}

			return false, nil
		}

		return false, nil
	})
	if err != nil {
		return exitCode, err
	}

	return exitCode, nil
}
