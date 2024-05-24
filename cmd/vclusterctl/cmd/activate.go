package cmd

import (
	loftctlUtil "github.com/loft-sh/loftctl/v4/pkg/util"
	"github.com/loft-sh/log"
	platformcmd "github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"
)

func NewActivateCmd(globalFlags *flags.GlobalFlags) (*cobra.Command, error) {
	cmd := &platformcmd.ImportCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	description := `########################################################
################### vcluster activate ####################
########################################################
Imports a vCluster into a vCluster platform project.

Example:
vcluster activate my-vcluster --cluster connected-cluster \
--namespace vcluster-my-vcluster --project my-project --import-name my-vcluster
########################################################
	`

	importCmd := &cobra.Command{
		Use:     "activate" + loftctlUtil.VClusterNameOnlyUseLine,
		Aliases: []string{"import"},
		Short:   "Imports a vCluster into a vCluster platform project",
		Long:    description,
		Args:    loftctlUtil.VClusterNameOnlyValidator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	importCmd.Flags().StringVar(&cmd.Manager, "manager", "", "The manager to use for managing the virtual cluster, can be either helm or platform.")

	importCmd.Flags().StringVar(&cmd.ClusterName, "cluster", "", "Cluster name of the cluster the virtual cluster is running on")
	importCmd.Flags().StringVar(&cmd.Project, "project", "", "The project to import the vCluster into")
	importCmd.Flags().StringVar(&cmd.ImportName, "import-name", "", "The name of the vCluster under projects. If unspecified, will use the vcluster name")

	return importCmd, nil
}
