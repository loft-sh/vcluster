package cmd

import (
	loftctlUtil "github.com/loft-sh/loftctl/v4/pkg/util"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/spf13/cobra"
)

// DeleteCmd holds the delete cmd flags
type DeleteCmd struct {
	*flags.GlobalFlags
	cli.DeleteOptions

	log log.Logger
}

// NewDeleteCmd creates a new command
func NewDeleteCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &DeleteCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "delete" + loftctlUtil.VClusterNameOnlyUseLine,
		Short: "Deletes a virtual cluster",
		Long: `
#######################################################
################### vcluster delete ###################
#######################################################
Deletes a virtual cluster

Example:
vcluster delete test --namespace test
#######################################################
	`,
		Args:              loftctlUtil.VClusterNameOnlyValidator,
		Aliases:           []string{"rm"},
		ValidArgsFunction: newValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd, args)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.Manager, "manager", "", "The manager to use for managing the virtual cluster, can be either helm or platform.")

	cobraCmd.Flags().BoolVar(&cmd.Wait, "wait", true, "If enabled, vcluster will wait until the vcluster is deleted")
	cobraCmd.Flags().BoolVar(&cmd.DeleteConfigMap, "delete-configmap", false, "If enabled, vCluster will delete the ConfigMap of the vCluster")
	cobraCmd.Flags().BoolVar(&cmd.KeepPVC, "keep-pvc", false, "If enabled, vcluster will not delete the persistent volume claim of the vcluster")
	cobraCmd.Flags().BoolVar(&cmd.DeleteNamespace, "delete-namespace", false, "If enabled, vcluster will delete the namespace of the vcluster. In the case of multi-namespace mode, will also delete all other namespaces created by vcluster")
	cobraCmd.Flags().BoolVar(&cmd.AutoDeleteNamespace, "auto-delete-namespace", true, "If enabled, vcluster will delete the namespace of the vcluster if it was created by vclusterctl. In the case of multi-namespace mode, will also delete all other namespaces created by vcluster")
	cobraCmd.Flags().BoolVar(&cmd.IgnoreNotFound, "ignore-not-found", false, "If enabled, vcluster will not error out in case the target vcluster does not exist")

	// Platform flags
	cobraCmd.Flags().StringVar(&cmd.Project, "project", "", "[Platform] The vCluster platform project to use")

	return cobraCmd
}

// Run executes the functionality
func (cmd *DeleteCmd) Run(cobraCmd *cobra.Command, args []string) error {
	manager, err := platform.GetManager(cmd.Manager)
	if err != nil {
		return err
	}

	// check if we should delete a platform vCluster
	platform.PrintManagerInfo("delete", manager, cmd.log)
	if manager == platform.ManagerPlatform {
		// deploy platform cluster
		return cli.DeletePlatform(cobraCmd.Context(), &cmd.DeleteOptions, args[0], cmd.log)
	}

	return cli.DeleteHelm(cobraCmd.Context(), &cmd.DeleteOptions, cmd.GlobalFlags, args[0], cmd.log)
}
