package node

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/snapshot/pod"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/ptr"
)

type UpgradeOptions struct {
	*flags.GlobalFlags

	Log log.Logger

	BinariesPath     string
	CNIBinariesPath  string
	BundleRepository string

	Image           string
	ImagePullPolicy string
}

func NewUpgradeCommand(globalFlags *flags.GlobalFlags) *cobra.Command {
	options := &UpgradeOptions{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	upgradeCmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade the node",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return options.Run(cmd.Context(), args)
		},
	}

	upgradeCmd.Flags().StringVar(&options.BundleRepository, "bundle-repository", "https://github.com/loft-sh/kubernetes/releases/download", "The repository to use for downloading the Kubernetes bundle")
	upgradeCmd.Flags().StringVar(&options.BinariesPath, "binaries-path", "/usr/local/bin", "The path to the kubeadm binaries")
	upgradeCmd.Flags().StringVar(&options.CNIBinariesPath, "cni-binaries-path", "/opt/cni/bin", "The path to the CNI binaries")
	upgradeCmd.Flags().StringVar(&options.Image, "image", "", "The image to use for the upgrade")
	upgradeCmd.Flags().StringVar(&options.ImagePullPolicy, "image-pull-policy", "IfNotPresent", "The image pull policy")

	return upgradeCmd
}

func (o *UpgradeOptions) Run(ctx context.Context, args []string) error {
	// get the node name
	nodeName := args[0]
	kubeClient, err := getClient(o.GlobalFlags)
	if err != nil {
		return fmt.Errorf("failed to get vcluster client: %w", err)
	}

	// get the image
	image := o.Image
	if image == "" {
		image = "ghcr.io/loft-sh/vcluster-pro:" + strings.TrimPrefix(upgrade.GetVersion(), "v")
	}

	// figure out kubernetes version
	serverVersion, err := kubeClient.DiscoveryClient.ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to get server version: %w", err)
	}
	kubernetesVersion := serverVersion.GitVersion

	// get the node object
	node, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}
	if node.Status.NodeInfo.KubeletVersion == kubernetesVersion {
		o.Log.Infof("Node %s is already running kubernetes version %s", nodeName, kubernetesVersion)
		return nil
	}

	// get the command
	command := []string{"/vcluster", "node", "upgrade", "--kubernetes-version", kubernetesVersion}
	if o.BinariesPath != "" {
		command = append(command, "--binaries-path", o.BinariesPath)
	}
	if o.CNIBinariesPath != "" {
		command = append(command, "--cni-binaries-path", o.CNIBinariesPath)
	}

	// create a pod with the image
	upgradePod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "upgrade-node-",
			Namespace:    "kube-system",
			Labels: map[string]string{
				"vcluster.loft.sh/upgrade-node": "true",
			},
		},
		Spec: corev1.PodSpec{
			NodeName:      nodeName,
			RestartPolicy: corev1.RestartPolicyNever,
			HostPID:       true,
			HostNetwork:   true,
			Tolerations: []corev1.Toleration{
				{
					Operator: corev1.TolerationOpExists,
				},
			},
			PriorityClassName: "system-node-critical",
			Containers: []corev1.Container{
				{
					Name:            "upgrade",
					Image:           image,
					ImagePullPolicy: corev1.PullPolicy(o.ImagePullPolicy),
					Command:         command,
					SecurityContext: &corev1.SecurityContext{
						Privileged: ptr.To(true),
					},
					Env: []corev1.EnvVar{
						{
							Name:  "NODE_NAME",
							Value: nodeName,
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "host",
							MountPath: "/host",
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "host",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/",
						},
					},
				},
			},
		},
	}

	// create the pod
	o.Log.Infof("Creating upgrade pod...")
	upgradePod, err = kubeClient.CoreV1().Pods("kube-system").Create(ctx, upgradePod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create upgrade pod: %w", err)
	}

	// wait for the pod to be ready
	o.Log.Infof("Waiting for upgrade pod %s to be ready...", upgradePod.Name)
	err = pod.WaitForReadyPod(ctx, kubeClient, upgradePod.Namespace, upgradePod.Name, "upgrade", o.Log)
	if err != nil {
		return fmt.Errorf("failed to wait for upgrade pod to be ready: %w", err)
	}

	// now log the upgrade pod
	reader, err := kubeClient.CoreV1().Pods(upgradePod.Namespace).GetLogs(upgradePod.Name, &corev1.PodLogOptions{
		Follow: true,
	}).Stream(ctx)
	if err != nil {
		o.Log.Errorf("stream upgrade pod logs: %w", err)
	} else {
		defer reader.Close()

		// stream into stdout
		o.Log.Infof("Printing logs of pod %s...", upgradePod.Name)
		_, _ = io.Copy(os.Stdout, reader)
	}

	// check restore pod for exit code
	o.Log.Infof("Waiting for upgrade pod %s to complete...", upgradePod.Name)
	exitCode, err := pod.WaitForCompletedPod(ctx, kubeClient, upgradePod.Namespace, upgradePod.Name, "upgrade", 30*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to wait for upgrade pod to complete: %w", err)
	}

	// check if the node is schedulable again
	o.Log.Infof("Waiting for node %s to be at kubernetes version %s...", nodeName, kubernetesVersion)
	err = wait.PollUntilContextTimeout(ctx, time.Second*2, time.Minute, true, func(ctx context.Context) (bool, error) {
		node, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to get node: %w", err)
		}

		// check if the node is at the correct version and is schedulable again
		if node.Status.NodeInfo.KubeletVersion == kubernetesVersion && !node.Spec.Unschedulable {
			return true, nil
		}

		return false, nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for node to be at kubernetes version %s: %w", kubernetesVersion, err)
	}

	// check exit code of upgrade pod
	if exitCode == 1 {
		return fmt.Errorf("upgrade pod failed: exit code %d", exitCode)
	}

	// delete the pod when we are done
	o.Log.Infof("Deleting upgrade pod...")
	err = kubeClient.CoreV1().Pods(upgradePod.Namespace).Delete(ctx, upgradePod.Name, metav1.DeleteOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete upgrade pod: %w", err)
	}

	o.Log.Infof("Upgrade completed successfully")
	return nil
}

func getClient(flags *flags.GlobalFlags) (*kubernetes.Clientset, error) {
	// first load the kube config
	kubeClientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{
		CurrentContext: flags.Context,
	})

	// get the client config
	restConfig, err := kubeClientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get client config: %w", err)
	}

	return kubernetes.NewForConfig(restConfig)
}
