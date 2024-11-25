package debug

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/pods/translate"
	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/loft-sh/vcluster/pkg/util/clihelper"
	"github.com/loft-sh/vcluster/pkg/util/portforward"
	"github.com/loft-sh/vcluster/pkg/util/stringutil"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"
)

var defaultHostResources = []string{
	"serviceaccounts",
	"pods",
	"events",
	"statefulsets",
	"networkpolicies",
	"deployments",
	"replicasets",
	"endpoints",
	"endpointslices",
	"services",
	"ingresses",
	"configmaps",
	"pvc",
}

var defaultVirtualResources = []string{
	"serviceaccounts",
	"pods",
	"events",
	"statefulsets",
	"networkpolicies",
	"deployments",
	"replicasets",
	"endpoints",
	"endpointslices",
	"services",
	"ingresses",
	"configmaps",
	"pvc",
	"namespaces",
	"persistentvolumes",
	"ingressclasses",
	"storageclasses",
	"priorityclasses",
	"runtimeclasses",
	"customresourcedefinitions",
	"apiservices",
}

type CollectCmd struct {
	*flags.GlobalFlags
	log log.Logger

	Release bool
	Logs    bool

	HostInfo      bool
	HostResources []string

	VirtualInfo      bool
	VirtualResources []string

	CountVirtualClusterObjects bool

	OutputFilename string
}

func NewCollectCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &CollectCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "collect",
		Short: "Collects debugging information from the vCluster",
		Long: `##############################################################
################### vcluster debug collect ###################
##############################################################
Collects debugging information from the vCluster

Examples:
vcluster debug collect
##############################################################
	`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			newArgs, err := util.PromptForArgs(cmd.log, args, "vcluster name")
			if err != nil {
				switch {
				case errors.Is(err, util.ErrNonInteractive):
					if err := util.VClusterNameOnlyValidator(cobraCmd, args); err != nil {
						return err
					}
				default:
					return err
				}
			}

			return cmd.Run(cobraCmd.Context(), newArgs)
		}}

	cobraCmd.Flags().BoolVar(&cmd.Release, "release", true, "Collect vCluster release info")
	cobraCmd.Flags().BoolVar(&cmd.Logs, "logs", true, "Collect vCluster logs")
	cobraCmd.Flags().BoolVar(&cmd.VirtualInfo, "virtual-info", true, "Collect virtual cluster info")
	cobraCmd.Flags().StringSliceVar(&cmd.VirtualResources, "virtual-resources", []string{}, "Collect virtual cluster resources")
	cobraCmd.Flags().BoolVar(&cmd.HostInfo, "host-info", true, "Collect host cluster info")
	cobraCmd.Flags().BoolVar(&cmd.CountVirtualClusterObjects, "count-virtual-cluster-objects", true, "Collect how many objects are in the vCluster")
	cobraCmd.Flags().StringVar(&cmd.OutputFilename, "output-filename", "", "If specified, will write to the given filename")
	cobraCmd.Flags().StringSliceVar(&cmd.HostResources, "host-resources", []string{}, "Collect host resources in vCluster namespace")

	return cobraCmd
}

