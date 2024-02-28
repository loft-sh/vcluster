package find

import (
	"context"
	"fmt"
	"strings"
	"time"

	loftclientset "github.com/loft-sh/agentapi/v3/pkg/client/loft/clientset_generated/clientset"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
	"github.com/loft-sh/log/terminal"
	"github.com/loft-sh/vcluster/pkg/procli"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const VirtualClusterSelector = "app=vcluster"

type VCluster struct {
	Name      string
	Namespace string

	Status        Status
	Created       metav1.Time
	Context       string
	Version       string
	ClientFactory clientcmd.ClientConfig `json:"-"`
}

type Status string

const (
	StatusRunning Status = "Running"
	StatusPaused  Status = "Paused"
	StatusUnknown Status = "Unknown"
)

type VclusterNotFoundError struct {
	Name string
}

func (e *VclusterNotFoundError) Error() string {
	return fmt.Sprintf("couldn't find vcluster %s", e.Name)
}

func SwitchContext(kubeConfig *clientcmdapi.Config, otherContext string) error {
	kubeConfig.CurrentContext = otherContext
	return clientcmd.ModifyConfig(clientcmd.NewDefaultClientConfigLoadingRules(), *kubeConfig, false)
}

func CurrentContext() (string, *clientcmdapi.Config, error) {
	rawConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).RawConfig()
	if err != nil {
		return "", nil, err
	}

	return rawConfig.CurrentContext, &rawConfig, nil
}

func GetVCluster(ctx context.Context, proClient procli.Client, context, name, namespace, project string, log log.Logger) (*VCluster, *procli.VirtualClusterInstanceProject, error) {
	if name == "" {
		return nil, nil, fmt.Errorf("please specify a name")
	}

	// list vclusters
	ossVClusters, proVClusters, err := ListVClusters(ctx, proClient, context, name, namespace, project, log)
	if err != nil {
		return nil, nil, err
	}

	// figure out what we want to return
	if len(ossVClusters) == 0 && len(proVClusters) == 0 {
		return nil, nil, &VclusterNotFoundError{Name: name}
	} else if len(ossVClusters) == 1 && len(proVClusters) == 0 {
		return &ossVClusters[0], nil, nil
	} else if len(proVClusters) == 1 && len(ossVClusters) == 0 {
		return nil, &proVClusters[0], nil
	}

	// check if terminal
	if !terminal.IsTerminalIn {
		return nil, nil, fmt.Errorf("multiple vclusters with name %s found, please specify a project via --project or a namespace via --namespace to select the correct one", name)
	}

	// ask a question
	questionOptionsUnformatted := [][]string{}
	for _, vCluster := range ossVClusters {
		questionOptionsUnformatted = append(questionOptionsUnformatted, []string{name, vCluster.Namespace, "false"})
	}
	for _, vCluster := range proVClusters {
		questionOptionsUnformatted = append(questionOptionsUnformatted, []string{name, vCluster.Project.Name, "true"})
	}
	questionOptions := FormatOptions("Name: %s | Namespace / Project: %s | Pro: %s ", questionOptionsUnformatted)
	selectedVCluster, err := log.Question(&survey.QuestionOptions{
		Question:     "Please choose a virtual cluster to use",
		DefaultValue: questionOptions[0],
		Options:      questionOptions,
	})
	if err != nil {
		return nil, nil, err
	}

	// match answer
	for idx, s := range questionOptions {
		if s == selectedVCluster {
			if idx < len(ossVClusters) {
				return &ossVClusters[idx], nil, nil
			}

			return nil, &proVClusters[idx-len(ossVClusters)], nil
		}
	}

	return nil, nil, fmt.Errorf("unexpected error searching for selected vcluster")
}

func FormatOptions(format string, options [][]string) []string {
	if len(options) == 0 {
		return []string{}
	}

	columnLengths := make([]int, len(options[0]))
	for _, row := range options {
		for i, column := range row {
			if len(column) > columnLengths[i] {
				columnLengths[i] = len(column)
			}
		}
	}

	retOptions := []string{}
	for _, row := range options {
		columns := []interface{}{}
		for i := range row {
			value := row[i]
			if columnLengths[i] > len(value) {
				value = value + strings.Repeat(" ", columnLengths[i]-len(value))
			}

			columns = append(columns, value)
		}

		retOptions = append(retOptions, fmt.Sprintf(format, columns...))
	}

	return retOptions
}

