package list

import (
	"context"
	"time"

	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/table"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
)

// ClustersCmd holds the login cmd flags
type ClustersCmd struct {
	*flags.GlobalFlags

	log log.Logger
}

// newClustersCmd creates a new spaces command
func newClustersCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ClustersCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}
	description := product.ReplaceWithHeader("list clusters", `
List the vcluster platform clusters you have access to

Example:
vcluster platform list clusters
########################################################
	`)
	clustersCmd := &cobra.Command{
		Use:   "clusters",
		Short: product.Replace("Lists the loft clusters you have access to"),
		Long:  description,
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			return cmd.RunClusters(cobraCmd.Context())
		},
	}

	return clustersCmd
}

// RunClusters executes the functionality
func (cmd *ClustersCmd) RunClusters(ctx context.Context) error {
	platformClient, err := platform.InitClientFromConfig(ctx, cmd.LoadedConfig(cmd.log))
	if err != nil {
		return err
	}

	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	clusterList, err := managementClient.Loft().ManagementV1().Clusters().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	header := []string{
		"Cluster",
		"Age",
	}
	values := [][]string{}
	for _, cluster := range clusterList.Items {
		values = append(values, []string{
			cluster.Name,
			duration.HumanDuration(time.Since(cluster.CreationTimestamp.Time)),
		})
	}

	table.PrintTable(cmd.log, header, values)
	return nil
}