func (cmd *CollectCmd) Run(ctx context.Context, args []string) error {
	// gather resources
	cmd.HostResources = mergeResources(defaultHostResources, cmd.HostResources)
	cmd.VirtualResources = mergeResources(defaultVirtualResources, cmd.VirtualResources)

	// find vCluster
	vClusterName := args[0]
	vCluster, err := find.GetVCluster(ctx, cmd.Context, vClusterName, cmd.Namespace, cmd.log)
	if err != nil {
		return fmt.Errorf("failed to find vCluster %s: %w", vClusterName, err)
	}

	// get kube config
	kubeConfig, err := getKubeConfig(vCluster, cmd.GlobalFlags)
	if err != nil {
		return err
	}

	// create kube client
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}

	// create temp dir
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// get release info
	if cmd.Release {
		cmd.log.Info("Get vCluster release information")
		err = cmd.getRelease(ctx, kubeClient, vCluster, tempDir)
		if err != nil {
			return fmt.Errorf("failed to get vCluster release information: %w", err)
		}
	}

	// get logs
	if cmd.Logs {
		cmd.log.Info("Get vCluster logs...")
		err = cmd.getLogs(ctx, kubeClient, vCluster, tempDir)
		if err != nil {
			return fmt.Errorf("failed to get logs: %w", err)
		}
	}

	// get host info
	if cmd.HostInfo {
		cmd.log.Infof("Get host cluster information")
		err = cmd.getHostInfo(kubeClient, tempDir)
		if err != nil {
			return fmt.Errorf("failed to get host cluster information: %w", err)
		}
	}

	// get host resources
	if len(cmd.HostResources) > 0 {
		cmd.log.Infof("Collect host cluster resources")
		err = cmd.getHostResources(ctx, kubeConfig, vCluster, tempDir)
		if err != nil {
			return fmt.Errorf("failed to get host resources: %w", err)
		}
	}

	// collect virtual cluster
	if cmd.VirtualInfo || len(cmd.VirtualResources) > 0 || cmd.CountVirtualClusterObjects {
		// get virtual kube config & client
		vKubeConfig, err := cmd.getVClusterKubeConfig(ctx, kubeConfig, kubeClient, vCluster)
		if err != nil {
			return fmt.Errorf("failed to get virtual cluster config: %w", err)
		}
		vKubeClient, err := kubernetes.NewForConfig(vKubeConfig)
		if err != nil {
			return fmt.Errorf("failed to build virtual cluster client: %w", err)
		}

		// get virtual info
		if cmd.VirtualInfo {
			cmd.log.Infof("Get virtual cluster information")
			err = cmd.getVirtualInfo(vKubeClient, tempDir)
			if err != nil {
				return fmt.Errorf("failed to get virtual cluster information: %w", err)
			}
		}

		// get virtual resources
		if len(cmd.VirtualResources) > 0 {
			cmd.log.Infof("Collect virtual cluster resources")
			err = cmd.getVirtualResources(ctx, vKubeConfig, tempDir)
			if err != nil {
				return fmt.Errorf("failed to get virtual resources: %w", err)
			}
		}

		// count virtual resources
		if cmd.CountVirtualClusterObjects {
			cmd.log.Infof("Count virtual cluster resources")
			err = cmd.countVirtualResources(ctx, vKubeConfig, tempDir)
			if err != nil {
				return fmt.Errorf("failed to count virtual resources: %w", err)
			}
		}
	}

	// compress
	if cmd.OutputFilename == "" {
		cmd.OutputFilename = "vcluster." + vClusterName + ".debug." + strconv.FormatInt(time.Now().Unix(), 10) + ".tar.gz"
	}
	err = compress(tempDir, cmd.OutputFilename)
	if err != nil {
		return err
	}

	cmd.log.Donef("Wrote debug information to file %s", cmd.OutputFilename)
	return nil
}

