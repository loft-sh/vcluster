package pod

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/util/clihelper"
	"github.com/loft-sh/vcluster/pkg/util/osutil"
	"github.com/loft-sh/vcluster/pkg/util/podhelper"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"
)

type Options struct {
	Exec             bool
	Mounts           []string
	Env              []string
	Image            string
	ServiceAccount   string
	ImagePullSecrets []string
}

func AddFlags(fs *pflag.FlagSet, podOptions *Options, isRestore bool) {
	if !isRestore {
		fs.BoolVar(&podOptions.Exec, "pod-exec", podOptions.Exec, "Instead of creating a pod, exec into the vCluster container")
	}
	fs.StringVar(&podOptions.Image, "pod-image", podOptions.Image, "Image to use for the created pod")
	fs.StringVar(&podOptions.ServiceAccount, "pod-service-account", podOptions.ServiceAccount, "Service account to use for the created pod")
	fs.StringArrayVar(&podOptions.Mounts, "pod-mount", nil, "Additional mounts for the created pod. Use form <type>:<name>/<key>:<mount>. Supported types are: pvc, secret, configmap. E.g.: pvc:my-pvc:/path-in-pod or secret:my-secret/my-key:/path-in-pod")
	fs.StringArrayVar(&podOptions.Env, "pod-env", nil, "Additional environment variables for the created pod. Use key=value. E.g.: MY_ENV=my-value")
	fs.StringArrayVar(&podOptions.ImagePullSecrets, "pod-image-pull-secret", nil, "Additional pull secrets for the created pod")
}

func SnapshotExec(
	ctx context.Context,
	kubeConfig *rest.Config,
	command []string,
	vCluster *find.VCluster,
	podOptions *Options,
	snapshotOptions *snapshot.Options,
) error {
	// get target pod
	var targetPod *corev1.Pod
	for _, pod := range vCluster.Pods {
		if vCluster.StatefulSet != nil && strings.HasSuffix(pod.Name, "-0") {
			targetPod = &pod
			break
		} else if vCluster.Deployment != nil {
			targetPod = &pod
			break
		}
	}
	if targetPod == nil {
		return fmt.Errorf("couldn't find a running pod for vCluster %s", vCluster.Name)
	}

	// build env variables
	optionsString, err := toOptionsString(snapshotOptions)
	if err != nil {
		return err
	}
	envVariables := append([]string{
		"VCLUSTER_STORAGE_OPTIONS=" + optionsString,
		"POD_NAME=" + vCluster.Name + "-0",
	}, podOptions.Env...)

	// run the command
	return podhelper.ExecStream(ctx, kubeConfig, &podhelper.ExecStreamOptions{
		Pod:       targetPod.Name,
		Namespace: vCluster.Namespace,
		Container: "syncer",
		Command:   []string{"sh", "-c", fmt.Sprintf("%s %s", strings.Join(envVariables, " "), strings.Join(command, " "))},
		Stdout:    os.Stdout,
		Stderr:    os.Stdout,
	})
}

func RunSnapshotPod(
	ctx context.Context,
	kubeConfig *rest.Config,
	kubeClient *kubernetes.Clientset,
	command []string,
	vCluster *find.VCluster,
	podOptions *Options,
	snapshotOptions *snapshot.Options,
	log log.Logger,
) error {
	// should exec?
	if podOptions.Exec {
		return SnapshotExec(ctx, kubeConfig, command, vCluster, podOptions, snapshotOptions)
	}

	// create snapshot pod
	snapshotPod, err := CreateSnapshotPod(
		ctx,
		kubeClient,
		command,
		vCluster,
		podOptions,
		snapshotOptions,
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
		_ = kubeClient.CoreV1().Pods(snapshotPod.Namespace).Delete(ctx, snapshotPod.Name, metav1.DeleteOptions{})
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
		err = kubeClient.CoreV1().Pods(snapshotPod.Namespace).Delete(ctx, snapshotPod.Name, metav1.DeleteOptions{
			GracePeriodSeconds: ptr.To(int64(1)),
		})
		if err != nil {
			klog.Warningf("Error deleting snapshot pod: %v", err)
		}
		osutil.Exit(1)
	}()

	// wait for pod to become ready
	err = WaitForReadyPod(ctx, kubeClient, snapshotPod.Namespace, snapshotPod.Name, "snapshot", log)
	if err != nil {
		return fmt.Errorf("waiting for restore pod to become ready: %w", err)
	}

	// now log the snapshot pod
	reader, err := kubeClient.CoreV1().Pods(snapshotPod.Namespace).GetLogs(snapshotPod.Name, &corev1.PodLogOptions{
		Follow: true,
	}).Stream(ctx)
	if err != nil {
		return fmt.Errorf("stream snapshot pod logs: %w", err)
	}
	defer reader.Close()

	// stream into stdout
	log.Infof("Printing logs of pod %s...", snapshotPod.Name)
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		return fmt.Errorf("write pod logs: %w", err)
	}

	// check restore pod for exit code
	exitCode, err := WaitForCompletedPod(ctx, kubeClient, snapshotPod.Namespace, snapshotPod.Name, "snapshot", time.Minute)
	if err != nil {
		return err
	}

	// check exit code of snapshot pod
	if exitCode != 0 {
		return fmt.Errorf("snapshot pod failed: exit code %d", exitCode)
	}

	return nil
}

