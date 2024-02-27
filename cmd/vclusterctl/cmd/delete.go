package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	proclient "github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/loftctl/v3/pkg/kube"
	loftctlUtil "github.com/loft-sh/loftctl/v3/pkg/util"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/app/localkubernetes"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/find"
	"github.com/loft-sh/vcluster/pkg/procli"
	"github.com/loft-sh/vcluster/pkg/util/clihelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/spf13/cobra"
)

// DeleteCmd holds the delete cmd flags
type DeleteCmd struct {
	*flags.GlobalFlags
	log                 log.Logger
	rawConfig           *clientcmdapi.Config
	restConfig          *rest.Config
	kubeClient          *kubernetes.Clientset
	Project             string
	Wait                bool
	KeepPVC             bool
	DeleteNamespace     bool
	DeleteConfigMap     bool
	AutoDeleteNamespace bool
	IgnoreNotFound      bool
}

// NewDeleteCmd creates a new command
func NewDeleteCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &DeleteCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "delete" + loftctlUtil.VClusterNameOnlyUseLine,
		Short: "Deletes a virtual cluster",
		Long: `
#######################################################
################### vcluster delete ###################
#######################################################
Deletes a virtual cluster

Example:
vcluster delete test --namespace test
#######################################################
	`,
		Args:              loftctlUtil.VClusterNameOnlyValidator,
		Aliases:           []string{"rm"},
		ValidArgsFunction: newValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd, args)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.Project, "project", "", "[PRO] The pro project the vcluster is in")
	cobraCmd.Flags().BoolVar(&cmd.Wait, "wait", true, "If enabled, vcluster will wait until the vcluster is deleted")
	cobraCmd.Flags().BoolVar(&cmd.DeleteConfigMap, "delete-configmap", false, "If enabled, vCluster will delete the ConfigMap of the vCluster")
	cobraCmd.Flags().BoolVar(&cmd.KeepPVC, "keep-pvc", false, "If enabled, vcluster will not delete the persistent volume claim of the vcluster")
	cobraCmd.Flags().BoolVar(&cmd.DeleteNamespace, "delete-namespace", false, "If enabled, vcluster will delete the namespace of the vcluster. In the case of multi-namespace mode, will also delete all other namespaces created by vcluster")
	cobraCmd.Flags().BoolVar(&cmd.AutoDeleteNamespace, "auto-delete-namespace", true, "If enabled, vcluster will delete the namespace of the vcluster if it was created by vclusterctl. In the case of multi-namespace mode, will also delete all other namespaces created by vcluster")
	cobraCmd.Flags().BoolVar(&cmd.IgnoreNotFound, "ignore-not-found", false, "If enabled, vcluster will not error out in case the target vcluster does not exist")
	return cobraCmd
}

