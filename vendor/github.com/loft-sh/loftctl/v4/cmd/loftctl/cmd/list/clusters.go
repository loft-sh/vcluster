package list

import (
	"context"
	"fmt"
	"time"

	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/loftctl/v4/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v4/pkg/client"
	"github.com/loft-sh/loftctl/v4/pkg/projectutil"
	"github.com/loft-sh/loftctl/v4/pkg/upgrade"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/table"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
)

// ClustersCmd holds the login cmd flags
type ClustersCmd struct {
	*flags.GlobalFlags

	log log.Logger
}

// NewClustersCmd creates a new spaces command
func NewClustersCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ClustersCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}
	description := product.ReplaceWithHeader("list clusters", `
List the loft clusters you have access to

Example:
loft list clusters
########################################################
	`)
	if upgrade.IsPlugin == "true" {
		description = `
########################################################
############### devspace list clusters #################
########################################################
List the loft clusters you have access to

Example:
devspace list clusters
########################################################
	`
	}
	clustersCmd := &cobra.Command{
		Use:   "clusters",
		Short: product.Replace("Lists the loft clusters you have access to"),
		Long:  description,
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunClusters(cobraCmd.Context())
		},
	}

	return clustersCmd
}

// RunClusters executes the functionality
func (cmd *ClustersCmd) RunClusters(ctx context.Context) error {
	baseClient, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return err
	}
	self, err := baseClient.GetSelf(ctx)
	if err != nil {
		return fmt.Errorf("failed to get self: %w", err)
	}
	projectutil.SetProjectNamespacePrefix(self.Status.ProjectNamespacePrefix)

	managementClient, err := baseClient.Management()
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
