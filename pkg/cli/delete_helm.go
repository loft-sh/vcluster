package cli

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/ghodss/yaml"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/localkubernetes"
	"github.com/loft-sh/vcluster/pkg/coredns"
	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/util/clihelper"
	"github.com/loft-sh/vcluster/pkg/util/helmdownloader"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type DeleteOptions struct {
	Driver string

	Project             string
	Wait                bool
	KeepPVC             bool
	DeleteNamespace     bool
	DeleteContext       bool
	DeleteConfigMap     bool
	AutoDeleteNamespace bool
	IgnoreNotFound      bool
}

type deleteHelm struct {
	*flags.GlobalFlags
	*DeleteOptions

	rawConfig  *clientcmdapi.Config
	restConfig *rest.Config
	kubeClient *kubernetes.Clientset

	log log.Logger
}

func DeleteHelm(ctx context.Context, platformClient platform.Client, options *DeleteOptions, globalFlags *flags.GlobalFlags, vClusterName string, log log.Logger) error {
	cmd := deleteHelm{
		GlobalFlags:   globalFlags,
		DeleteOptions: options,
		log:           log,
	}

	// find vcluster
	vCluster, err := find.GetVCluster(ctx, cmd.Context, vClusterName, cmd.Namespace, cmd.log)
	if err != nil {
		if !cmd.IgnoreNotFound {
			return err
		}

		var errorNotFound *find.VClusterNotFoundError
		if !errors.As(err, &errorNotFound) {
			return err
		}

		return nil
	}

	// Check if vCluster is created via platform and has deletion prevention enabled
	if vCluster.HasPreventDeletionEnabled() {
		return fmt.Errorf("deletion of virtual cluster %s is prevented, disable \"Prevent Deletion\" via platform in order to delete this virtual cluster", vClusterName)
	}

	// prepare client
	err = cmd.prepare(vCluster)
	if err != nil {
		return err
	}

	if platformClient != nil {
		cmd.log.Debugf("deleting vcluster in platform")
		err = cmd.deleteVClusterInPlatform(ctx, platformClient, vClusterName)
		if err != nil {
			return fmt.Errorf("deleting vcluster in platform failed: %w", err)
		}
	}

	// test for helm
	helmBinaryPath, err := helmdownloader.GetHelmBinaryPath(ctx, cmd.log)
	if err != nil {
		return err
	}

	output, err := exec.Command(helmBinaryPath, "version", "--client", "--template", "{{.Version}}").Output()
	if err != nil {
		return err
	}

	err = clihelper.CheckHelmVersion(string(output))
	if err != nil {
		return err
	}

	// check if namespace
	if cmd.AutoDeleteNamespace {
		namespace, err := cmd.kubeClient.CoreV1().Namespaces().Get(ctx, cmd.Namespace, metav1.GetOptions{})
		if err != nil {
			cmd.log.Debugf("Error retrieving vcluster namespace: %v", err)
		} else if namespace != nil && namespace.Annotations != nil && namespace.Annotations[CreatedByVClusterAnnotation] == "true" {
			cmd.DeleteNamespace = true
		}
	}

	helmClient := helm.NewClient(cmd.rawConfig, cmd.log, helmBinaryPath)
	// before removing vCluster release, we need to get the config from values for later use
	values, err := helmClient.GetValues(ctx, vClusterName, cmd.Namespace, true)
	if err != nil {
		return err
	}

	vclusterConfig := &config.Config{}
	err = yaml.Unmarshal(values, vclusterConfig)
	if err != nil {
		return err
	}

	// we have to delete the chart
	cmd.log.Infof("Delete vcluster %s...", vClusterName)
	err = helmClient.Delete(vClusterName, cmd.Namespace)
	if err != nil {
		return err
	}
	cmd.log.Donef("Successfully deleted virtual cluster %s in namespace %s", vClusterName, cmd.Namespace)

	// delete priorityclasses
	if err = deletePriorityClasses(ctx, cmd, vClusterName); err != nil {
		return err
	}

	// try to delete the pvc
	if !cmd.KeepPVC && !cmd.DeleteNamespace {
		pvcName := fmt.Sprintf("data-%s-0", vClusterName)
		pvcNameForK8sAndEks := fmt.Sprintf("data-%s-etcd-0", vClusterName)

		err = cmd.kubeClient.CoreV1().PersistentVolumeClaims(cmd.Namespace).Delete(ctx, pvcName, metav1.DeleteOptions{})
		if err != nil {
			if !kerrors.IsNotFound(err) {
				return fmt.Errorf("delete pvc: %w", err)
			}
		} else {
			cmd.log.Donef("Successfully deleted virtual cluster pvc %s in namespace %s", pvcName, cmd.Namespace)
		}

		// Deleting PVC for K8s and eks distro as well.
		err = cmd.kubeClient.CoreV1().PersistentVolumeClaims(cmd.Namespace).Delete(ctx, pvcNameForK8sAndEks, metav1.DeleteOptions{})
		if err != nil {
			if !kerrors.IsNotFound(err) {
				return fmt.Errorf("delete pvc: %w", err)
			}
		} else {
			cmd.log.Donef("Successfully deleted virtual cluster pvc %s in namespace %s", pvcName, cmd.Namespace)
		}
	}

	// try to delete the ConfigMap
	if cmd.DeleteConfigMap {
		// Attempt to delete the ConfigMap
		configMapName := fmt.Sprintf("configmap-%s", vClusterName)
		err = cmd.kubeClient.CoreV1().ConfigMaps(cmd.Namespace).Delete(ctx, configMapName, metav1.DeleteOptions{})
		if err != nil {
			if !kerrors.IsNotFound(err) {
				return fmt.Errorf("delete configmap: %w", err)
			}
		} else {
			cmd.log.Donef("Successfully deleted ConfigMap %s in namespace %s", configMapName, cmd.Namespace)
		}
	}

	// delete coreDNS components since they're separately deployed and not with the vCluster helm chart
	err = coredns.DeleteCoreDNSComponents(ctx, cmd.kubeClient, cmd.Namespace)
	cmd.log.Info("Deleting CoreDNS components...")
	if err != nil {
		cmd.log.Warnf("delete coreDNS components: %v", err)
	}

	// check if there are any other vclusters in the namespace you are deleting vcluster in.
	vClusters, err := find.ListVClusters(ctx, cmd.Context, "", cmd.Namespace, cmd.log)
	if err != nil {
		return err
	}
	if len(vClusters) > 0 {
		// set to false if there are any virtual clusters running in the same namespace. The vcluster supposed to be deleted by the command has been deleted by now and hence the check should be greater than 0
		cmd.DeleteNamespace = false
	}

	// if namespace sync is enabled, use cleanup handlers to handle namespace cleanup
	if vclusterConfig.Sync.ToHost.Namespaces.Enabled {
		if err := CleanupSyncedNamespaces(ctx, cmd.Namespace, vClusterName, cmd.restConfig, cmd.kubeClient, cmd.log); err != nil {
			return fmt.Errorf("run namespace cleanup: %w", err)
		}
	}

	// check if we should cleanup vCluster host namespace
	if cmd.DeleteNamespace {
		// delete namespace
		err = cmd.kubeClient.CoreV1().Namespaces().Delete(ctx, cmd.Namespace, metav1.DeleteOptions{})
		if err != nil {
			if !kerrors.IsNotFound(err) {
				return fmt.Errorf("delete namespace: %w", err)
			}
		} else {
			cmd.log.Donef("Successfully deleted virtual cluster namespace %s", cmd.Namespace)
		}

		// wait for namespace deletion
		if cmd.Wait {
			cmd.log.Info("Waiting for virtual cluster to be deleted...")
			for {
				_, err = cmd.kubeClient.CoreV1().Namespaces().Get(ctx, cmd.Namespace, metav1.GetOptions{})
				if err != nil {
					break
				}

				time.Sleep(time.Second)
			}
			cmd.log.Done("Virtual Cluster is deleted")
		}
	}

	return nil
}

