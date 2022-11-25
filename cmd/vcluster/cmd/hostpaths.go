package cmd

import (
	"context"
	"fmt"
	podtranslate "github.com/loft-sh/vcluster/pkg/controllers/resources/pods/translate"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/util/blockingcacheclient"
	"github.com/loft-sh/vcluster/pkg/util/pluginhookclient"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	LogsMountPath    = "/var/log"
	PodLogsMountPath = "/var/log/pods"

	NodeIndexName                    = "spec.nodeName"
	HostpathMapperSelfNodeNameEnvVar = "VCLUSTER_HOSTPATH_MAPPER_CURRENT_NODE_NAME"

	// naming format <pod_name>_<namespace>_<container_name>-<containerdID(hash, with <docker/cri>:// prefix removed)>.log
	ContainerSymlinkSourceTemplate = "%s_%s_%s-%s.log"
)

func podNodeIndexer(obj client.Object) []string {
	res := []string{}
	pod := obj.(*corev1.Pod)
	if pod.Spec.NodeName != "" {
		res = append(res, pod.Spec.NodeName)
	}

	return res
}

// map of physical pod names to the corresponding virtual pod
type PhysicalPodMap map[string]*PodDetail

type PodDetail struct {
	Target      string
	SymLinkName *string
	PhysicalPod corev1.Pod
}

type key int

const (
	optionsKey key = iota
)

func NewHostpathMapperCommand() *cobra.Command {
	options := &context2.VirtualClusterOptions{}
	cmd := &cobra.Command{
		Use:   "maphostpaths",
		Short: "Map host to virtual pod logs",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return MapHostPaths(options)
		},
	}

	cmd.Flags().StringVar(&options.ClientCaCert, "client-ca-cert", "/data/server/tls/client-certificate", "The path to the client ca certificate")
	cmd.Flags().StringVar(&options.ServerCaCert, "server-ca-cert", "/data/server/tls/certificate-authority", "The path to the server ca certificate")
	cmd.Flags().StringVar(&options.ServerCaKey, "server-ca-key", "/data/server/tls/client-key", "The path to the server ca key")

	cmd.Flags().StringVar(&options.TargetNamespace, "target-namespace", "", "The namespace to run the virtual cluster in (defaults to current namespace)")

	cmd.Flags().StringVar(&options.Name, "name", "vcluster", "The name of the virtual cluster")

	return cmd
}

func MapHostPaths(options *context2.VirtualClusterOptions) error {
	// get current namespace
	currentNamespace, err := clienthelper.CurrentNamespace()
	if err != nil {
		return err
	}

	// ensure target namespace
	if options.TargetNamespace == "" {
		options.TargetNamespace = currentNamespace
	}
	translate.Default = translate.NewSingleNamespaceTranslator(options.TargetNamespace)

	virtualPath := fmt.Sprintf(podtranslate.VirtualPathTemplate, options.TargetNamespace, options.Name)
	options.VirtualKubeletPodPath = filepath.Join(virtualPath, "kubelet", "pods")
	options.VirtualLogsPath = filepath.Join(virtualPath, "log")
	options.VirtualPodLogsPath = filepath.Join(options.VirtualLogsPath, "pods")
	options.VirtualContainerLogsPath = filepath.Join(options.VirtualLogsPath, "containers")
	err = os.Mkdir(options.VirtualContainerLogsPath, os.ModeDir)
	if err != nil {
		if !os.IsExist(err) {
			klog.Errorf("error creating container dir in log path: %v", err)
			return err
		}
	}

	inClusterConfig := ctrl.GetConfigOrDie()

	inClusterConfig.QPS = 40
	inClusterConfig.Burst = 80
	inClusterConfig.Timeout = 0

	translate.Suffix = options.Name

	var virtualClusterConfig *rest.Config
	err = wait.PollImmediate(time.Second, time.Hour, func() (bool, error) {
		virtualClusterConfig = &rest.Config{
			Host: options.Name,
			TLSClientConfig: rest.TLSClientConfig{
				ServerName: options.Name,
				CertFile:   options.ClientCaCert,
				KeyFile:    options.ServerCaKey,
				CAFile:     options.ServerCaCert,
			},
		}

		kubeClient, err := kubernetes.NewForConfig(virtualClusterConfig)
		if err != nil {
			return false, errors.Wrap(err, "create kube client")
		}

		_, err = kubeClient.Discovery().ServerVersion()
		if err != nil {
			klog.Infof("couldn't retrieve virtual cluster version (%v), will retry in 1 seconds", err)
			return false, nil
		}
		_, err = kubeClient.CoreV1().ServiceAccounts("default").Get(context.Background(), "default", metav1.GetOptions{})
		if err != nil {
			klog.Infof("default ServiceAccount is not available yet, will retry in 1 seconds")
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return err
	}

	localManager, err := ctrl.NewManager(inClusterConfig, ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: "0",
		LeaderElection:     false,
		Namespace:          options.TargetNamespace,
		NewClient:          pluginhookclient.NewPhysicalPluginClientFactory(blockingcacheclient.NewCacheClient),
	})
	if err != nil {
		return err
	}

	virtualClusterManager, err := ctrl.NewManager(virtualClusterConfig, ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: "0",
		LeaderElection:     false,
		NewClient:          pluginhookclient.NewVirtualPluginClientFactory(blockingcacheclient.NewCacheClient),
	})
	if err != nil {
		return err
	}

	ctx := context.WithValue(context.Background(), optionsKey, options)

	startManagers(ctx, localManager, virtualClusterManager)
	mapHostPaths(ctx, localManager, virtualClusterManager)
	return nil
}