func ListVClusters(ctx context.Context, proClient procli.Client, context, name, namespace, project string, log log.Logger) ([]VCluster, []procli.VirtualClusterInstanceProject, error) {
	var err error
	if context == "" {
		var err error
		context, _, err = CurrentContext()
		if err != nil {
			return nil, nil, err
		}
	}

	var ossVClusters []VCluster
	if project == "" {
		ossVClusters, err = ListOSSVClusters(ctx, context, name, namespace)
		if err != nil {
			log.Warnf("Error retrieving vclusters: %v", err)
		}
	}

	var proVClusters []procli.VirtualClusterInstanceProject
	if proClient != nil {
		proVClusters, err = procli.ListVClusters(ctx, proClient, name, project)
		if err != nil {
			log.Warnf("Error retrieving pro vclusters: %v", err)
		}
	}

	return ossVClusters, proVClusters, nil
}

func ListOSSVClusters(ctx context.Context, context, name, namespace string) ([]VCluster, error) {
	var err error

	timeout := time.Minute
	vClusterName, _, vClusterContext := VClusterProFromContext(context)
	if vClusterContext != "" {
		timeout = time.Second * 10
	} else {
		vClusterName, _, vClusterContext = VClusterFromContext(context)
		if vClusterName != "" {
			timeout = time.Second * 5
		}
	}

	vclusters, err := findInContext(ctx, context, name, namespace, timeout, false)
	if err != nil && vClusterName == "" {
		return nil, errors.Wrap(err, "find vcluster")
	}

	if vClusterName != "" {
		parentContextVClusters, err := findInContext(ctx, vClusterContext, name, namespace, time.Minute, true)
		if err != nil {
			return nil, errors.Wrap(err, "find vcluster")
		}

		vclusters = append(vclusters, parentContextVClusters...)
	}

	return vclusters, nil
}

func VClusterProContextName(vClusterName string, projectName string, currentContext string) string {
	return "vcluster-pro_" + vClusterName + "_" + projectName + "_" + currentContext
}

func VClusterContextName(vClusterName string, vClusterNamespace string, currentContext string) string {
	return "vcluster_" + vClusterName + "_" + vClusterNamespace + "_" + currentContext
}

func VClusterConnectBackgroundProxyName(vClusterName string, vClusterNamespace string, currentContext string) string {
	return VClusterContextName(vClusterName, vClusterNamespace, currentContext) + "_background_proxy"
}

func VClusterProFromContext(originalContext string) (name string, project string, context string) {
	if !strings.HasPrefix(originalContext, "vcluster-pro_") {
		return "", "", ""
	}

	splitted := strings.Split(originalContext, "_")
	// vcluster-pro_<name>_<namespace>_<context>
	if len(splitted) >= 4 {
		return splitted[1], splitted[2], strings.Join(splitted[3:], "_")
	}

	// we don't know for sure, but most likely specified custom vcluster context name
	return originalContext, "", ""
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
	loftClient, err := loftclientset.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "create new loft api client")
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

			// skip pro clusters
			virtualCluster, err := loftClient.StorageV1().VirtualClusters(p.Namespace).Get(ctx, p.Name, metav1.GetOptions{})
			if err == nil && (virtualCluster.Annotations == nil || virtualCluster.Annotations["loft.sh/skip-helm-deploy"] != "true") {
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

			// skip pro clusters
			virtualCluster, err := loftClient.StorageV1().VirtualClusters(p.Namespace).Get(ctx, p.Name, metav1.GetOptions{})
			if err == nil && (virtualCluster.Annotations == nil || virtualCluster.Annotations["loft.sh/skip-helm-deploy"] != "true") {
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
	version := ""

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

	switch vclusterObject := object.(type) {
	case *appsv1.StatefulSet:
		for _, container := range vclusterObject.Spec.Template.Spec.Containers {
			if container.Name == "syncer" {
				tag := strings.Split(container.Image, ":")
				if len(tag) == 2 {
					version = tag[1]
				}
				break
			}
		}
	case *appsv1.Deployment:
		for _, container := range vclusterObject.Spec.Template.Spec.Containers {
			if container.Name == "syncer" {
				tag := strings.Split(container.Image, ":")
				if len(tag) == 2 {
					version = tag[1]
				}
				break
			}
		}
	}

	return VCluster{
		Name:          release,
		Namespace:     namespace,
		Status:        Status(status),
		Created:       created,
		Context:       context,
		Version:       version,
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