func CreateSnapshotPod(
	ctx context.Context,
	kubeClient *kubernetes.Clientset,
	command []string,
	vCluster *find.VCluster,
	podOptions *Options,
	snapshotOptions *snapshot.Options,
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
	optionsString, err := toOptionsString(snapshotOptions)
	if err != nil {
		return nil, err
	}
	env = append(env, corev1.EnvVar{
		Name:  "VCLUSTER_STORAGE_OPTIONS",
		Value: optionsString,
	})

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

	// parse extra volumes
	extraVolumes, extraVolumeMounts, err := parseExtraVolumes(ctx, kubeClient, vCluster, podOptions.Mounts)
	if err != nil {
		return nil, fmt.Errorf("parsing extra volumes: %w", err)
	}

	// parse extra env
	extraEnv, err := parseExtraEnv(podOptions.Env)
	if err != nil {
		return nil, fmt.Errorf("parsing extra env: %w", err)
	}

	// image
	image := syncerContainer.Image
	if podOptions.Image != "" {
		image = podOptions.Image
	}

	// service account
	serviceAccount := podSpec.ServiceAccountName
	if podOptions.ServiceAccount != "" {
		serviceAccount = podOptions.ServiceAccount
	}

	// image pull secrets
	imagePullSecrets := podSpec.ImagePullSecrets
	for _, secret := range podOptions.ImagePullSecrets {
		imagePullSecrets = append(imagePullSecrets, corev1.LocalObjectReference{Name: secret})
	}

	// build the pod spec
	newPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "vcluster-snapshot-",
			Namespace:    vCluster.Namespace,
			Labels: map[string]string{
				"app": "vcluster-snapshot",
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy:                 corev1.RestartPolicyNever,
			ServiceAccountName:            serviceAccount,
			TerminationGracePeriodSeconds: ptr.To(int64(1)),
			NodeSelector:                  podSpec.NodeSelector,
			Affinity:                      podSpec.Affinity,
			Tolerations:                   podSpec.Tolerations,
			SecurityContext:               podSpec.SecurityContext,
			ImagePullSecrets:              imagePullSecrets,
			Volumes:                       append(podSpec.Volumes, extraVolumes...),
			InitContainers:                podSpec.InitContainers,
			Containers: []corev1.Container{
				{
					Name:            "snapshot",
					Image:           image,
					Command:         command,
					SecurityContext: syncerContainer.SecurityContext,
					Env:             append(env, extraEnv...),
					EnvFrom:         syncerContainer.EnvFrom,
					VolumeMounts:    append(syncerContainer.VolumeMounts, extraVolumeMounts...),
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

	// check if we need to error because we write the snapshot to a temporary directory
	if snapshotOptions != nil && snapshotOptions.Type == "container" && snapshotOptions.Container.Path != "" {
		// check if there is a mount at that path
		found := false
		for _, volumeMount := range newPod.Spec.Containers[0].VolumeMounts {
			isPvc := false
			for _, volume := range newPod.Spec.Volumes {
				if volume.Name == volumeMount.Name {
					if volume.VolumeSource.PersistentVolumeClaim != nil {
						isPvc = true
						break
					}
				}
			}
			if !isPvc {
				continue
			}

			if strings.HasPrefix(snapshotOptions.Container.Path, volumeMount.MountPath) {
				found = true
				break
			}
		}

		if !found {
			return nil, fmt.Errorf("container snapshot path %s is not persisted, taking a snapshot on this path has no effect since it will write the snapshot to a temporary filesystem", snapshotOptions.Container.Path)
		}
	}

	// create the pod
	log.Infof("Starting snapshot pod for vCluster %s/%s...", vCluster.Namespace, vCluster.Name)
	newPod, err = kubeClient.CoreV1().Pods(vCluster.Namespace).Create(ctx, newPod, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating pod: %w", err)
	}

	// print pod in debug mode
	if log.GetLevel() >= logrus.DebugLevel {
		out, err := yaml.Marshal(newPod)
		if err != nil {
			return nil, fmt.Errorf("marshalling pod: %w", err)
		}

		log.Debugf("Created snapshot pod: %s", string(out))
	}

	return newPod, nil
}

func toOptionsString(options *snapshot.Options) (string, error) {
	jsonBytes, err := json.Marshal(options)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(jsonBytes), nil
}

func parseExtraEnv(env []string) ([]corev1.EnvVar, error) {
	extraEnv := make([]corev1.EnvVar, 0, len(env))
	for _, envVar := range env {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid environment variable %s", envVar)
		}

		extraEnv = append(extraEnv, corev1.EnvVar{
			Name:  parts[0],
			Value: parts[1],
		})
	}

	return extraEnv, nil
}

func parseExtraVolumes(ctx context.Context, kubeClient *kubernetes.Clientset, vCluster *find.VCluster, volumes []string) ([]corev1.Volume, []corev1.VolumeMount, error) {
	extraVolumes := make([]corev1.Volume, 0, len(volumes))
	extraVolumeMounts := make([]corev1.VolumeMount, 0, len(volumes))
	for idx, volume := range volumes {
		volumeName := fmt.Sprintf("extra-volume-%d", idx)
		volumeSplit := strings.Split(volume, ":")
		if len(volumeSplit) != 3 {
			return nil, nil, fmt.Errorf("invalid volume format: %s, expected type:name:path", volume)
		}

		items := []corev1.KeyToPath{}
		name := volumeSplit[1]
		nameSplit := strings.Split(volumeSplit[1], "/")
		if len(nameSplit) == 2 {
			name = nameSplit[0]
			items = append(items, corev1.KeyToPath{
				Key:  nameSplit[1],
				Path: nameSplit[1],
			})
		}

		switch volumeSplit[0] {
		case "pvc":
			if len(items) > 0 {
				return nil, nil, fmt.Errorf("invalid name format: %s, expected type:name:path", name)
			}

			// check if the pvc exists
			_, err := kubeClient.CoreV1().PersistentVolumeClaims(vCluster.Namespace).Get(ctx, volumeSplit[1], metav1.GetOptions{})
			if err != nil {
				return nil, nil, fmt.Errorf("pvc %s not found", volumeSplit[1])
			}

			extraVolumes = append(extraVolumes, corev1.Volume{
				Name: volumeName,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: volumeSplit[1],
					},
				},
			})
		case "secret":
			// check if the secret exists
			_, err := kubeClient.CoreV1().Secrets(vCluster.Namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return nil, nil, fmt.Errorf("secret %s not found", name)
			}

			extraVolumes = append(extraVolumes, corev1.Volume{
				Name: volumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: name,
						Items:      items,
					},
				},
			})
		case "configmap":
			// check if the configmap exists
			_, err := kubeClient.CoreV1().ConfigMaps(vCluster.Namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return nil, nil, fmt.Errorf("configmap %s not found", name)
			}

			extraVolumes = append(extraVolumes, corev1.Volume{
				Name: volumeName,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: name,
						},
						Items: items,
					},
				},
			})
		default:
			return nil, nil, fmt.Errorf("invalid type: %s, expected pvc, secret or configmap", volumeSplit[0])
		}

		extraVolumeMounts = append(extraVolumeMounts, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: volumeSplit[2],
		})
	}

	return extraVolumes, extraVolumeMounts, nil
}

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
				}).Do(context.Background()).Raw()
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
			} else if containerStatus.State.Terminated != nil {
				exitCode = containerStatus.State.Terminated.ExitCode
				return true, nil
			} else if containerStatus.State.Waiting != nil {
				if containerStatus.State.Waiting.Message != "" {
					return false, fmt.Errorf("error: %s container is waiting: %s (%s)", container, containerStatus.State.Waiting.Message, containerStatus.State.Waiting.Reason)
				} else if containerStatus.State.Waiting.Reason != "" {
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
