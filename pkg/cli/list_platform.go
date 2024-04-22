package cli

import (
	"context"
	"strings"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"k8s.io/client-go/tools/clientcmd"
)

func ListPlatform(ctx context.Context, options *ListOptions, globalFlags *flags.GlobalFlags, log log.Logger) error {
	rawConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).RawConfig()
	if err != nil {
		return err
	}
	currentContext := rawConfig.CurrentContext

	proClient, err := platform.CreatePlatformClient()
	if err != nil {
		return err
	}

	proVClusters, err := platform.ListVClusters(ctx, proClient, "", "")
	if err != nil {
		return err
	}

	return printVClusters(options, globalFlags, proToVClusters(proVClusters, currentContext), log)
}

func proToVClusters(vClusters []platform.VirtualClusterInstanceProject, currentContext string) []ListVCluster {
	var output []ListVCluster
	for _, vCluster := range vClusters {
		status := string(vCluster.VirtualCluster.Status.Phase)
		if vCluster.VirtualCluster.DeletionTimestamp != nil {
			status = "Terminating"
		} else if status == "" {
			status = "Pending"
		}

		connected := strings.HasPrefix(currentContext, "vcluster-platform_"+vCluster.VirtualCluster.Name+"_"+vCluster.Project.Name)
		vClusterOutput := ListVCluster{
			Name:       vCluster.VirtualCluster.Spec.ClusterRef.VirtualCluster,
			Namespace:  vCluster.VirtualCluster.Spec.ClusterRef.Namespace,
			Connected:  connected,
			Created:    vCluster.VirtualCluster.CreationTimestamp.Time,
			AgeSeconds: int(time.Since(vCluster.VirtualCluster.CreationTimestamp.Time).Round(time.Second).Seconds()),
			Status:     status,
			Version:    vCluster.VirtualCluster.Status.VirtualCluster.HelmRelease.Chart.Version,
		}
		output = append(output, vClusterOutput)
	}
	return output
}
