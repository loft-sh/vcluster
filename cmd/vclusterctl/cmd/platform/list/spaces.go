package list

import (
	"context"
	"strconv"
	"time"

	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/loftctl/v4/pkg/upgrade"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/table"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/clihelper"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/duration"
)

// SpacesCmd holds the login cmd flags
type SpacesCmd struct {
	*flags.GlobalFlags

	log log.Logger
}

// newSpacesCmd creates a new spaces command
func newSpacesCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &SpacesCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}
	description := product.ReplaceWithHeader("list spaces", `
List the vCluster platform spaces you have access to
Example:
vcluster platform list spaces
########################################################
	`)
	if upgrade.IsPlugin == "true" {
		description = `
########################################################
################ devspace list spaces ##################
########################################################
List the vCluster platform spaces you have access to
Example:
devspace list spaces
########################################################
	`
	}
	listCmd := &cobra.Command{
		Use:   "spaces",
		Short: product.Replace("Lists the vCluster platform spaces you have access to"),
		Long:  description,
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			return cmd.RunSpaces(cobraCmd.Context())
		},
	}
	return listCmd
}

// RunSpaces executes the functionality
func (cmd *SpacesCmd) RunSpaces(ctx context.Context) error {
	platformClient, err := platform.NewClientFromConfig(ctx, cmd.LoadedConfig(cmd.log))
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
	if len(spaceInstances) == 0 {
		spaces, err := platform.GetSpaces(ctx, platformClient, cmd.log)
		if err != nil {
			return err
		}
		for _, space := range spaces {
			sleepModeConfig := space.Status.SleepModeConfig
			sleeping := "false"
			if sleepModeConfig.Status.SleepingSince != 0 {
				sleeping = duration.HumanDuration(time.Since(time.Unix(sleepModeConfig.Status.SleepingSince, 0)))
			}
			spaceName := space.Name
			if space.Annotations != nil && space.Annotations["loft.sh/display-name"] != "" {
				spaceName = space.Annotations["loft.sh/display-name"] + " (" + spaceName + ")"
			}

			values = append(values, []string{
				spaceName,
				"",
				space.Cluster,
				sleeping,
				string(space.Space.Status.Phase),
				duration.HumanDuration(time.Since(space.Space.CreationTimestamp.Time)),
			})
		}
	}

	table.PrintTable(cmd.log, header, values)
	return nil
}
