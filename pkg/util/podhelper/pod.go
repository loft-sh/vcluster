package podhelper

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"
)

func RunSyncerPod(
	ctx context.Context,
	containerName string,
	kubeClient *kubernetes.Clientset,
	command []string,
	vCluster *find.VCluster,
	writer io.Writer,
	log log.Logger,
) error {
	// create pod
	pod, err := CreateSyncerPod(
		ctx,
		containerName,
		kubeClient,
		command,
		vCluster,
		log,
	)
	if err != nil {
		return err
	}

	// create interrupt channel
	sigint := make(chan os.Signal, 1)
	defer func() {
		// make sure we won't interfere with interrupts anymore
		signal.Stop(sigint)

		// delete the pod when we are done
		_ = kubeClient.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
	}()

	// also delete on interrupt
	go func() {
		// interrupt signal sent from terminal
		signal.Notify(sigint, os.Interrupt)
		// sigterm signal sent from kubernetes
		signal.Notify(sigint, syscall.SIGTERM)

		// wait until we get killed
		<-sigint

		// cleanup pod
		err = kubeClient.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{
			GracePeriodSeconds: ptr.To(int64(1)),
		})
		if err != nil {
			klog.Warningf("Error deleting %s pod: %v", containerName, err)
		}
		os.Exit(1)
	}()

	// wait for pod to become ready
	err = WaitForReadyPod(ctx, kubeClient, pod.Namespace, pod.Name, containerName, log)
	if err != nil {
		return fmt.Errorf("waiting for %s pod to become ready: %w", containerName, err)
	}

	// now log the pod
	reader, err := kubeClient.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
		Follow: true,
	}).Stream(ctx)
	if err != nil {
		return fmt.Errorf("stream %s pod logs: %w", containerName, err)
	}
	defer reader.Close()

	// stream into writer or os.Stdout
	if writer == nil {
		writer = os.Stdout
	}
	_, err = io.Copy(writer, reader)
	if err != nil {
		return fmt.Errorf("write pod logs: %w", err)
	}

	// check pod for exit code
	exitCode, err := WaitForCompletedPod(ctx, kubeClient, pod.Namespace, pod.Name, containerName, time.Minute)
	if err != nil {
		return err
	}

	// check exit code of pod
	if exitCode != 0 {
		return fmt.Errorf("%s pod failed: exit code %d", containerName, exitCode)
	}

	return nil
}

func CreateSyncerPod(
	ctx context.Context,
	containerName string,
	kubeClient *kubernetes.Clientset,
	command []string,
	vCluster *find.VCluster,
	log log.Logger,
) (*corev1.Pod, error) {
	// get pod spec
	var podSpec *corev1.PodSpec
	if vCluster.StatefulSet != nil {
		podSpec = &vCluster.StatefulSet.Spec.Template.Spec
	} else if vCluster.Deployment != nil {
		podSpec = &vCluster.Deployment.Spec.Template.Spec
	} else {
		return nil, fmt.Errorf("vCluster %s has no StatefulSet or Deployment", vCluster.Name)
	}

	var syncerContainer *corev1.Container
	for _, container := range podSpec.Containers {
		if container.Name == "syncer" {
			syncerContainer = &container
			break
		}
	}
	if syncerContainer == nil {
		return nil, fmt.Errorf("couldn't find syncer container")
	}

	// build args
	env := syncerContainer.Env

	// this is needed for embedded etcd as it otherwise wouldn't
	// start the embedded etcd cluster correctly
	env = slices.DeleteFunc(env, func(envVar corev1.EnvVar) bool {
		return envVar.Name == "POD_NAME"
	})
	env = append(env, corev1.EnvVar{
		Name:  "POD_NAME",
		Value: vCluster.Name + "-0",
	})

	// add debug
	if log.GetLevel() >= logrus.DebugLevel {
		env = append(env, corev1.EnvVar{
			Name:  "DEBUG",
			Value: "true",
		})
	}

	// build the pod spec
	newPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("vcluster-%s-", containerName),
			Namespace:    vCluster.Namespace,
			Labels: map[string]string{
				"app": fmt.Sprintf("vcluster-%s", containerName),
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy:                 corev1.RestartPolicyNever,
			ServiceAccountName:            podSpec.ServiceAccountName,
			TerminationGracePeriodSeconds: ptr.To(int64(1)),
			NodeSelector:                  podSpec.NodeSelector,
			Affinity:                      podSpec.Affinity,
			Tolerations:                   podSpec.Tolerations,
			SecurityContext:               podSpec.SecurityContext,
			ImagePullSecrets:              podSpec.ImagePullSecrets,
			Volumes:                       podSpec.Volumes,
			Containers: []corev1.Container{
				{
					Name:            containerName,
					Image:           syncerContainer.Image,
					Command:         command,
					SecurityContext: syncerContainer.SecurityContext,
					Env:             env,
					EnvFrom:         syncerContainer.EnvFrom,
					VolumeMounts:    syncerContainer.VolumeMounts,
				},
			},
		},
	}

	// add persistent volume claim volume if necessary
	for _, volumeMount := range syncerContainer.VolumeMounts {
		if volumeMount.Name == "data" {
			// check if its part of the pod spec
			found := false
			for _, volume := range newPod.Spec.Volumes {
				if volume.Name == volumeMount.Name {
					found = true
					break
				}
			}
			if !found {
				newPod.Spec.Volumes = append(newPod.Spec.Volumes, corev1.Volume{
					Name: volumeMount.Name,
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "data-" + vCluster.Name + "-0",
						},
					},
				})
			}
		}
	}

	newPod, err := kubeClient.CoreV1().Pods(vCluster.Namespace).Create(ctx, newPod, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating pod: %w", err)
	}

	// print pod in debug mode
	if log.GetLevel() >= logrus.DebugLevel {
		out, err := yaml.Marshal(newPod)
		if err != nil {
			return nil, fmt.Errorf("marshalling pod: %w", err)
		}

		log.Debugf("Created %s pod: %s", containerName, string(out))
	}

	return newPod, nil
}
