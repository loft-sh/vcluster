package find

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const VirtualClusterSelector = "app=vcluster"

type VCluster struct {
	Name      string
	Namespace string

	Status        Status
	Created       metav1.Time
	Context       string
	ClientFactory clientcmd.ClientConfig `json:"-"`
}

type Status string

const (
	StatusRunning Status = "Running"
	StatusPaused  Status = "Paused"
	StatusUnknown Status = "Unknown"
)

func CurrentContext() (string, *api.Config, error) {
	rawConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).RawConfig()
	if err != nil {
		return "", nil, err
	}

	return rawConfig.CurrentContext, &rawConfig, nil
}

func GetVCluster(ctx context.Context, context, name, namespace string) (*VCluster, error) {
	if name == "" {
		return nil, fmt.Errorf("please specify a name")
	}

	vclusters, err := ListVClusters(ctx, context, name, namespace)
	if err != nil {
		return nil, err
	} else if len(vclusters) == 0 {
		return nil, fmt.Errorf("couldn't find vcluster %s", name)
	} else if len(vclusters) == 1 {
		return &vclusters[0], nil
	}

	return nil, fmt.Errorf("multiple vclusters with name %s found, please specify a namespace via -n", name)
}

func ListVClusters(ctx context.Context, context, name, namespace string) ([]VCluster, error) {
	if context == "" {
		var err error
		context, _, err = CurrentContext()
		if err != nil {
			return nil, err
		}
	}

	vClusterName, _, vClusterContext := VClusterFromContext(context)
	timeout := time.Minute
	if vClusterName != "" {
		timeout = time.Second * 5
	}

	vclusters, err := findInContext(ctx, context, name, namespace, timeout, false)
	// In case of error in vcluster listing in vcluster context, the below check will skip the error and try searching for parent context vclusters.
	if err != nil && vClusterName == "" {
		return nil, errors.Wrap(err, "find vcluster")
	}

	if vClusterName != "" {
		parentContextVclusters, err := findInContext(ctx, vClusterContext, name, namespace, time.Minute, true)
		if err != nil {
			return nil, errors.Wrap(err, "find vcluster")
		}

		vclusters = append(vclusters, parentContextVclusters...)
	}

	return vclusters, nil
}

func VClusterContextName(vClusterName string, vClusterNamespace string, currentContext string) string {
	return "vcluster_" + vClusterName + "_" + vClusterNamespace + "_" + currentContext
}

func VClusterConnectBackgroundProxyName(vClusterName string, vClusterNamespace string, currentContext string) string {
	return VClusterContextName(vClusterName, vClusterNamespace, currentContext) + "_background_proxy"
}

func VClusterFromContext(originalContext string) (name string, namespace string, context string) {
	if !strings.HasPrefix(originalContext, "vcluster_") {
		return "", "", ""
	}

	splitted := strings.Split(originalContext, "_")
	// vcluster_<name>_<namespace>_<context>
	if len(splitted) >= 4 {
		return splitted[1], splitted[2], strings.Join(splitted[3:], "_")
	}

	// we don't know for sure, but most likely specified custom vcluster context name
	return originalContext, "", ""
}