func mapHostPaths(ctx context.Context, pManager, vManager manager.Manager) {
	options := ctx.Value(optionsKey).(*context2.VirtualClusterOptions)

	wait.Forever(func() {
		podList := &corev1.PodList{}
		err := pManager.GetClient().List(ctx, podList, &client.ListOptions{
			Namespace: options.TargetNamespace,
			FieldSelector: fields.SelectorFromSet(fields.Set{
				NodeIndexName: os.Getenv(HostpathMapperSelfNodeNameEnvVar),
			}),
		})
		if err != nil {
			klog.Errorf("unable to list pods: %v", err)
			return
		}

		podMappings := make(PhysicalPodMap)

		fillUpPodMapping(ctx, podList, podMappings)

		vPodList := &corev1.PodList{}
		err = vManager.GetClient().List(ctx, vPodList, &client.ListOptions{
			FieldSelector: fields.SelectorFromSet(fields.Set{
				NodeIndexName: os.Getenv(HostpathMapperSelfNodeNameEnvVar),
			}),
		})
		if err != nil {
			klog.Errorf("unable to list pods: %v", err)
			return
		}

		existingVPodsWithNamespace := make(map[string]bool)
		existingPodsPath := make(map[string]bool)
		existingKubeletPodsPath := make(map[string]bool)

		for _, vPod := range vPodList.Items {
			existingVPodsWithNamespace[fmt.Sprintf("%s_%s", vPod.Name, vPod.Namespace)] = true
			pName := translate.Default.PhysicalName(vPod.Name, vPod.Namespace)

			if podDetail, ok := podMappings[pName]; ok {
				// create pod log symlink
				source := filepath.Join(options.VirtualPodLogsPath, fmt.Sprintf("%s_%s_%s", vPod.Namespace, vPod.Name, string(vPod.UID)))
				target := filepath.Join(podtranslate.PhysicalPodLogVolumeMountPath, podDetail.Target)

				existingPodsPath[source] = true

				_, err := createPodLogSymlinkToPhysical(source, target)
				if err != nil {
					klog.Errorf("unable to create symlink for %s: %v", podDetail.Target, err)
				}

				// create kubelet pod symlink
				kubeletPodSymlinkSource := filepath.Join(options.VirtualKubeletPodPath, string(vPod.GetUID()))
				kubeletPodSymlinkTarget := filepath.Join(podtranslate.PhysicalKubeletVolumeMountPath, string(podDetail.PhysicalPod.GetUID()))
				existingKubeletPodsPath[kubeletPodSymlinkSource] = true
				createKubeletVirtualToPhysicalPodLinks(kubeletPodSymlinkSource, kubeletPodSymlinkTarget)

				// podDetail.SymLinkName = symlinkName

				// create container to vPod symlinks
				containerSymlinkTargetDir := filepath.Join(PodLogsMountPath,
					fmt.Sprintf("%s_%s_%s", vPod.Namespace, vPod.Name, string(vPod.UID)))
				createContainerToPodSymlink(ctx, vPod, podDetail, containerSymlinkTargetDir)
			}
		}

		// cleanup old pod symlinks
		err = cleanupOldPodPath(ctx, options.VirtualPodLogsPath, existingPodsPath)
		if err != nil {
			klog.Errorf("error cleaning up old pod log paths: %v", err)
		}

		err = cleanupOldContainerPaths(ctx, existingVPodsWithNamespace)
		if err != nil {
			klog.Errorf("error cleaning up old container log paths: %v", err)
		}

		err = cleanupOldPodPath(ctx, options.VirtualKubeletPodPath, existingKubeletPodsPath)
		if err != nil {
			klog.Errorf("error cleaning up old kubelet pod paths: %v", err)
		}

		klog.Infof("successfully reconciled mapper")

	}, time.Second*5)
}

