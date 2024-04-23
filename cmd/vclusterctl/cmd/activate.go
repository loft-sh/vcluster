package cmd

import (
	"os"

	loftctlUtil "github.com/loft-sh/loftctl/v3/pkg/util"
	"github.com/loft-sh/log"
	platformcmd "github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform"
	"github.com/loft-sh/vcluster/config"
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
#######################################################
	`

	importCmd := &cobra.Command{
		Use:     "activate" + loftctlUtil.VClusterNameOnlyUseLine,
		Aliases: []string{"import"},
		Short:   "Imports a vCluster into a vCluster platform project",
		Long:    description,
		Args:    loftctlUtil.VClusterNameOnlyValidator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			if config.ShouldCheckForProFeatures() {
				cmd.Log.Warnf("In order to use a Pro feature, please contact us at https://www.vcluster.com/pro-demo or downgrade by running `vcluster upgrade --version v0.19.5`")
				os.Exit(1)
			}

			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	importCmd.Flags().StringVar(&cmd.Manager, "manager", "", "The manager to use for managing the virtual cluster, can be either helm or platform.")

	importCmd.Flags().StringVar(&cmd.ClusterName, "cluster", "", "Cluster name of the cluster the virtual cluster is running on")
	importCmd.Flags().StringVar(&cmd.Project, "project", "", "The project to import the vCluster into")
	importCmd.Flags().StringVar(&cmd.ImportName, "import-name", "", "The name of the vCluster under projects. If unspecified, will use the vcluster name")

	return importCmd, nil
}
