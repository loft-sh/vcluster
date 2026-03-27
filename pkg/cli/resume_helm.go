package cli

import (
	"context"
	"errors"
	"fmt"
	"time"

	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/lifecycle"
	"github.com/loft-sh/vcluster/pkg/platform/clihelper"
	"github.com/loft-sh/vcluster/pkg/platform/sleepmode"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

type ResumeOptions struct {
	Driver string

	Project string
}

var (
	ErrPlatformDriverRequired = errors.New("cannotwakea virtual cluster that is paused by the platform, please run 'vcluster use driver platform' or use the '--driver platform' flag")
	errWorkloadWake           = errors.New("failed to wake")
)

func ResumeHelm(ctx context.Context, globalFlags *flags.GlobalFlags, vClusterName string, log log.Logger) error {
	vCluster, err := find.GetVCluster(ctx, globalFlags.Context, vClusterName, globalFlags.Namespace, log)
	if err != nil {
		return err
	}

	if vCluster.IsSleeping() {
		return ErrPlatformDriverRequired
	}

	kubeClient, err := prepareResume(vCluster, globalFlags)
	if err != nil {
		return err
	}

	if vCluster.Status == find.StatusWorkloadSleeping {
		if err := workloadWake(ctx, kubeClient, vCluster); err != nil {
			return err
		}
		log.Donef("Successfully woke vcluster %s in namespace %s", vClusterName, globalFlags.Namespace)
		return nil
	}

	err = lifecycle.ResumeVCluster(ctx, kubeClient, vClusterName, globalFlags.Namespace, false, log)
	if err != nil {
		return err
	}

	log.Donef("Successfully woke vcluster %s in namespace %s", vClusterName, globalFlags.Namespace)
	return nil
}

// workloadWake clears the sleep annotations from the vc-config secret, waking the
// vCluster's workloads. For standalone vClusters it also clears the sleep secret inside the
// virtual cluster that the in-cluster sleep state controller watches.
func workloadWake(ctx context.Context, kubeClient kubernetes.Interface, vCluster *find.VCluster) (retErr error) {
	configSecret, err := kubeClient.CoreV1().Secrets(vCluster.Namespace).Get(ctx, vClusterConfigSecretName(vCluster.Name), metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get config secret: %w", err)
	}

	vClusterConfig, hasConfig, err := vClusterConfigFromSecret(configSecret)
	if err != nil {
		return err
	}
	if hasConfig && vClusterConfig.ControlPlane.Standalone.Enabled {
		virtualKubeClient, err := standaloneKubeClient(vCluster)
		if err != nil {
			return err
		}

		return wakeStandaloneOrRollback(ctx, kubeClient, virtualKubeClient, vCluster.Namespace, configSecret)
	}

	return clearSecretSleepAnnotations(ctx, kubeClient, vCluster.Namespace, configSecret)
}

func wakeStandaloneOrRollback(ctx context.Context, hostKubeClient, virtualKubeClient kubernetes.Interface, namespace string, configSecret *corev1.Secret) error {
	if err := wakeStandalone(ctx, virtualKubeClient); err != nil {
		return errWorkloadWake
	}

	originalConfigSecret := configSecret.DeepCopy()
	if err := clearSecretSleepAnnotations(ctx, hostKubeClient, namespace, configSecret.DeepCopy()); err != nil {
		if rollbackErr := wait.PollUntilContextTimeout(ctx, time.Second, clihelper.Timeout(), true, func(ctx context.Context) (bool, error) {
			if err := restoreStandaloneWorkloadSleep(ctx, virtualKubeClient, originalConfigSecret); err != nil {
				return false, nil
			}

			return true, nil
		}); rollbackErr != nil {
			return errWorkloadWake
		}

		return errWorkloadWake
	}

	return nil
}

func restoreStandaloneWorkloadSleep(ctx context.Context, virtualKubeClient kubernetes.Interface, configSecret *corev1.Secret) error {
	sleepType, hasSleepType := configSecret.Annotations[clusterv1.SleepModeSleepTypeAnnotation]
	sleepingSince, hasSleepingSince := configSecret.Annotations[clusterv1.SleepModeSleepingSinceAnnotation]
	if !hasSleepType || sleepType == "" || !hasSleepingSince {
		return nil
	}

	forceDuration, hasForceDuration := configSecret.Annotations[clusterv1.SleepModeForceDurationAnnotation]
	return ensureAndUpdateSecret(ctx, virtualKubeClient, defaultSleepModeNamespace, sleepmode.StandaloneSleepSecretName,
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: sleepmode.StandaloneSleepSecretName, Namespace: defaultSleepModeNamespace}},
		func(s *corev1.Secret) {
			s.Annotations[clusterv1.SleepModeSleepTypeAnnotation] = sleepType
			s.Annotations[clusterv1.SleepModeSleepingSinceAnnotation] = sleepingSince
			if hasForceDuration {
				s.Annotations[clusterv1.SleepModeForceDurationAnnotation] = forceDuration
			} else {
				delete(s.Annotations, clusterv1.SleepModeForceDurationAnnotation)
			}
			delete(s.Annotations, clusterv1.SleepModeForceAnnotation)
			delete(s.Annotations, clusterv1.SleepModeLastActivityAnnotation)
		},
	)
}

// wakeStandalone clears sleep annotations from the vc-standalone-sleep-state
// secret inside the virtual cluster's default namespace. Mirrors sleepStandalone.
func wakeStandalone(ctx context.Context, virtualKubeClient kubernetes.Interface) error {
	// nil initial: if the secret doesn't exist there is nothing to wake.
	return ensureAndUpdateSecret(ctx, virtualKubeClient, defaultSleepModeNamespace, sleepmode.StandaloneSleepSecretName, nil, func(s *corev1.Secret) {
		clearSleepAnnotations(s)
	})
}

func prepareResume(vCluster *find.VCluster, globalFlags *flags.GlobalFlags) (*kubernetes.Clientset, error) {
	// load the rest config
	kubeConfig, err := vCluster.ClientFactory.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("there is an error loading your current kube config (%w), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	globalFlags.Namespace = vCluster.Namespace
	return kubeClient, nil
}