func cleanupOldContainerPaths(ctx context.Context, existingVPodsWithNS map[string]bool) error {
	options := ctx.Value(optionsKey).(*context2.VirtualClusterOptions)

	vPodsContainersOnDisk, err := os.ReadDir(options.VirtualContainerLogsPath)
	if err != nil {
		return err
	}

	for _, vPodContainerOnDisk := range vPodsContainersOnDisk {
		nameParts := strings.Split(vPodContainerOnDisk.Name(), "_")
		vPodOnDiskName, vPodOnDiskNS := nameParts[0], nameParts[1]

		if _, ok := existingVPodsWithNS[fmt.Sprintf("%s_%s", vPodOnDiskName, vPodOnDiskNS)]; !ok {
			// this pod no longer exists, hence this container
			// belonging to it should no longer exist either
			fullPathToCleanup := filepath.Join(options.VirtualContainerLogsPath, vPodContainerOnDisk.Name())

			klog.Infof("cleaning up %s", fullPathToCleanup)
			err := os.RemoveAll(fullPathToCleanup)
			if err != nil {
				klog.Errorf("error deleting symlink %s: %v", fullPathToCleanup, err)
			}
		}
	}

	return nil
}

func createKubeletVirtualToPhysicalPodLinks(vPodDirName, pPodDirName string) {
	err := os.MkdirAll(vPodDirName, os.ModeDir)
	if err != nil {
		klog.Errorf("error creating vPod kubelet directory for %s: %v", vPodDirName, err)
		return
	}

	// scan all contents in the physical pod dir
	// and create equivalent symlinks from virtual
	// path to physical
	contents, err := os.ReadDir(pPodDirName)
	if err != nil {
		klog.Errorf("error reading physical kubelet pod dir %s: %v", pPodDirName, err)
		return
	}

	for _, content := range contents {
		fullKubeletVirtualPodPath := filepath.Join(vPodDirName, content.Name())
		fullKubeletPhysicalPodPath := filepath.Join(pPodDirName, content.Name())

		err := os.Symlink(
			fullKubeletPhysicalPodPath,
			fullKubeletVirtualPodPath)
		if err != nil {
			if !os.IsExist(err) {
				klog.Errorf("error creating symlink for %s -> %s: %v", fullKubeletVirtualPodPath, fullKubeletPhysicalPodPath, err)
			}
		} else {
			klog.Infof("created kubelet pod symlink %s -> %s", fullKubeletVirtualPodPath, fullKubeletPhysicalPodPath)
		}
	}
}

func cleanupOldPodPath(ctx context.Context, cleanupDirPath string, existingPodPathsFromAPIServer map[string]bool) error {
	vPodDirsOnDisk, err := os.ReadDir(cleanupDirPath)
	if err != nil {
		return err
	}

	options := ctx.Value(optionsKey).(*context2.VirtualClusterOptions)

	for _, vPodDirOnDisk := range vPodDirsOnDisk {
		fullVPodDirDiskPath := filepath.Join(cleanupDirPath, vPodDirOnDisk.Name())
		if _, ok := existingPodPathsFromAPIServer[fullVPodDirDiskPath]; !ok {

			if cleanupDirPath == options.VirtualKubeletPodPath {
				// check if the symlinks resolve
				// this extra check for kubelet is because velero backups
				// depend on it and we don't want to delete the virtual paths
				// which the physical paths are still not cleaned up by the
				// kubelet
				symlinks, err := os.ReadDir(fullVPodDirDiskPath)
				if err != nil {
					klog.Errorf("error iterating over vpod dir %s: %v", fullVPodDirDiskPath, err)
				}

				for _, sl := range symlinks {
					target := filepath.Join(fullVPodDirDiskPath, sl.Name())
					_, readLinkErr := os.Readlink(target)
					if readLinkErr != nil {
						// symlink no longer resolves, hence delete
						klog.Infof("cleaning up %s", target)
						err := os.RemoveAll(target)
						if err != nil {
							klog.Errorf("error deleting symlink %s: %v", target, err)
						}
					}
				}
				continue
			}

			// this symlink source exists on the disk but the vPod
			// lo longer exists as per the API server, hence delete
			// the symlink
			klog.Infof("cleaning up %s", fullVPodDirDiskPath)
			err := os.RemoveAll(fullVPodDirDiskPath)
			if err != nil {
				klog.Errorf("error deleting symlink %s: %v", fullVPodDirDiskPath, err)
			}
		}
	}

	return nil
}

