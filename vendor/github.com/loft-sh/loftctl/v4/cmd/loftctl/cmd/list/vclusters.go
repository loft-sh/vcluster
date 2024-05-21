package list

import (
	"context"
	"fmt"
	"time"

	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/loftctl/v4/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v4/pkg/client"
	"github.com/loft-sh/loftctl/v4/pkg/client/helper"
	"github.com/loft-sh/loftctl/v4/pkg/clihelper"
	"github.com/loft-sh/loftctl/v4/pkg/projectutil"
	"github.com/loft-sh/loftctl/v4/pkg/upgrade"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/table"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/duration"
)

// VirtualClustersCmd holds the data
type VirtualClustersCmd struct {
	*flags.GlobalFlags

	ShowLegacy bool
	log        log.Logger
}

// NewVirtualClustersCmd creates a new command
func NewVirtualClustersCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &VirtualClustersCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}
	description := product.ReplaceWithHeader("list vclusters", `
List the loft virtual clusters you have access to

Example:
loft list vclusters
########################################################
	`)
	if upgrade.IsPlugin == "true" {
		description = `
########################################################
############### devspace list vclusters ################
########################################################
List the loft virtual clusters you have access to

Example:
devspace list vclusters
########################################################
	`
	}
	listCmd := &cobra.Command{
		Use:   "vclusters",
		Short: product.Replace("Lists the loft virtual clusters you have access to"),
		Long:  description,
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}
	listCmd.Flags().BoolVar(&cmd.ShowLegacy, "show-legacy", false, "If true, will always show the legacy virtual clusters as well")
	return listCmd
}

// Run executes the functionality
func (cmd *VirtualClustersCmd) Run(ctx context.Context) error {
	baseClient, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return err
	}
	self, err := baseClient.GetSelf(ctx)
	if err != nil {
		return fmt.Errorf("failed to get self: %w", err)
	}
	projectutil.SetProjectNamespacePrefix(self.Status.ProjectNamespacePrefix)

	header := []string{
		"Name",
		"Project",
		"Cluster",
		"Namespace",
		"Status",
		"Age",
	}
	values := [][]string{}

	virtualClusterInstances, err := helper.GetVirtualClusterInstances(ctx, baseClient)
	if err != nil {
		return err
	}

	for _, virtualCluster := range virtualClusterInstances {
		values = append(values, []string{
			clihelper.GetTableDisplayName(virtualCluster.VirtualCluster.Name, virtualCluster.VirtualCluster.Spec.DisplayName),
			virtualCluster.Project.Name,
			virtualCluster.VirtualCluster.Spec.ClusterRef.Cluster,
			virtualCluster.VirtualCluster.Spec.ClusterRef.Namespace,
			string(virtualCluster.VirtualCluster.Status.Phase),
			duration.HumanDuration(time.Since(virtualCluster.VirtualCluster.CreationTimestamp.Time)),
		})
	}
	if len(virtualClusterInstances) == 0 || cmd.ShowLegacy {
		virtualClusters, err := helper.GetVirtualClusters(ctx, baseClient, cmd.log)
		if err != nil {
			return err
		}
		for _, virtualCluster := range virtualClusters {
			status := "Active"
			if virtualCluster.VirtualCluster.Status.HelmRelease != nil {
				status = virtualCluster.VirtualCluster.Status.HelmRelease.Phase
			}
			vClusterName := virtualCluster.VirtualCluster.Name
			if virtualCluster.VirtualCluster.Annotations != nil && virtualCluster.VirtualCluster.Annotations["loft.sh/display-name"] != "" {
				vClusterName = virtualCluster.VirtualCluster.Annotations["loft.sh/display-name"] + " (" + vClusterName + ")"
			}

			values = append(values, []string{
				vClusterName,
				"",
				virtualCluster.Cluster,
				virtualCluster.VirtualCluster.Namespace,
				status,
				duration.HumanDuration(time.Since(virtualCluster.VirtualCluster.CreationTimestamp.Time)),
			})
		}
	}

	table.PrintTable(cmd.log, header, values)
	return nil
}
