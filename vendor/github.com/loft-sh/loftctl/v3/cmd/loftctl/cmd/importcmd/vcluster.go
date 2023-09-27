package importcmd

import (
	"context"

	"github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/loftctl/v3/pkg/util"
	"github.com/loft-sh/log"
	"github.com/mgutz/ansi"

	managementv1 "github.com/loft-sh/api/v3/pkg/apis/management/v1"
	"github.com/loft-sh/api/v3/pkg/product"

	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/upgrade"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type VClusterCmd struct {
	*flags.GlobalFlags

	VClusterClusterName string
	VClusterNamespace   string
	Project             string
	ImportName          string
	UpgradeToPro        bool

	Log log.Logger
}

// NewVClusterCmd creates a new command
func NewVClusterCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &VClusterCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	description := product.ReplaceWithHeader("import vcluster", `
Imports a vcluster into a Loft project.

Example:
loft import vcluster my-vcluster --cluster connected-cluster my-vcluster \
  --namespace vcluster-my-vcluster --project my-project --importname my-vcluster
#######################################################
	`)
	if upgrade.IsPlugin == "true" {
		description = `
#######################################################
################ devspace import vcluster #############
#######################################################
Imports a vcluster into a Loft project.

Example:
devspace import vcluster my-vcluster --cluster connected-cluster my-vcluster \
  --namespace vcluster-my-vcluster --project my-project --importname my-vcluster
#######################################################
	`
	}
	c := &cobra.Command{
		Use:   "vcluster" + util.VClusterNameOnlyUseLine,
		Short: "Imports a vcluster into a Loft project",
		Long:  description,
		Args:  util.VClusterNameOnlyValidator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	c.Flags().StringVar(&cmd.VClusterClusterName, "cluster", "", "Cluster name of the cluster the virtual cluster is running on")
	c.Flags().StringVar(&cmd.VClusterNamespace, "namespace", "", "The namespace of the vcluster")
	c.Flags().StringVar(&cmd.Project, "project", "", "The project to import the vcluster into")
	c.Flags().StringVar(&cmd.ImportName, "importname", "", "The name of the vcluster under projects. If unspecified, will use the vcluster name")
	c.Flags().BoolVar(&cmd.UpgradeToPro, "pro-upgrade", false, "If true, will upgrade the vcluster to vCluster.Pro upon import")

	_ = c.MarkFlagRequired("cluster")
	_ = c.MarkFlagRequired("namespace")
	_ = c.MarkFlagRequired("project")

	return c
}

func (cmd *VClusterCmd) Run(ctx context.Context, args []string) error {
	// Get vclusterName from command argument
	var vclusterName string = args[0]

	baseClient, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return err
	}

	err = client.VerifyVersion(baseClient)
	if err != nil {
		return err
	}

	managementClient, err := baseClient.Management()
	if err != nil {
		return err
	}

	if _, err = managementClient.Loft().ManagementV1().Projects().ImportVirtualCluster(ctx, cmd.Project, &managementv1.ProjectImportVirtualCluster{
		SourceVirtualCluster: managementv1.ProjectImportVirtualClusterSource{
			Name:       vclusterName,
			Namespace:  cmd.VClusterNamespace,
			Cluster:    cmd.VClusterClusterName,
			ImportName: cmd.ImportName,
		},
		UpgradeToPro: cmd.UpgradeToPro,
	}, metav1.CreateOptions{}); err != nil {
		return err
	}

	cmd.Log.Donef("Successfully imported vcluster %s into project %s", ansi.Color(vclusterName, "white+b"), ansi.Color(cmd.Project, "white+b"))

	return nil
}
