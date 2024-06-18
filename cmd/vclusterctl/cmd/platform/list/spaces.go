package list

import (
	"context"
	"strconv"
	"time"

	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/table"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/clihelper"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/duration"
)

// NamespacesCmd holds the login cmd flags
type NamespacesCmd struct {
	*flags.GlobalFlags

	log log.Logger
}

// newNamespacesCmd creates a new spaces command
func newNamespacesCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &NamespacesCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}
	description := product.ReplaceWithHeader("list namespaces", `
List the vCluster platform namespaces you have access to
Example:
vcluster platform list namespaces
########################################################
	`)
	listCmd := &cobra.Command{
		Use:     "namespaces",
		Short:   product.Replace("Lists the vCluster platform namespaces you have access to"),
		Long:    description,
		Args:    cobra.NoArgs,
		Aliases: []string{"spaces"},
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			return cmd.RunSpaces(cobraCmd.Context())
		},
	}
	return listCmd
}

// RunSpaces executes the functionality
func (cmd *NamespacesCmd) RunSpaces(ctx context.Context) error {
	platformClient, err := platform.InitClientFromConfig(ctx, cmd.LoadedConfig(cmd.log))
	if err != nil {
		return err
	}

	header := []string{
		"Name",
		"Project",
		"Cluster",
		"Sleeping",
		"Status",
		"Age",
	}
	values := [][]string{}
	spaceInstances, err := platform.GetSpaceInstances(ctx, platformClient)
	if err != nil {
		return err
	}
	for _, space := range spaceInstances {
		values = append(values, []string{
			clihelper.GetTableDisplayName(space.SpaceInstance.Name, space.SpaceInstance.Spec.DisplayName),
			space.Project.Name,
			space.SpaceInstance.Spec.ClusterRef.Cluster,
			strconv.FormatBool(space.SpaceInstance.Status.Phase == storagev1.InstanceSleeping),
			string(space.SpaceInstance.Status.Phase),
			duration.HumanDuration(time.Since(space.SpaceInstance.CreationTimestamp.Time)),
		})
	}

	table.PrintTable(cmd.log, header, values)
	return nil
}
