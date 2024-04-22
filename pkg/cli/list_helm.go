package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/log/table"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

// ListVCluster holds information about a cluster
type ListVCluster struct {
	Created    time.Time
	Name       string
	Namespace  string
	Version    string
	Status     string
	AgeSeconds int
	Connected  bool
}

type ListOptions struct {
	Manager string

	Output string
}

func ListHelm(ctx context.Context, options *ListOptions, globalFlags *flags.GlobalFlags, log log.Logger) error {
	rawConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).RawConfig()
	if err != nil {
		return err
	}
	currentContext := rawConfig.CurrentContext

	if globalFlags.Context == "" {
		globalFlags.Context = currentContext
	}

	namespace := metav1.NamespaceAll
	if globalFlags.Namespace != "" {
		namespace = globalFlags.Namespace
	}

	vClusters, err := find.ListVClusters(ctx, globalFlags.Context, "", namespace, log.ErrorStreamOnly())
	if err != nil {
		return err
	}

	return printVClusters(options, globalFlags, ossToVClusters(vClusters, currentContext), log)
}

func printVClusters(options *ListOptions, globalFlags *flags.GlobalFlags, output []ListVCluster, log log.Logger) error {
	if options.Output == "json" {
		bytes, err := json.MarshalIndent(output, "", "    ")
		if err != nil {
			return fmt.Errorf("json marshal vClusters: %w", err)
		}

		log.WriteString(logrus.InfoLevel, string(bytes)+"\n")
	} else {
		header := []string{"NAME", "NAMESPACE", "STATUS", "VERSION", "CONNECTED", "AGE"}
		values := toValues(output)
		table.PrintTable(log, header, values)
		if strings.HasPrefix(globalFlags.Context, "vcluster_") || strings.HasPrefix(globalFlags.Context, "vcluster-platform_") {
			log.Infof("Run `vcluster disconnect` to switch back to the parent context")
		}
	}

	return nil
}

func ossToVClusters(vClusters []find.VCluster, currentContext string) []ListVCluster {
	var output []ListVCluster
	for _, vCluster := range vClusters {
		vClusterOutput := ListVCluster{
			Name:       vCluster.Name,
			Namespace:  vCluster.Namespace,
			Created:    vCluster.Created.Time,
			Version:    vCluster.Version,
			AgeSeconds: int(time.Since(vCluster.Created.Time).Round(time.Second).Seconds()),
			Status:     string(vCluster.Status),
		}
		vClusterOutput.Connected = currentContext == find.VClusterContextName(
			vCluster.Name,
			vCluster.Namespace,
			vCluster.Context,
		)
		output = append(output, vClusterOutput)
	}
	return output
}

func toValues(vClusters []ListVCluster) [][]string {
	var values [][]string
	for _, vCluster := range vClusters {
		isConnected := ""
		if vCluster.Connected {
			isConnected = "True"
		}

		values = append(values, []string{
			vCluster.Name,
			vCluster.Namespace,
			vCluster.Status,
			vCluster.Version,
			isConnected,
			time.Since(vCluster.Created).Round(1 * time.Second).String(),
		})
	}
	return values
}