func findInContext(ctx context.Context, context, name, namespace string, timeout time.Duration, isParentContext bool) ([]VCluster, error) {
	vclusters := []VCluster{}
	kubeClientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{
		CurrentContext: context,
	})
	restConfig, err := kubeClientConfig.ClientConfig()
	if err != nil {
		// we can ignore this error for parent context, it just means that the kubeconfig set doesn't have parent config in it.
		if isParentContext {
			logger := log.GetInstance()
			logger.Warn("parent context unreachable - No vclusters listed from parent context")
			return vclusters, nil
		}
		return nil, errors.Wrap(err, "load kube config")
	}
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "create kube client")
	}

	// statefulset based vclusters
	statefulSets, err := getStatefulSets(ctx, kubeClient, namespace, kubeClientConfig, timeout)
	if err != nil {
		return nil, err
	}
	for _, p := range statefulSets.Items {
		if release, ok := p.Labels["release"]; ok {
			if name != "" && name != release {
				continue
			}

			var paused string

			if p.Annotations != nil {
				paused = p.Annotations[constants.PausedAnnotation]
			}
			if p.Spec.Replicas != nil && *p.Spec.Replicas == 0 && paused != "true" {
				// if the stateful set has been scaled down we'll ignore it -- this happens when
				// using devspace to do vcluster plugin dev for example, devspace scales down the
				// vcluster stateful set and re-creates a deployment for "dev mode" so we end up
				// with a duplicate vcluster in the list, one for the statefulset and one for the
				// deployment. Of course if the vcluster is paused (via `vcluster pause`), we *do*
				// still need to care about it even if replicas == 0.

				continue
			}

			vCluster, err := getVCluster(ctx, &p, context, release, kubeClient, kubeClientConfig)
			if err != nil {
				return nil, err
			}
			vCluster.Context = context
			vclusters = append(vclusters, vCluster)
		}
	}

	// deployment based vclusters
	deployments, err := getDeployments(ctx, kubeClient, namespace, kubeClientConfig, timeout)
	if err != nil {
		return nil, err
	}
	for _, p := range deployments.Items {
		if release, ok := p.Labels["release"]; ok {
			if name != "" && name != release {
				continue
			}

			vCluster, err2 := getVCluster(ctx, &p, context, release, kubeClient, kubeClientConfig)
			if err2 != nil {
				return nil, err2
			}

			vCluster.Context = context
			vclusters = append(vclusters, vCluster)
		}
	}

	return vclusters, nil
}

func getVCluster(ctx context.Context, object client.Object, context, release string, client *kubernetes.Clientset, kubeClientConfig clientcmd.ClientConfig) (VCluster, error) {
	namespace := object.GetNamespace()
	created := object.GetCreationTimestamp()
	releaseName := ""
	status := ""
	if object.GetAnnotations() != nil && object.GetAnnotations()[constants.PausedAnnotation] == "true" {
		status = string(StatusPaused)
	} else {
		releaseName = "release=" + release
	}

	if status == "" {
		pods, err := getPods(ctx, client, kubeClientConfig, namespace, releaseName)
		if err != nil {
			return VCluster{}, err
		}
		for _, pod := range pods.Items {
			status = GetPodStatus(&pod)
		}
	}
	if status == "" {
		status = string(StatusUnknown)
	}

	return VCluster{
		Name:          release,
		Namespace:     namespace,
		Status:        Status(status),
		Created:       created,
		Context:       context,
		ClientFactory: kubeClientConfig,
	}, nil
}

func getPods(ctx context.Context, client *kubernetes.Clientset, kubeClientConfig clientcmd.ClientConfig, namespace, podSelector string) (*corev1.PodList, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	podList, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: VirtualClusterSelector + "," + podSelector,
	})
	if err != nil {
		if kerrors.IsForbidden(err) {
			// try the current namespace instead
			if namespace, err = getAccessibleNS(kubeClientConfig); err != nil {
				return nil, err
			}
			return client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
				LabelSelector: VirtualClusterSelector,
			})
		}
		return nil, err
	}
	return podList, nil
}

func getDeployments(ctx context.Context, client *kubernetes.Clientset, namespace string, kubeClientConfig clientcmd.ClientConfig, timeout time.Duration) (*appsv1.DeploymentList, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	deploymentList, err := client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: VirtualClusterSelector,
	})
	if err != nil {
		if kerrors.IsForbidden(err) {
			// try the current namespace instead
			if namespace, err = getAccessibleNS(kubeClientConfig); err != nil {
				return nil, err
			}
			return client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
				LabelSelector: VirtualClusterSelector,
			})
		}
		return nil, err
	}
	return deploymentList, nil
}

func getStatefulSets(ctx context.Context, client *kubernetes.Clientset, namespace string, kubeClientConfig clientcmd.ClientConfig, timeout time.Duration) (*appsv1.StatefulSetList, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	statefulSetList, err := client.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: VirtualClusterSelector,
	})
	if err != nil {
		if kerrors.IsForbidden(err) {
			if namespace, err = getAccessibleNS(kubeClientConfig); err != nil {
				return nil, err
			}
			return client.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{
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