func (cmd *deleteHelm) deleteVClusterInPlatform(ctx context.Context, platformClient platform.Client, vClusterName string) error {
	managementClient, err := platformClient.Management()
	if err != nil {
		cmd.log.Debugf("Error creating management client: %v", err)
		return nil
	}

	virtualClusterInstances, err := managementClient.Loft().ManagementV1().VirtualClusterInstances(corev1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		cmd.log.Debugf("Error retrieving vcluster instances: %v", err)
		return nil
	}

	// get service uid
	vClusterService, err := cmd.kubeClient.CoreV1().Services(cmd.Namespace).Get(ctx, vClusterName, metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return fmt.Errorf("error retrieving vcluster service: %w", err)
	} else if kerrors.IsNotFound(err) {
		cmd.log.Warn("vcluster service not found, could not delete in platform")
		return nil
	}

	var toDelete []managementv1.VirtualClusterInstance
	for _, virtualClusterInstance := range virtualClusterInstances.Items {
		if virtualClusterInstance.Status.ServiceUID != "" && virtualClusterInstance.Status.ServiceUID == string(vClusterService.UID) {
			toDelete = append(toDelete, virtualClusterInstance)
		}
	}
	cmd.log.Debugf("found %d matching virtualclusterinstances", len(toDelete))

	for _, virtualClusterInstance := range toDelete {
		cmd.log.Infof("Delete virtual cluster instance %s/%s in platform", virtualClusterInstance.Namespace, virtualClusterInstance.Name)
		err = managementClient.Loft().ManagementV1().VirtualClusterInstances(virtualClusterInstance.Namespace).Delete(ctx, virtualClusterInstance.Name, metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("delete virtual cluster instance %s/%s: %w", virtualClusterInstance.Namespace, virtualClusterInstance.Name, err)
		}
	}

	return nil
}