func createContainerToPodSymlink(ctx context.Context, vPod corev1.Pod, pPodDetail *PodDetail, targetDir string) {
	options := ctx.Value(optionsKey).(*context2.VirtualClusterOptions)

	for _, containerStatus := range vPod.Status.ContainerStatuses {
		_, containerID, _ := strings.Cut(containerStatus.ContainerID, "://")
		containerName := containerStatus.Name

		source := fmt.Sprintf(ContainerSymlinkSourceTemplate,
			vPod.Name,
			vPod.Namespace,
			containerName,
			containerID)

		pPod := pPodDetail.PhysicalPod
		physicalContainerFileName := fmt.Sprintf(ContainerSymlinkSourceTemplate,
			pPod.Name,
			pPod.Namespace,
			containerName,
			containerID)

		physicalLogFileName, err := getPhysicalLogFilename(ctx, physicalContainerFileName)
		if err != nil {
			klog.Errorf("error reading destination filename from physical container symlink: %v", err)
			continue
		}

		target := filepath.Join(targetDir, containerName, physicalLogFileName)
		source = filepath.Join(options.VirtualContainerLogsPath, source)

		err = os.Symlink(target, source)
		if err != nil {
			if !os.IsExist(err) {
				klog.Errorf("error creating container:%s to pod:%s symlink: %v", source, target, err)
			}

			continue
		}

		klog.Infof("created container:%s -> pod:%s symlink", source, target)
	}
}

// we need to get the info that which log file in the physical pod dir
// should this virtual container symlink point to. for eg.
// <physical_container> -> /var/log/pods/<pod>/<container>/xxx.log
// <virtual_container> -> <virtual_pod_path>/<container>/xxx.log
func getPhysicalLogFilename(ctx context.Context, physicalContainerFileName string) (string, error) {
	pContainerFilePath := filepath.Join(LogsMountPath, "containers", physicalContainerFileName)
	pDestination, err := os.Readlink(pContainerFilePath)
	if err != nil {
		return "", err
	}

	splits := strings.Split(pDestination, "/")
	fileName := splits[len(splits)-1]

	return fileName, nil
}

func fillUpPodMapping(ctx context.Context, pPodList *corev1.PodList, podMappings PhysicalPodMap) {
	for _, pPod := range pPodList.Items {
		lookupName := fmt.Sprintf("%s_%s_%s", pPod.Namespace, pPod.Name, pPod.UID)

		ok, err := checkIfPathExists(lookupName)
		if err != nil {
			klog.Errorf("error checking existence for path %s: %v", lookupName, err)
		}

		if ok {
			// check entry in podMapping
			if _, ok := podMappings[pPod.Name]; !ok {
				podMappings[pPod.Name] = &PodDetail{
					Target:      lookupName,
					PhysicalPod: pPod,
				}
			}
		}
	}
}

// check if folder exists
func checkIfPathExists(path string) (bool, error) {
	fullPath := filepath.Join(PodLogsMountPath, path)

	if _, err := os.Stat(fullPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func startManagers(ctx context.Context, pManager, vManager manager.Manager) {
	go func() {
		err := pManager.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()

	go func() {
		err := vManager.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()

	err := pManager.GetFieldIndexer().IndexField(ctx, &corev1.Pod{}, NodeIndexName, podNodeIndexer)
	if err != nil {
		panic(err)
	}

	err = vManager.GetFieldIndexer().IndexField(ctx, &corev1.Pod{}, NodeIndexName, podNodeIndexer)
	if err != nil {
		panic(err)
	}
}

func createPodLogSymlinkToPhysical(vPodDirName, pPodDirName string) (*string, error) {
	err := os.Symlink(pPodDirName, vPodDirName)
	if err != nil {
		if os.IsExist(err) {
			return &vPodDirName, nil
		}

		return nil, err
	}

	klog.Infof("created symlink from %s -> %s", vPodDirName, pPodDirName)
	return &vPodDirName, nil
}
