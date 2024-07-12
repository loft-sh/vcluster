package find

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
	"github.com/loft-sh/log/terminal"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/sleepmode"
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
	Name          string
	Namespace     string
	Annotations   map[string]string
	Labels        map[string]string
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

type VClusterNotFoundError struct {
	Name string
}

func (e *VClusterNotFoundError) Error() string {
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

func GetPlatformVCluster(ctx context.Context, platformClient platform.Client, name, project string, log log.Logger) (*platform.VirtualClusterInstanceProject, error) {
	platformVClusters, err := platform.ListVClusters(ctx, platformClient, name, project)
	if err != nil {
		log.Warnf("Error retrieving platform vclusters: %v", err)
	}

	// figure out what we want to return
	if len(platformVClusters) == 0 {
		return nil, &VClusterNotFoundError{Name: name}
	} else if len(platformVClusters) == 1 {
		return platformVClusters[0], nil
	}

	// check if terminal
	if !terminal.IsTerminalIn {
		return nil, fmt.Errorf("multiple vclusters with name %s found, please specify a project via --project to select the correct one", name)
	}

	// ask a question
	questionOptionsUnformatted := [][]string{}
	for _, vCluster := range platformVClusters {
		questionOptionsUnformatted = append(questionOptionsUnformatted, []string{name, vCluster.Project.Name})
	}
	questionOptions := FormatOptions("Name: %s | Project: %s", questionOptionsUnformatted)
	selectedVCluster, err := log.Question(&survey.QuestionOptions{
		Question:     "Please choose a virtual cluster to use",
		DefaultValue: questionOptions[0],
		Options:      questionOptions,
	})
	if err != nil {
		return nil, err
	}

	// match answer
	for idx, s := range questionOptions {
		if s == selectedVCluster {
			return platformVClusters[idx], nil
		}
	}

	return nil, fmt.Errorf("unexpected error searching for selected virtual cluster")
}

func GetVCluster(ctx context.Context, context, name, namespace string, log log.Logger) (*VCluster, error) {
	if name == "" {
		return nil, fmt.Errorf("please specify a name")
	}

	// list virtual clusters
	ossVClusters, err := ListVClusters(ctx, context, name, namespace, log)
	if err != nil {
		return nil, err
	}

	// figure out what we want to return
	if len(ossVClusters) == 0 {
		return nil, &VClusterNotFoundError{Name: name}
	} else if len(ossVClusters) == 1 {
		return &ossVClusters[0], nil
	}

	// check if terminal
	if !terminal.IsTerminalIn {
		return nil, fmt.Errorf("multiple vclusters with name %s found, please specify a namespace via --namespace to select the correct one", name)
	}

	// ask a question
	questionOptionsUnformatted := [][]string{}
	for _, vCluster := range ossVClusters {
		questionOptionsUnformatted = append(questionOptionsUnformatted, []string{name, vCluster.Namespace})
	}
	questionOptions := FormatOptions("Name: %s | Namespace: %s", questionOptionsUnformatted)
	selectedVCluster, err := log.Question(&survey.QuestionOptions{
		Question:     "Please choose a virtual cluster to use",
		DefaultValue: questionOptions[0],
		Options:      questionOptions,
	})
	if err != nil {
		return nil, err
	}

	// match answer
	for idx, s := range questionOptions {
		if s == selectedVCluster {
			return &ossVClusters[idx], nil
		}
	}

	return nil, fmt.Errorf("unexpected error searching for selected virtual cluster")
}

func (v *VCluster) IsSleeping() bool {
	return sleepmode.IsSleeping(v)
}

// GetAnnotations implements Annotated
func (v *VCluster) GetAnnotations() map[string]string {
	return v.Annotations
}

// GetLabels implements Labeled
func (v *VCluster) GetLabels() map[string]string {
	return v.Labels
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

func ListVClusters(ctx context.Context, context, name, namespace string, log log.Logger) ([]VCluster, error) {
	var err error
	if context == "" {
		var err error
		context, _, err = CurrentContext()
		if err != nil {
			return nil, err
		}
	}

	ossVClusters, err := ListOSSVClusters(ctx, context, name, namespace)
	if err != nil {
		log.Warnf("Error retrieving vclusters: %v", err)
	}

	return ossVClusters, nil
}

func ListOSSVClusters(ctx context.Context, context, name, namespace string) ([]VCluster, error) {
	var err error

	timeout := time.Minute
	vClusterName, _, vClusterContext := VClusterFromContext(context)
	if vClusterName != "" {
		timeout = time.Second * 5
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

func VClusterContextName(vClusterName string, vClusterNamespace string, currentContext string) string {
	return "vcluster_" + vClusterName + "_" + vClusterNamespace + "_" + currentContext
}

func VClusterPlatformContextName(vClusterName string, projectName string, currentContext string) string {
	return "vcluster-platform_" + vClusterName + "_" + projectName + "_" + currentContext
}

func VClusterPlatformFromContext(originalContext string) (name string, project string, context string) {
	if !strings.HasPrefix(originalContext, "vcluster-platform_") {
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

var NonAllowedCharactersRegEx = regexp.MustCompile(`[^a-zA-Z0-9\-_]+`)

func VClusterConnectBackgroundProxyName(vClusterName string, vClusterNamespace string, currentContext string) string {
	return NonAllowedCharactersRegEx.ReplaceAllString(VClusterContextName(vClusterName, vClusterNamespace, currentContext)+"_background_proxy", "")
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

			if p.Spec.Replicas != nil && *p.Spec.Replicas == 0 && !isPaused(&p) {
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
		Annotations:   object.GetAnnotations(),
		Labels:        object.GetLabels(),
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

func isPaused(v client.Object) bool {
	annotations := v.GetAnnotations()
	labels := v.GetLabels()

	return annotations[constants.PausedAnnotation] == "true" || labels[sleepmode.Label] == "true"
}
