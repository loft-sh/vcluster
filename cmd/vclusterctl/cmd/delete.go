package cmd

import (
	"context"
	"fmt"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
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
		Use:   "delete" + util.VClusterNameOnlyUseLine,
		Short: "Deletes a virtual cluster",
		Long: `#######################################################
################### vcluster delete ###################
#######################################################
Deletes a virtual cluster

Example:
vcluster delete test --namespace test
#######################################################
	`,
		Args:              util.VClusterNameOnlyValidator,
		Aliases:           []string{"rm"},
		ValidArgsFunction: newValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.Manager, "manager", "", "The manager to use for managing the virtual cluster, can be either helm or platform.")

	cobraCmd.Flags().BoolVar(&cmd.Wait, "wait", true, "If enabled, vcluster will wait until the vcluster is deleted")
	cobraCmd.Flags().BoolVar(&cmd.DeleteConfigMap, "delete-configmap", false, "If enabled, vCluster will delete the ConfigMap of the vCluster")
	cobraCmd.Flags().BoolVar(&cmd.KeepPVC, "keep-pvc", false, "If enabled, vcluster will not delete the persistent volume claim of the vcluster")
	cobraCmd.Flags().BoolVar(&cmd.DeleteNamespace, "delete-namespace", false, "If enabled, vcluster will delete the namespace of the vcluster. In the case of multi-namespace mode, will also delete all other namespaces created by vcluster")
	cobraCmd.Flags().BoolVar(&cmd.DeleteContext, "delete-context", true, "If the corresponding kube context should be deleted if there is any")
	cobraCmd.Flags().BoolVar(&cmd.AutoDeleteNamespace, "auto-delete-namespace", true, "If enabled, vcluster will delete the namespace of the vcluster if it was created by vclusterctl. In the case of multi-namespace mode, will also delete all other namespaces created by vcluster")
	cobraCmd.Flags().BoolVar(&cmd.IgnoreNotFound, "ignore-not-found", false, "If enabled, vcluster will not error out in case the target vcluster does not exist")

	// Platform flags
	cobraCmd.Flags().StringVar(&cmd.Project, "project", "", "[PLATFORM] The vCluster platform project to use")

	return cobraCmd
}

// Run executes the functionality
func (cmd *DeleteCmd) Run(ctx context.Context, args []string) error {
	cfg := cmd.LoadedConfig(cmd.log)

	// If manager has been passed as flag use it, otherwise read it from the config file
	var manager string
	if cmd.Manager != "" {
		manager = cmd.Manager
	} else {
		manager = string(cfg.Manager.Type)
	}

	managerType, err := config.ParseManagerType(manager)
	if err != nil {
		return fmt.Errorf("parse manager type: %w", err)
	}

	// check if there is a platform client or we skip the info message
	_, err = platform.InitClientFromConfig(ctx, cfg)
	if err == nil {
		config.PrintManagerInfo("delete", cfg.Manager.Type, cmd.log)
	}

	if managerType == config.ManagerPlatform {
		return cli.DeletePlatform(ctx, &cmd.DeleteOptions, cfg, args[0], cmd.log)
	}

	return cli.DeleteHelm(ctx, &cmd.DeleteOptions, cmd.GlobalFlags, args[0], cmd.log)
}
