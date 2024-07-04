package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

func DeletePlatform(ctx context.Context, options *DeleteOptions, config *config.CLI, vClusterName string, log log.Logger) error {
	platformClient, err := platform.InitClientFromConfig(ctx, config)
	if err != nil {
		return err
	}

	// retrieve the vcluster
	vCluster, err := find.GetPlatformVCluster(ctx, platformClient, vClusterName, options.Project, log)
	if err != nil {
		return err
	} else if vCluster.VirtualCluster != nil && vCluster.VirtualCluster.Spec.External {
		return fmt.Errorf("cannot delete a virtual cluster that was created via helm, please run 'vcluster use driver helm' or use the '--driver helm' flag")
	}

	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	log.Infof("Deleting virtual cluster %s in project %s", vCluster.VirtualCluster.Name, vCluster.Project.Name)
	err = managementClient.Loft().ManagementV1().VirtualClusterInstances(vCluster.VirtualCluster.Namespace).Delete(ctx, vCluster.VirtualCluster.Name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("delete virtual cluster: %w", err)
	}

	log.Donef("Successfully deleted virtual cluster %s in project %s", vCluster.VirtualCluster.Name, vCluster.Project.Name)

	// update kube config
	if options.DeleteContext {
		err = deletePlatformContext(vCluster.VirtualCluster.Name, vCluster.Project.Name)
		if err != nil {
			return fmt.Errorf("delete kube context: %w", err)
		}
	}

	// wait until deleted
	if options.Wait {
		log.Info("Waiting for virtual cluster to be deleted...")
		for isVirtualClusterInstanceStillThere(ctx, managementClient, vCluster.VirtualCluster.Namespace, vCluster.VirtualCluster.Name) {
			time.Sleep(time.Second)
		}
		log.Done("Virtual Cluster is deleted")
	}

	return nil
}

func isVirtualClusterInstanceStillThere(ctx context.Context, managementClient kube.Interface, namespace, name string) bool {
	_, err := managementClient.Loft().ManagementV1().VirtualClusterInstances(namespace).Get(ctx, name, metav1.GetOptions{})
	return err == nil
}

func deletePlatformContext(vClusterName, projectName string) error {
	kubeClientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})
	kubeConfig, err := kubeClientConfig.RawConfig()
	if err != nil {
		return fmt.Errorf("load kube config: %w", err)
	}

	// remove matching contexts
	for contextName := range kubeConfig.Contexts {
		name, project, previousContext := find.VClusterPlatformFromContext(contextName)
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
