package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/loft-sh/log"
	"github.com/loft-sh/log/table"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"k8s.io/client-go/tools/clientcmd"
)

func ListPlatform(ctx context.Context, options *ListOptions, globalFlags *flags.GlobalFlags, logger log.Logger, projectName string, showUserOwned bool) error {
	rawConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).RawConfig()
	if err != nil {
		return err
	}
	currentContext := rawConfig.CurrentContext

	if globalFlags.Context == "" {
		globalFlags.Context = currentContext
	}

	platformClient, err := platform.InitClientFromConfig(ctx, globalFlags.LoadedConfig(logger))
	if err != nil {
		return err
	}

	proVClusters, err := platform.ListVClusters(ctx, platformClient, "", projectName, showUserOwned)
	if err != nil {
		return err
	}

	vClusters, vClusterProjectMapping := proToVClusters(proVClusters, currentContext)
	err = printProVClusters(ctx, options, vClusters, vClusterProjectMapping, globalFlags, logger)
	if err != nil {
		return err
	}

	return nil
}

func proToVClusters(vClusters []*platform.VirtualClusterInstanceProject, currentContext string) ([]ListVCluster, vClusterProjectMap) {
	var output []ListVCluster
	vClusterProjectMapping := make(vClusterProjectMap)
	for _, vCluster := range vClusters {
		status := string(vCluster.VirtualCluster.Status.Phase)
		if vCluster.VirtualCluster.DeletionTimestamp != nil {
			status = "Terminating"
		} else if status == "" {
			status = "Pending"
		}

		version := ""
		if vCluster.VirtualCluster.Status.VirtualCluster != nil && vCluster.VirtualCluster.Status.VirtualCluster.HelmRelease.Chart.Version != "" {
			version = vCluster.VirtualCluster.Status.VirtualCluster.HelmRelease.Chart.Version
		} else if vCluster.VirtualCluster.Spec.Template != nil && vCluster.VirtualCluster.Spec.Template.HelmRelease.Chart.Version != "" {
			version = vCluster.VirtualCluster.Spec.Template.HelmRelease.Chart.Version
		}

		name := vCluster.VirtualCluster.Spec.ClusterRef.VirtualCluster
		if vCluster.VirtualCluster.Spec.NetworkPeer {
			name = vCluster.VirtualCluster.Name
		}

		connected := strings.HasPrefix(currentContext, "vcluster-platform_"+vCluster.VirtualCluster.Name+"_"+vCluster.Project.Name)
		vClusterOutput := ListVCluster{
			Name:       name,
			Namespace:  vCluster.VirtualCluster.Spec.ClusterRef.Namespace,
			Connected:  connected,
			Created:    vCluster.VirtualCluster.CreationTimestamp.Time,
			AgeSeconds: int(time.Since(vCluster.VirtualCluster.CreationTimestamp.Time).Round(time.Second).Seconds()),
			Status:     status,
			Version:    version,
		}
		uniqueVClusterIdentifier := vClusterOutput.Name + "_" + vClusterOutput.Namespace
		vClusterProjectMapping[uniqueVClusterIdentifier] = vCluster.Project.Name
		output = append(output, vClusterOutput)
	}
	return output, vClusterProjectMapping
}

func printProVClusters(ctx context.Context, options *ListOptions, output []ListVCluster,
	vClusterProjectMapping vClusterProjectMap, globalFlags *flags.GlobalFlags, logger log.Logger) error {
	if options.Output == "json" {
		bytes, err := json.MarshalIndent(output, "", "    ")
		if err != nil {
			return fmt.Errorf("json marshal vClusters: %w", err)
		}

		logger.WriteString(logrus.InfoLevel, string(bytes)+"\n")
	} else {
		header := []string{"NAME", "NAMESPACE", "PROJECT", "STATUS", "VERSION", "CONNECTED", "AGE"}
		values := toTableValues(output, vClusterProjectMapping)
		table.PrintTable(logger, header, values)

		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		vClusters, _ := find.ListVClusters(ctx, globalFlags.Context, "", "", log.Discard)
		if len(vClusters) > 0 {
			logger.Infof("You also have %d virtual clusters in your current kube-context.", len(vClusters))
			logger.Info("If you want to see them, run: 'vcluster list --driver helm' or 'vcluster use driver helm' to change the default")
		}

		// show disconnect command
		if strings.HasPrefix(globalFlags.Context, "vcluster_") || strings.HasPrefix(globalFlags.Context, "vcluster-platform_") {
			logger.Infof("Run `vcluster disconnect` to switch back to the parent context")
		}
	}
	return nil
}

func toTableValues(vClusters []ListVCluster, vClusterProjectMapping vClusterProjectMap) [][]string {
	var values [][]string
	for _, vCluster := range vClusters {
		isConnected := ""
		if vCluster.Connected {
			isConnected = "True"
		}

		uniqueVClusterIdentifier := vCluster.Name + "_" + vCluster.Namespace
		values = append(values, []string{
			vCluster.Name,
			vCluster.Namespace,
			vClusterProjectMapping[uniqueVClusterIdentifier],
			vCluster.Status,
			vCluster.Version,
			isConnected,
			time.Since(vCluster.Created).Round(1 * time.Second).String(),
		})
	}
	return values
}
