package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// VCluster holds information about a cluster
type VCluster struct {
	Name       string
	Namespace  string
	Created    time.Time
	AgeSeconds int
	Status     string
}

const VirtualClusterSelector = "app=vcluster"

// ListCmd holds the login cmd flags
type ListCmd struct {
	*flags.GlobalFlags

	log    log.Logger
	output string
}

// NewListCmd creates a new command
func NewListCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ListCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "list",
		Short: "Lists all virtual clusters",
		Long: `
#######################################################
#################### vcluster list ####################
#######################################################
Lists all virtual clusters

Example:
vcluster list
vcluster list --output json
vcluster list --namespace test
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd, args)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.output, "output", "table", "Choose the format of the output. [table|json]")

	return cobraCmd
}

// Run executes the functionality
func (cmd *ListCmd) Run(cobraCmd *cobra.Command, args []string) error {
	// first load the kube config
	kubeClientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{
		CurrentContext: cmd.Context,
	})
	namespace := metav1.NamespaceAll
	if cmd.Namespace != "" {
		namespace = cmd.Namespace
	}

	// get all statefulsets with the label app=vcluster
	restConfig, err := kubeClientConfig.ClientConfig()
	if err != nil {
		return err
	}
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	vclusters := []VCluster{}

	// statefulset based vclusters
	statefulSets, err := getStatefulSets(client, namespace, kubeClientConfig)
	if err != nil {
		return err
	}
	for _, p := range statefulSets.Items {
		if v, ok := p.Labels["release"]; ok {
			vCluster, err := getVCluster(&p, v, client, kubeClientConfig)
			if err != nil {
				return err
			}
			vclusters = append(vclusters, vCluster)
		}
	}

	// deployment based vclusters
	deployments, err := getDeployments(client, namespace, kubeClientConfig)
	if err != nil {
		return err
	}
	for _, p := range deployments.Items {
		if v, ok := p.Labels["release"]; ok {
			vCluster, err := getVCluster(&p, v, client, kubeClientConfig)
			if err != nil {
				return err
			}
			vclusters = append(vclusters, vCluster)
		}
	}

	if cmd.output == "json" {
		bytes, err := json.MarshalIndent(&vclusters, "", "    ")
		if err != nil {
			return errors.Wrap(err, "json marshal vclusters")
		}
		cmd.log.WriteString(string(bytes) + "\n")
	} else {
		header := []string{"NAME", "NAMESPACE", "STATUS", "CREATED", "AGE"}
		values := [][]string{}
		for _, vcluster := range vclusters {
			values = append(values, []string{
				vcluster.Name,
				vcluster.Namespace,
				vcluster.Status,
				vcluster.Created.String(),
				time.Since(vcluster.Created).Round(1 * time.Second).String(),
			})
		}

		log.PrintTable(cmd.log, header, values)
	}

	return nil
}

func getVCluster(object client.Object, v string, client *kubernetes.Clientset, kubeClientConfig clientcmd.ClientConfig) (VCluster, error) {
	namespace := object.GetNamespace()
	created := object.GetCreationTimestamp().Time
	var releaseName string
	var status string
	if object.GetAnnotations() != nil && object.GetAnnotations()[PausedAnnotation] == "true" {
		status = "Paused"
	} else {
		releaseName = "release=" + v
	}
	if len(status) < 1 {
		pods, err := getPods(client, kubeClientConfig, namespace, releaseName)
		if err != nil {
			return VCluster{}, err
		}
		for _, pod := range pods.Items {
			status = GetPodStatus(&pod)
		}
	}
	if len(status) < 1 {
		status = "Unknown"
	}
	return VCluster{
		Name:       v,
		Namespace:  namespace,
		Created:    created,
		AgeSeconds: int(time.Since(created).Seconds()),
		Status:     status,
	}, nil
}

func getPods(client *kubernetes.Clientset, kubeClientConfig clientcmd.ClientConfig, namespace, podSelector string) (*corev1.PodList, error) {
	podList, err := client.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: VirtualClusterSelector + "," + podSelector,
	})
	if err != nil {
		if kerrors.IsForbidden(err) {
			// try the current namespace instead
			if namespace, err = getAccessibleNS(kubeClientConfig); err != nil {
				return nil, err
			}
			return client.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: VirtualClusterSelector,
			})
		}
		return nil, err
	}
	return podList, nil
}

func getDeployments(client *kubernetes.Clientset, namespace string, kubeClientConfig clientcmd.ClientConfig) (*appsv1.DeploymentList, error) {
	deploymentList, err := client.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: VirtualClusterSelector,
	})
	if err != nil {
		if kerrors.IsForbidden(err) {
			// try the current namespace instead
			if namespace, err = getAccessibleNS(kubeClientConfig); err != nil {
				return nil, err
			}
			return client.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: VirtualClusterSelector,
			})
		}
		return nil, err
	}
	return deploymentList, nil
}

func getStatefulSets(client *kubernetes.Clientset, namespace string, kubeClientConfig clientcmd.ClientConfig) (*appsv1.StatefulSetList, error) {
	statefulSetList, err := client.AppsV1().StatefulSets(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: VirtualClusterSelector,
	})
	if err != nil {
		if kerrors.IsForbidden(err) {
			if namespace, err = getAccessibleNS(kubeClientConfig); err != nil {
				return nil, err
			}
			return client.AppsV1().StatefulSets(namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: VirtualClusterSelector,
			})
		}
		return nil, err
	}
	return statefulSetList, nil
}

func getAccessibleNS(kubeClientConfig clientcmd.ClientConfig) (string, error) {
	// try the current namespace instead
	namespace, _, err := kubeClientConfig.Namespace()
	if err != nil {
		return "", err
	} else if namespace == "" {
		namespace = "default"
	}
	return namespace, nil
}

// GetPodStatus returns the pod status as a string
// Taken from https://github.com/kubernetes/kubernetes/pkg/printers/internalversion/printers.go
func GetPodStatus(pod *corev1.Pod) string {
	reason := string(pod.Status.Phase)
	if pod.Status.Reason != "" {
		reason = pod.Status.Reason
	}
	initializing := false
	for i := range pod.Status.InitContainerStatuses {
		container := pod.Status.InitContainerStatuses[i]
		switch {
		case container.State.Terminated != nil && container.State.Terminated.ExitCode == 0:
			continue
		case container.State.Terminated != nil:
			// initialization is failed
			if len(container.State.Terminated.Reason) == 0 {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Init:Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("Init:ExitCode:%d", container.State.Terminated.ExitCode)
				}
			} else {
				reason = "Init:" + container.State.Terminated.Reason
			}
			initializing = true
		case container.State.Waiting != nil && len(container.State.Waiting.Reason) > 0 && container.State.Waiting.Reason != "PodInitializing":
			reason = "Init:" + container.State.Waiting.Reason
			initializing = true
		default:
			reason = fmt.Sprintf("Init:%d/%d", i, len(pod.Spec.InitContainers))
			initializing = true
		}
		break
	}
	if !initializing {
		hasRunning := false
		for i := len(pod.Status.ContainerStatuses) - 1; i >= 0; i-- {
			container := pod.Status.ContainerStatuses[i]
			if container.State.Waiting != nil && container.State.Waiting.Reason != "" {
				reason = container.State.Waiting.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason != "" {
				reason = container.State.Terminated.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason == "" {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("ExitCode:%d", container.State.Terminated.ExitCode)
				}
			} else if container.Ready && container.State.Running != nil {
				hasRunning = true
			}
		}
		// change pod status back to "Running" if there is at least one container still reporting as "Running" status
		if reason == "Completed" && hasRunning {
			reason = "Running"
		}
	}
	if pod.DeletionTimestamp != nil && pod.Status.Reason == "NodeLost" {
		reason = "Unknown"
	} else if pod.DeletionTimestamp != nil {
		reason = "Terminating"
	}
	return reason
}