// Run executes the functionality
func (cmd *DeleteCmd) Run(cobraCmd *cobra.Command, args []string) error {
	ctx := cobraCmd.Context()

	// get pro client
	proClient, err := procli.CreateProClient()
	if err != nil {
		cmd.log.Debugf("Error creating pro client: %v", err)
	}

	// find vcluster
	vClusterName := args[0]
	vCluster, proVCluster, err := find.GetVCluster(ctx, proClient, cmd.Context, vClusterName, cmd.Namespace, cmd.Project, cmd.log)
	if err != nil {
		if !cmd.IgnoreNotFound {
			return err
		}
		var errorNotFound *find.VclusterNotFoundError
		if !errors.As(err, &errorNotFound) {
			return err
		}
		return nil
	} else if proVCluster != nil {
		return cmd.deleteProVCluster(cobraCmd.Context(), proClient, proVCluster)
	}

	// prepare client
	err = cmd.prepare(vCluster)
	if err != nil {
		return err
	}

	// test for helm
	helmBinaryPath, err := GetHelmBinaryPath(ctx, cmd.log)
	if err != nil {
		return err
	}

	output, err := exec.Command(helmBinaryPath, "version", "--client").CombinedOutput()
	if errHelm := clihelper.CheckHelmVersion(string(output)); errHelm != nil {
		return errHelm
	}
	if err != nil {
		return fmt.Errorf("seems like there are issues with your helm client: \n\n%s", output)
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

	// we have to delete the chart
	cmd.log.Infof("Delete vcluster %s...", args[0])
	err = helm.NewClient(cmd.rawConfig, cmd.log, helmBinaryPath).Delete(args[0], cmd.Namespace)
	if err != nil {
		return err
	}
	cmd.log.Donef("Successfully deleted virtual cluster %s in namespace %s", args[0], cmd.Namespace)

	// try to delete the pvc
	if !cmd.KeepPVC && !cmd.DeleteNamespace {
		pvcName := fmt.Sprintf("data-%s-0", args[0])
		pvcNameForK8sAndEks := fmt.Sprintf("data-%s-etcd-0", args[0])

		client, err := kubernetes.NewForConfig(cmd.restConfig)
		if err != nil {
			return err
		}

		err = client.CoreV1().PersistentVolumeClaims(cmd.Namespace).Delete(ctx, pvcName, metav1.DeleteOptions{})
		if err != nil {
			if !kerrors.IsNotFound(err) {
				return errors.Wrap(err, "delete pvc")
			}
		} else {
			cmd.log.Donef("Successfully deleted virtual cluster pvc %s in namespace %s", pvcName, cmd.Namespace)
		}

		// Deleting PVC for K8s and eks distro as well.
		err = client.CoreV1().PersistentVolumeClaims(cmd.Namespace).Delete(ctx, pvcNameForK8sAndEks, metav1.DeleteOptions{})
		if err != nil {
			if !kerrors.IsNotFound(err) {
				return errors.Wrap(err, "delete pvc")
			}
		} else {
			cmd.log.Donef("Successfully deleted virtual cluster pvc %s in namespace %s", pvcName, cmd.Namespace)
		}
	}

	// try to delete the ConfigMap
	if cmd.DeleteConfigMap {
		client, err := kubernetes.NewForConfig(cmd.restConfig)
		if err != nil {
			return err
		}

		configMapName := fmt.Sprintf("configmap-%s", args[0])

		// Attempt to delete the ConfigMap
		err = client.CoreV1().ConfigMaps(cmd.Namespace).Delete(ctx, configMapName, metav1.DeleteOptions{})
		if err != nil {
			if !kerrors.IsNotFound(err) {
				return errors.Wrap(err, "delete configmap")
			}
		} else {
			cmd.log.Donef("Successfully deleted ConfigMap %s in namespace %s", configMapName, cmd.Namespace)
		}
	}

	// check if there are any other vclusters in the namespace you are deleting vcluster in.
	vClusters, _, err := find.ListVClusters(cobraCmd.Context(), nil, cmd.Context, "", cmd.Namespace, "", cmd.log)
	if err != nil {
		return err
	}
	if len(vClusters) > 0 {
		// set to false if there are any virtual clusters running in the same namespace. The vcluster supposed to be deleted by the command has been deleted by now and hence the check should be greater than 0
		cmd.DeleteNamespace = false
	}

	// try to delete the namespace
	if cmd.DeleteNamespace {
		client, err := kubernetes.NewForConfig(cmd.restConfig)
		if err != nil {
			return err
		}

		// delete namespace
		err = client.CoreV1().Namespaces().Delete(ctx, cmd.Namespace, metav1.DeleteOptions{})
		if err != nil {
			if !kerrors.IsNotFound(err) {
				return errors.Wrap(err, "delete namespace")
			}
		} else {
			cmd.log.Donef("Successfully deleted virtual cluster namespace %s", cmd.Namespace)
		}

		// delete multi namespace mode namespaces
		namespaces, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
			LabelSelector: translate.MarkerLabel + "=" + translate.SafeConcatName(cmd.Namespace, "x", args[0]),
		})
		if err != nil && !kerrors.IsForbidden(err) {
			return errors.Wrap(err, "list namespaces")
		}

		// delete all namespaces
		if namespaces != nil && len(namespaces.Items) > 0 {
			for _, namespace := range namespaces.Items {
				err = client.CoreV1().Namespaces().Delete(ctx, namespace.Name, metav1.DeleteOptions{})
				if err != nil {
					if !kerrors.IsNotFound(err) {
						return errors.Wrap(err, "delete namespace")
					}
				} else {
					cmd.log.Donef("Successfully deleted virtual cluster namespace %s", namespace.Name)
				}
			}
		}

		// wait for vcluster deletion
		if cmd.Wait {
			cmd.log.Info("Waiting for virtual cluster to be deleted...")
			for {
				_, err = client.CoreV1().Namespaces().Get(ctx, cmd.Namespace, metav1.GetOptions{})
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

func (cmd *DeleteCmd) deleteProVCluster(ctx context.Context, proClient proclient.Client, vCluster *procli.VirtualClusterInstanceProject) error {
	managementClient, err := proClient.Management()
	if err != nil {
		return err
	}

	cmd.log.Infof("Deleting virtual cluster %s in project %s", vCluster.VirtualCluster.Name, vCluster.Project.Name)

	err = managementClient.Loft().ManagementV1().VirtualClusterInstances(vCluster.VirtualCluster.Namespace).Delete(ctx, vCluster.VirtualCluster.Name, metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrap(err, "delete virtual cluster")
	}

	cmd.log.Donef("Successfully deleted virtual cluster %s in project %s", vCluster.VirtualCluster.Name, vCluster.Project.Name)

	// update kube config
	err = deleteProContext(vCluster.VirtualCluster.Name, vCluster.Project.Name)
	if err != nil {
		return errors.Wrap(err, "delete kube context")
	}

	// wait until deleted
	if cmd.Wait {
		cmd.log.Info("Waiting for virtual cluster to be deleted...")
		for isVirtualClusterInstanceStillThere(ctx, managementClient, vCluster.VirtualCluster.Namespace, vCluster.VirtualCluster.Name) {
			time.Sleep(time.Second)
		}
		cmd.log.Done("Virtual Cluster is deleted")
	}

	return nil
}

func isVirtualClusterInstanceStillThere(ctx context.Context, managementClient kube.Interface, namespace, name string) bool {
	_, err := managementClient.Loft().ManagementV1().VirtualClusterInstances(namespace).Get(ctx, name, metav1.GetOptions{})
	return err == nil
}

func (cmd *DeleteCmd) prepare(vCluster *find.VCluster) error {
	// load the raw config
	rawConfig, err := vCluster.ClientFactory.RawConfig()
	if err != nil {
		return fmt.Errorf("there is an error loading your current kube config (%w), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}
	err = deleteContext(&rawConfig, find.VClusterContextName(vCluster.Name, vCluster.Namespace, vCluster.Context), vCluster.Context)
	if err != nil {
		return errors.Wrap(err, "delete kube context")
	}

	rawConfig.CurrentContext = vCluster.Context
	restConfig, err := vCluster.ClientFactory.ClientConfig()
	if err != nil {
		return err
	}

	err = localkubernetes.CleanupLocal(vCluster.Name, vCluster.Namespace, &rawConfig, cmd.log)
	if err != nil {
		cmd.log.Warnf("error cleaning up: %v", err)
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

func deleteProContext(vClusterName, projectName string) error {
	kubeClientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})
	kubeConfig, err := kubeClientConfig.RawConfig()
	if err != nil {
		return fmt.Errorf("load kube config: %w", err)
	}

	// remove matching contexts
	for contextName := range kubeConfig.Contexts {
		name, project, previousContext := find.VClusterProFromContext(contextName)
		if vClusterName != name || projectName != project {
			continue
		}

		err := deleteContext(&kubeConfig, contextName, previousContext)
		if err != nil {
			return err
		}
	}

	return nil
}

func deleteContext(kubeConfig *clientcmdapi.Config, kubeContext string, otherContext string) error {
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
