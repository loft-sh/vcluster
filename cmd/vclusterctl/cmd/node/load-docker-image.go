package node

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/snapshot/pod"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type LoadImageOptions struct {
	*flags.GlobalFlags

	Image   string
	Archive string

	PodImage string

	Log log.Logger
}

func NewLoadImageCommand(globalFlags *flags.GlobalFlags) *cobra.Command {
	o := &LoadImageOptions{
		GlobalFlags: globalFlags,

		Log: log.GetInstance(),
	}

	cmd := &cobra.Command{
		Use:   "load-image",
		Short: "Load a local docker image into a node",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run(cmd.Context(), args[0])
		},
	}

	cmd.Flags().StringVar(&o.Image, "image", "", "The image to load")
	cmd.Flags().StringVar(&o.Archive, "archive", "", "The tar.gz archive to load")
	cmd.Flags().StringVar(&o.PodImage, "pod-image", "alpine", "The image to use for the node pod")

	return cmd
}

func (o *LoadImageOptions) Run(ctx context.Context, nodeName string) error {
	if o.Image == "" && o.Archive == "" {
		return fmt.Errorf("either --image or --archive is required")
	} else if o.Image != "" && o.Archive != "" {
		return fmt.Errorf("either --image or --archive is required")
	}

	// get kube client
	kubeClient, err := getClient(o.GlobalFlags)
	if err != nil {
		return fmt.Errorf("failed to get vcluster client: %w", err)
	}
	_, err = kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}

	// save image to archive
	if o.Image != "" {
		o.Log.Infof("Saving image %s to archive...", o.Image)
		if err := runCommand("docker", "save", "-o", "image.tar.gz", o.Image); err != nil {
			return fmt.Errorf("failed to save image: %w", err)
		}

		// cleanup
		defer os.Remove("image.tar.gz")
		o.Archive = "image.tar.gz"
	}

	// create a pod
	loadImagePod, err := kubeClient.CoreV1().Pods("kube-system").Create(ctx, getLoadImagePod(o.PodImage, nodeName), metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create pod: %w", err)
	}
	defer func() {
		o.Log.Infof("Deleting pod %s...", loadImagePod.Name)
		if err := kubeClient.CoreV1().Pods("kube-system").Delete(ctx, loadImagePod.Name, metav1.DeleteOptions{}); err != nil {
			o.Log.Errorf("failed to delete pod: %w", err)
		}
	}()

	// wait for the pod to be ready
	o.Log.Infof("Waiting for load image pod %s to be ready...", loadImagePod.Name)
	err = pod.WaitForReadyPod(ctx, kubeClient, loadImagePod.Namespace, loadImagePod.Name, "load-image", o.Log)
	if err != nil {
		return fmt.Errorf("failed to wait for load image pod to be ready: %w", err)
	}

	// copy the archive to the pod
	o.Log.Infof("Copying archive to pod %s...", loadImagePod.Name)
	if err := runCommand("kubectl", "cp", o.Archive, fmt.Sprintf("%s/%s:/host/tmp/image.tar.gz", loadImagePod.Namespace, loadImagePod.Name)); err != nil {
		return fmt.Errorf("failed to copy archive to pod: %w", err)
	}

	// load the image in the node
	o.Log.Infof("Importing image in node %s...", nodeName)
	if err := runCommand("kubectl", "exec", loadImagePod.Name, "-n", loadImagePod.Namespace, "--", "nsenter", "-t", "1", "-m", "-u", "-i", "-n", "-S", "0", "-G", "0", "sh", "-c", "ctr --namespace=k8s.io images import /tmp/image.tar.gz && rm -f /tmp/image.tar.gz"); err != nil {
		return fmt.Errorf("failed to load image in node: %w", err)
	}

	return nil
}

func runCommand(args ...string) error {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func getLoadImagePod(image, node string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "load-image-",
			Namespace:    "kube-system",
		},
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: &[]int64{1}[0],
			NodeName:                      node,
			HostPID:                       true,
			Containers: []corev1.Container{
				{
					Name:    "load-image",
					Image:   image,
					Command: []string{"tail", "-f", "/dev/null"},
					SecurityContext: &corev1.SecurityContext{
						Privileged: &[]bool{true}[0],
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
}