func (cmd *deleteHelm) prepare(vCluster *find.VCluster) error {
	// load the raw config
	rawConfig, err := vCluster.ClientFactory.RawConfig()
	if err != nil {
		return fmt.Errorf("there is an error loading your current kube config (%w), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}
	err = deleteContext(&rawConfig, find.VClusterContextName(vCluster.Name, vCluster.Namespace, vCluster.Context), vCluster.Context)
	if err != nil {
		return fmt.Errorf("delete kube context: %w", err)
	}

	rawConfig.CurrentContext = vCluster.Context
	restConfig, err := vCluster.ClientFactory.ClientConfig()
	if err != nil {
		return err
	}

	// construct proxy name
	proxyName := find.VClusterConnectBackgroundProxyName(vCluster.Name, vCluster.Namespace, rawConfig.CurrentContext)
	_ = localkubernetes.CleanupBackgroundProxy(proxyName, cmd.log)

	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	cmd.Namespace = vCluster.Namespace
	cmd.rawConfig = &rawConfig
	cmd.restConfig = restConfig
	cmd.kubeClient = kubeClient
	return nil
}

func deleteContext(kubeConfig *clientcmdapi.Config, kubeContext string, otherContext string) error {
	if kubeConfig == nil || kubeConfig.Contexts == nil {
		return nil
	}

	// Get context
	contextRaw, ok := kubeConfig.Contexts[kubeContext]
	if !ok {
		return nil
	}

	// Remove context
	delete(kubeConfig.Contexts, kubeContext)

	removeAuthInfo := true
	removeCluster := true

	// Check if AuthInfo or Cluster is used by any other context
	for name, ctx := range kubeConfig.Contexts {
		if ctx == nil {
			continue
		}

		if name != kubeContext && ctx.AuthInfo == contextRaw.AuthInfo {
			removeAuthInfo = false
		}

		if name != kubeContext && ctx.Cluster == contextRaw.Cluster {
			removeCluster = false
		}
	}

	// Remove AuthInfo if not used by any other context
	if removeAuthInfo {
		delete(kubeConfig.AuthInfos, contextRaw.AuthInfo)
	}

	// Remove Cluster if not used by any other context
	if removeCluster {
		delete(kubeConfig.Clusters, contextRaw.Cluster)
	}

	if kubeConfig.CurrentContext == kubeContext {
		kubeConfig.CurrentContext = ""

		if otherContext != "" {
			kubeConfig.CurrentContext = otherContext
		} else if len(kubeConfig.Contexts) > 0 {
			for contextName, contextObj := range kubeConfig.Contexts {
				if contextObj != nil {
					kubeConfig.CurrentContext = contextName
					break
				}
			}
		}
	}

	return clientcmd.ModifyConfig(clientcmd.NewDefaultClientConfigLoadingRules(), *kubeConfig, false)
}

func deletePriorityClasses(ctx context.Context, cmd deleteHelm, vClusterName string) error {
	priorityClasses, err := cmd.kubeClient.SchedulingV1().PriorityClasses().List(ctx, metav1.ListOptions{
		LabelSelector: translate.MarkerLabel + "=" + translate.SafeConcatName(cmd.Namespace, "x", vClusterName),
	})
	if err != nil && !kerrors.IsForbidden(err) {
		return fmt.Errorf("list priorityClasses: %w", err)
	}

	if priorityClasses != nil && len(priorityClasses.Items) > 0 {
		for _, pc := range priorityClasses.Items {
			err = cmd.kubeClient.SchedulingV1().PriorityClasses().Delete(ctx, pc.Name, metav1.DeleteOptions{})
			if err != nil && !kerrors.IsNotFound(err) {
				return fmt.Errorf("delete priorityClass: %w", err)
			}
		}
	}

	return nil
}
