package importcmd

import (
	"context"

	"github.com/loft-sh/loftctl/v4/pkg/util"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/mgutz/ansi"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"github.com/loft-sh/api/v4/pkg/product"

	"github.com/loft-sh/loftctl/v4/pkg/upgrade"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SpaceCmd struct {
	*flags.GlobalFlags

	ClusterName string
	Project     string
	ImportName  string

	log log.Logger
}

// NewSpaceCmd creates a new command
func NewSpaceCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &SpaceCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	description := product.ReplaceWithHeader("import space", `
Imports a space into a vCluster platform project.
Example:
vcluster platform import space my-space --cluster connected-cluster \
  --project my-project --importname my-space
#######################################################
	`)
	if upgrade.IsPlugin == "true" {
		description = `
#######################################################
################ devspace import space ################
#######################################################
Imports a space into a vCluster platform project.
Example:
devspace import space my-space --cluster connected-cluster \
  --project my-project --importname my-space
#######################################################
	`
	}
	c := &cobra.Command{
		Use:   "space" + util.SpaceNameOnlyUseLine,
		Short: product.Replace("Imports a space into a Loft project"),
		Long:  description,
		Args:  util.SpaceNameOnlyValidator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	c.Flags().StringVar(&cmd.ClusterName, "cluster", "", "Cluster name of the cluster from where the space is to be imported")
	c.Flags().StringVar(&cmd.Project, "project", "", "The project to import the space into")
	c.Flags().StringVar(&cmd.ImportName, "importname", "", "The name of the space under projects. If unspecified, will use the space name")

	_ = c.MarkFlagRequired("cluster")
	_ = c.MarkFlagRequired("project")

	return c
}

func (cmd *SpaceCmd) Run(ctx context.Context, args []string) error {
	// Get spaceName from command argument
	spaceName := args[0]

	platformClient, err := platform.InitClientFromConfig(ctx, cmd.LoadedConfig(cmd.log))
	if err != nil {
		return err
	}

	err = platform.VerifyVersion(platformClient)
	if err != nil {
		return err
	}

	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	if _, err = managementClient.Loft().ManagementV1().Projects().ImportSpace(ctx, cmd.Project, &managementv1.ProjectImportSpace{
		SourceSpace: managementv1.ProjectImportSpaceSource{
			Name:       spaceName,
			Cluster:    cmd.ClusterName,
			ImportName: cmd.ImportName,
		},
	}, metav1.CreateOptions{}); err != nil {
		return err
	}

	cmd.log.Donef("Successfully imported space %s into project %s", ansi.Color(spaceName, "white+b"), ansi.Color(cmd.Project, "white+b"))

	return nil
}
