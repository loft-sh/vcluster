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
	"k8s.io/apimachinery/pkg/util/duration"
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

	err = printProVClusters(ctx, options, proToVClusters(proVClusters, currentContext), globalFlags, logger)
	if err != nil {
		return err
	}

	return nil
}

func proToVClusters(vClusters []*platform.VirtualClusterInstanceProject, currentContext string) []ListProVCluster {
	var output []ListProVCluster
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
		vClusterOutput := ListProVCluster{
			ListVCluster{
				Name:       name,
				Namespace:  vCluster.VirtualCluster.Spec.ClusterRef.Namespace,
				Connected:  connected,
				Created:    vCluster.VirtualCluster.CreationTimestamp.Time,
				AgeSeconds: int(time.Since(vCluster.VirtualCluster.CreationTimestamp.Time).Round(time.Second).Seconds()),
				Status:     status,
				Version:    version,
			},
			vCluster.Project.Name,
		}
		output = append(output, vClusterOutput)
	}
	return output
}

func printProVClusters(ctx context.Context, options *ListOptions, output []ListProVCluster, globalFlags *flags.GlobalFlags, logger log.Logger) error {
	if options.Output == "json" {
		bytes, err := json.MarshalIndent(output, "", "    ")
		if err != nil {
			return fmt.Errorf("json marshal vClusters: %w", err)
		}

		logger.WriteString(logrus.InfoLevel, string(bytes)+"\n")
	} else {
		header := []string{"NAME", "NAMESPACE", "PROJECT", "STATUS", "VERSION", "CONNECTED", "AGE"}
		values := toTableValues(output)
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

func toTableValues(vClusters []ListProVCluster) [][]string {
	var values [][]string
	for _, vCluster := range vClusters {
		isConnected := ""
		if vCluster.Connected {
			isConnected = "True"
		}

		values = append(values, []string{
			vCluster.Name,
			vCluster.Namespace,
			vCluster.Project,
			vCluster.Status,
			vCluster.Version,
			isConnected,
			duration.HumanDuration(time.Since(vCluster.Created)),
		})
	}
	return values
}