func (cmd *CollectCmd) getVClusterKubeConfig(ctx context.Context, kubeConfig *rest.Config, kubeClient *kubernetes.Clientset, vCluster *find.VCluster) (*rest.Config, error) {
	var err error
	podName := ""
	waitErr := wait.PollUntilContextTimeout(ctx, time.Second, time.Second*30, true, func(ctx context.Context) (bool, error) {
		// get vcluster pod name
		var pods *corev1.PodList
		pods, err = kubeClient.CoreV1().Pods(vCluster.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: "app=vcluster,release=" + vCluster.Name,
		})
		if err != nil {
			return false, err
		} else if len(pods.Items) == 0 {
			err = fmt.Errorf("can't find a running vcluster pod in namespace %s", cmd.Namespace)
			cmd.log.Debugf("can't find a running vcluster pod in namespace %s", cmd.Namespace)
			return false, nil
		}

		// sort by newest
		sort.Slice(pods.Items, func(i, j int) bool {
			return pods.Items[i].CreationTimestamp.Unix() > pods.Items[j].CreationTimestamp.Unix()
		})
		if pods.Items[0].DeletionTimestamp != nil {
			err = fmt.Errorf("can't find a running vcluster pod in namespace %s", cmd.Namespace)
			cmd.log.Debugf("can't find a running vcluster pod in namespace %s", cmd.Namespace)
			return false, nil
		}

		podName = pods.Items[0].Name
		return true, nil
	})
	if waitErr != nil {
		return nil, fmt.Errorf("finding vcluster pod: %w - %w", waitErr, err)
	}

	cmd.log.Infof("Start port-forwarding to virtual cluster")
	vKubeConfig, err := clihelper.GetKubeConfig(ctx, kubeClient, vCluster.Name, vCluster.Namespace, cmd.log)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kube config: %w", err)
	}

	// silence port-forwarding if a command is used
	stdout := io.Writer(os.Stdout)
	stderr := io.Writer(os.Stderr)
	localPort := clihelper.RandomPort()
	errorChan := make(chan error)
	go func() {
		errorChan <- portforward.StartPortForwardingWithRestart(ctx, kubeConfig, "127.0.0.1", podName, cmd.Namespace, strconv.Itoa(localPort), "8443", make(chan struct{}), stdout, stderr, cmd.log)
	}()

	for _, cluster := range vKubeConfig.Clusters {
		if cluster == nil {
			continue
		}
		cluster.Server = "https://localhost:" + strconv.Itoa(localPort)
	}

	restConfig, err := clientcmd.NewDefaultClientConfig(*vKubeConfig, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create rest client config: %w", err)
	}

	vKubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create vcluster client: %w", err)
	}

	err = wait.PollUntilContextTimeout(ctx, time.Millisecond*200, time.Minute*3, true, func(ctx context.Context) (bool, error) {
		select {
		case err := <-errorChan:
			return false, err
		default:
			// check if service account exists
			_, err = vKubeClient.CoreV1().ServiceAccounts("default").Get(ctx, "default", metav1.GetOptions{})
			return err == nil, nil
		}
	})
	if err != nil {
		return nil, fmt.Errorf("wait for vcluster to become ready: %w", err)
	}

	return restConfig, nil
}

func (cmd *CollectCmd) getVirtualInfo(kubeClient kubernetes.Interface, targetDir string) error {
	virtualDir := filepath.Join(targetDir, "virtual")
	err := os.MkdirAll(virtualDir, 0777)
	if err != nil {
		return err
	}

	return getVersion(kubeClient, virtualDir)
}

func (cmd *CollectCmd) getVirtualResources(ctx context.Context, kubeConfig *rest.Config, targetDir string) error {
	virtualDir := filepath.Join(targetDir, "virtual")
	err := os.MkdirAll(virtualDir, 0777)
	if err != nil {
		return err
	}

	return cmd.getResources(ctx, kubeConfig, "", virtualDir, cmd.VirtualResources)
}

func (cmd *CollectCmd) getHostInfo(kubeClient kubernetes.Interface, targetDir string) error {
	hostDir := filepath.Join(targetDir, "host")
	err := os.MkdirAll(hostDir, 0777)
	if err != nil {
		return err
	}

	return getVersion(kubeClient, hostDir)
}

func (cmd *CollectCmd) getHostResources(ctx context.Context, kubeConfig *rest.Config, vCluster *find.VCluster, targetDir string) error {
	hostDir := filepath.Join(targetDir, "host")
	err := os.MkdirAll(hostDir, 0777)
	if err != nil {
		return err
	}

	return cmd.getResources(ctx, kubeConfig, vCluster.Namespace, hostDir, cmd.HostResources)
}

func (cmd *CollectCmd) getResources(ctx context.Context, kubeConfig *rest.Config, namespace, targetDir string, groupVersionResources []string) error {
	dynamicClient, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}

	_, apiResources, err := kubeClient.Discovery().ServerGroupsAndResources()
	if err != nil {
		return fmt.Errorf("unable to get discovery information: %w", err)
	}

	resourcesDir := filepath.Join(targetDir, "resources")
	err = os.MkdirAll(resourcesDir, 0755)
	if err != nil {
		return err
	}

	for _, groupVersionResourceStr := range groupVersionResources {
		groupVersionResource := resourceToGroupVersionResource(groupVersionResourceStr, apiResources)
		if groupVersionResource.Resource == "" {
			return fmt.Errorf("couldn't find group schema version for resource %s", groupVersionResourceStr)
		}

		cmd.log.Infof("Retrieve %s...", groupVersionResource.Resource)
		resourceList, err := dynamicClient.Resource(groupVersionResource).Namespace(namespace).List(ctx, metav1.ListOptions{
			Limit: int64(2000),
		})
		if err != nil {
			return fmt.Errorf("failed to list resource %s in namespace %s: %w", groupVersionResource.String(), namespace, err)
		}

		resourceDir := filepath.Join(resourcesDir, groupVersionResource.Resource)
		if groupVersionResource.Group != "" {
			resourceDir = filepath.Join(resourcesDir, groupVersionResource.Resource+"."+groupVersionResource.Group)
		}

		err = os.MkdirAll(resourceDir, 0755)
		if err != nil {
			return err
		}

		for _, resource := range resourceList.Items {
			if groupVersionResource.Resource == "pods" {
				annotations := resource.GetAnnotations()
				for k := range annotations {
					if strings.HasPrefix(k, translate.ServiceAccountTokenAnnotation) {
						delete(annotations, k)
					}
				}
				resource.SetAnnotations(annotations)
			}

			rawResource, err := yaml.Marshal(resource.Object)
			if err != nil {
				return fmt.Errorf("failed to marshal resource %s in namespace %s: %w", groupVersionResource.String(), namespace, err)
			}

			filename := filepath.Join(resourceDir, resource.GetName())
			if resource.GetNamespace() != "" {
				filename = filepath.Join(resourceDir, resource.GetNamespace(), resource.GetName())
			}
			filename += ".yaml"

			err = os.MkdirAll(filepath.Dir(filename), 0777)
			if err != nil {
				return err
			}

			err = os.WriteFile(filename, rawResource, 0644)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func resourceToGroupVersionResource(resource string, apiResources []*metav1.APIResourceList) schema.GroupVersionResource {
	apiGroup := ""
	splittedResource := strings.SplitN(resource, ".", 2)
	if len(splittedResource) == 2 {
		resource = splittedResource[0]
		apiGroup = splittedResource[1]
	}

	// find the resource in the api resource list
	for _, apiResourceGroup := range apiResources {
		groupVersion, err := schema.ParseGroupVersion(apiResourceGroup.GroupVersion)
		if err != nil {
			continue
		} else if apiGroup != "" && apiGroup != groupVersion.Group {
			continue
		}

		// check if resource matches
		for _, apiResource := range apiResourceGroup.APIResources {
			if apiResource.Name == resource {
				return schema.GroupVersionResource{
					Group:    groupVersion.Group,
					Version:  groupVersion.Version,
					Resource: apiResource.Name,
				}
			}

			if apiResource.SingularName == resource {
				return schema.GroupVersionResource{
					Group:    groupVersion.Group,
					Version:  groupVersion.Version,
					Resource: apiResource.Name,
				}
			}

			for _, shortName := range apiResource.ShortNames {
				if shortName == resource {
					return schema.GroupVersionResource{
						Group:    groupVersion.Group,
						Version:  groupVersion.Version,
						Resource: apiResource.Name,
					}
				}
			}
		}
	}

	return schema.GroupVersionResource{}
}

func getVersion(kubeClient kubernetes.Interface, targetDir string) error {
	serverVersion, err := kubeClient.Discovery().ServerVersion()
	if err != nil {
		return err
	}

	versionRaw, err := json.MarshalIndent(serverVersion, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(targetDir, "version.json"), versionRaw, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (cmd *CollectCmd) getRelease(ctx context.Context, kubeClient kubernetes.Interface, vCluster *find.VCluster, targetDir string) error {
	release, err := helm.NewSecrets(kubeClient).Get(ctx, vCluster.Name, vCluster.Namespace)
	if err != nil {
		if kerrors.IsNotFound(err) {
			cmd.log.Info("Couldn't find vCluster release")
			return nil
		}

		return fmt.Errorf("getting vCluster release %s: %w", vCluster.Name, err)
	}

	rawRelease, err := json.MarshalIndent(release, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling vCluster release %s: %w", vCluster.Name, err)
	}

	return os.WriteFile(filepath.Join(targetDir, "release.json"), rawRelease, 0644)
}

func (cmd *CollectCmd) getLogs(ctx context.Context, kubeClient kubernetes.Interface, vCluster *find.VCluster, targetDir string) error {
	logsDir := filepath.Join(targetDir, "logs")
	err := os.MkdirAll(logsDir, 0755)
	if err != nil {
		return err
	}

	podList, err := kubeClient.CoreV1().Pods(vCluster.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=vcluster,release=" + vCluster.Name,
	})
	if err != nil {
		return fmt.Errorf("failed to list pods in vCluster %s: %w", vCluster.Name, err)
	}

	for _, pod := range podList.Items {
		// retrieve logs
		cmd.log.Infof("Capture pod %s/%s logs", pod.Namespace, pod.Name)
		logs, err := kubeClient.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{}).Do(ctx).Raw()
		if err != nil {
			// write the error to the file
			err = os.WriteFile(filepath.Join(logsDir, pod.Name+".error.log"), []byte(err.Error()), 0644)
			if err != nil {
				return err
			}
		} else {
			// write the error to the file
			err = os.WriteFile(filepath.Join(logsDir, pod.Name+".log"), logs, 0644)
			if err != nil {
				return err
			}
		}

		// check if we need to get previous logs as well
		if len(pod.Status.ContainerStatuses) > 0 && pod.Status.ContainerStatuses[0].RestartCount > 0 {
			cmd.log.Infof("Capture previous pod %s/%s logs", pod.Namespace, pod.Name)
			logs, err := kubeClient.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{Previous: true}).Do(ctx).Raw()
			if err != nil {
				// write the error to the file
				err = os.WriteFile(filepath.Join(logsDir, pod.Name+".previous.error.log"), []byte(err.Error()), 0644)
				if err != nil {
					return err
				}
			} else {
				// write the error to the file
				err = os.WriteFile(filepath.Join(logsDir, pod.Name+".previous.log"), logs, 0644)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (cmd *CollectCmd) countVirtualResources(ctx context.Context, kubeConfig *rest.Config, targetDir string) error {
	virtualDir := filepath.Join(targetDir, "virtual")
	err := os.MkdirAll(virtualDir, 0777)
	if err != nil {
		return err
	}

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}

	_, apiResources, err := kubeClient.Discovery().ServerGroupsAndResources()
	if err != nil {
		return fmt.Errorf("unable to get discovery information: %w", err)
	}

	metaClient, err := metadata.NewForConfig(kubeConfig)
	if err != nil {
		return fmt.Errorf("error creating metadata client: %w", err)
	}

	retMap := map[string]interface{}{}
	for _, apiGroup := range apiResources {
		groupVersion, err := schema.ParseGroupVersion(apiGroup.GroupVersion)
		if err != nil {
			continue
		}

		for _, apiResource := range apiGroup.APIResources {
			// exclude api resources that cannot be listed and watched
			if !stringutil.Contains(apiResource.Verbs, "get") ||
				!stringutil.Contains(apiResource.Verbs, "watch") ||
				!stringutil.Contains(apiResource.Verbs, "list") ||
				!stringutil.Contains(apiResource.Verbs, "create") ||
				!stringutil.Contains(apiResource.Verbs, "update") ||
				!stringutil.Contains(apiResource.Verbs, "patch") ||
				!stringutil.Contains(apiResource.Verbs, "delete") {
				continue
			}

			resource := schema.GroupVersionResource{
				Group:    groupVersion.Group,
				Version:  groupVersion.Version,
				Resource: apiResource.Name,
			}
			cmd.log.Infof("Count %s", resource.Resource)
			objectList, err := metaClient.Resource(resource).List(ctx, metav1.ListOptions{Limit: int64(6000)})
			if err != nil {
				cmd.log.Errorf("Error listing %s: %v", resource.String(), err)
				continue
			}

			names := []string{}
			for _, object := range objectList.Items {
				if object.Namespace != "" {
					names = append(names, object.Namespace+"/"+object.Name)
				} else {
					names = append(names, object.Name)
				}
			}

			retMap[resource.String()] = map[string]interface{}{
				"count": len(names),
				"names": names,
			}
		}
	}

	raw, err := json.MarshalIndent(retMap, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal object counts: %w", err)
	}

	err = os.WriteFile(filepath.Join(virtualDir, "virtual-resource-count.json"), raw, 0644)
	if err != nil {
		return fmt.Errorf("write virtual resource count: %w", err)
	}

	return nil
}

func compress(folder, target string) error {
	fileToWrite, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer fileToWrite.Close()

	zr := gzip.NewWriter(fileToWrite)
	defer zr.Close()
	tw := tar.NewWriter(zr)
	defer tw.Close()

	// walk through every file in the folder
	return filepath.Walk(folder, func(file string, fi os.FileInfo, _ error) error {
		// generate tar header
		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return err
		}

		// must provide real name
		// (see https://golang.org/src/archive/tar/common.go?#L626)
		header.Name = strings.TrimPrefix(filepath.ToSlash(file), filepath.ToSlash(folder))
		if header.Name == "" {
			return nil
		}

		// write header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// if not a dir, write file content
		if !fi.IsDir() {
			data, err := os.Open(file)
			if err != nil {
				return err
			}
			defer data.Close()
			if _, err := io.Copy(tw, data); err != nil {
				return err
			}
		}
		return nil
	})
}

func getKubeConfig(vCluster *find.VCluster, globalFlags *flags.GlobalFlags) (*rest.Config, error) {
	// load the rest config
	kubeConfig, err := vCluster.ClientFactory.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("there is an error loading your current kube config (%w), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}

	currentContext, currentRawConfig, err := find.CurrentContext()
	if err != nil {
		return nil, err
	}

	vClusterName, vClusterNamespace, vClusterContext := find.VClusterFromContext(currentContext)
	if vClusterName == vCluster.Name && vClusterNamespace == vCluster.Namespace && vClusterContext == vCluster.Context {
		err = find.SwitchContext(currentRawConfig, vCluster.Context)
		if err != nil {
			return nil, err
		}
	}

	globalFlags.Namespace = vCluster.Namespace
	return kubeConfig, nil
}

func mergeResources(defaultResources, userResources []string) []string {
	retResources := []string{}
	for _, resource := range defaultResources {
		if stringutil.Contains(userResources, "-"+resource) {
			continue
		}

		retResources = append(retResources, resource)
	}

	for _, resource := range userResources {
		if strings.HasPrefix(resource, "-") || stringutil.Contains(retResources, resource) {
			continue
		}

		retResources = append(retResources, resource)
	}

	return retResources
}
