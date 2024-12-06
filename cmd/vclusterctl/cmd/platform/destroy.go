package platform

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
	"github.com/loft-sh/log/terminal"
	"github.com/loft-sh/vcluster/pkg/cli/destroy"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/start"
	"github.com/loft-sh/vcluster/pkg/platform/clihelper"
	"github.com/spf13/cobra"
)

type DestroyCmd struct {
	destroy.DeleteOptions
}

func NewDestroyCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &DestroyCmd{
		DeleteOptions: destroy.DeleteOptions{
			Options: start.Options{
				GlobalFlags: globalFlags,
				Log:         log.GetInstance(),
				CommandName: "destroy",
			},
		},
	}

	destroyCmd := &cobra.Command{
		Use:   "destroy",
		Short: "Destroy a vCluster Platform instance",
		Long: `########################################################
############# vcluster platform destroy ##################
########################################################

Destroys a vCluster Platform instance in your Kubernetes cluster.

IMPORTANT: This action is done against the cluster the the kube-context is pointing to, and not the vCluster Platform instance that is logged in.
It does not require logging in to vCluster Platform.

Please make sure you meet the following requirements
before running this command:

1. Current kube-context has admin access to the cluster
2. Helm v3 must be installed


VirtualClusterInstances managed with driver helm will be deleted, but the underlying virtual cluster will not be uninstalled.

########################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			if cmd.NonInteractive {
				terminal.IsTerminalIn = false
			}
			return cmd.Run(cobraCmd.Context())
		},
	}

	destroyCmd.Flags().StringVar(&cmd.Context, "context", "", "The kube context to use for installation")
	destroyCmd.Flags().StringVar(&cmd.Namespace, "namespace", "", "The namespace vCluster Platform is installed in")
	destroyCmd.Flags().BoolVar(&cmd.DeleteNamespace, "delete-namespace", true, "Whether to delete the namespace or not")
	destroyCmd.Flags().BoolVar(&cmd.IgnoreNotFound, "ignore-not-found", false, "Exit successfully if platform installation is not found")
	destroyCmd.Flags().BoolVar(&cmd.Force, "force", false, "Try uninstalling even if the platform is not installed. '--namespace' is required if true")
	destroyCmd.Flags().BoolVar(&cmd.NonInteractive, "non-interactive", false, "Will not prompt for confirmation")
	destroyCmd.Flags().IntVar(&cmd.TimeoutMinutes, "timeout-minutes", 5, "How long to try deleting the platform before giving up. May increase when removing finalizers if --remove-finalizers is used")
	destroyCmd.Flags().BoolVar(&cmd.ForceRemoveFinalizers, "force-remove-finalizers", false, "IMPORTANT! Removing finalizers may cause unintended behaviours like leaving resources behind, but will ensure the platform is uninstalled.")

	return destroyCmd
}

func (cmd *DestroyCmd) Run(ctx context.Context) error {
	// initialise clients, verify binaries exist, sanity-check context
	err := cmd.Options.Prepare()
	if err != nil {
		return fmt.Errorf("failed to prepare clients: %w", err)
	}

	if cmd.Namespace == "" {
		namespace, err := clihelper.VClusterPlatformInstallationNamespace(ctx)
		if err != nil {
			if cmd.IgnoreNotFound && errors.Is(err, clihelper.ErrPlatformNamespaceNotFound) {
				cmd.Log.Info("no platform installation found")
				return nil
			}
			return fmt.Errorf("vCluster Platform may not be installed: %w", err)
		}
		cmd.Log.Infof("found platform installation in namespace %q", namespace)
		cmd.Namespace = namespace
	}

	found, err := clihelper.IsLoftAlreadyInstalled(ctx, cmd.KubeClient, cmd.Namespace)
	if err != nil {
		return fmt.Errorf("vCluster Platform may not be installed: %w", err)
	}
	shouldForce := cmd.Force && cmd.Namespace != ""
	if !found && !shouldForce {
		if cmd.IgnoreNotFound {
			cmd.Log.Info("no platform installation found")
			return nil
		}
		return fmt.Errorf("platform not installed in namespace %q", cmd.Namespace)
	}

	if terminal.IsTerminalIn {
		deleteOpt := "delete"
		out, err := cmd.Log.Question(&survey.QuestionOptions{
			Question: fmt.Sprintf("IMPORTANT! You are destroying the vCluster Platform installation in the namespace %q.\n  This may result in data loss.\n  Externally deployed virtual clusters depending on an external database connection will be irrecoverable after the platform is destroyed.\n  Please ensure your kube-context is pointed at the right cluster.\nPlease type %q to continue:", cmd.Namespace, deleteOpt),
		})
		if err != nil {
			return fmt.Errorf("failed to prompt for confirmation: %w", err)
		}
		if out != deleteOpt {
			cmd.Log.Info("destroy cancelled")
			return nil
		}
		if cmd.ForceRemoveFinalizers {
			forceRemoveOpt := "force-remove"
			out, err := cmd.Log.Question(&survey.QuestionOptions{
				Question: fmt.Sprintf("IMPORTANT! You have selected the --force-remove-finalizers option. Please ensure you understand the consequences. Removing finalizers may cause unintended behaviours like leaving resources behind, but will ensure the platform is uninstalled. To confirm, please type %q", forceRemoveOpt),
			})
			if err != nil {
				return fmt.Errorf("failed to prompt for confirmation: %w", err)
			}
			if out != forceRemoveOpt {
				cmd.Log.Info("destroy cancelled")
				return nil
			}
		}
	}

	err = destroy.Destroy(ctx, cmd.DeleteOptions)
	if err != nil {
		return fmt.Errorf("failed to destroy platform: %w", err)
	}

	cmd.Log.Infof("deleting platform config at %q", cmd.Config)
	cliConfig := cmd.LoadedConfig(cmd.Log)
	err = cliConfig.Delete()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && cmd.IgnoreNotFound {
			cmd.Log.Info("no platform config detected")
			return nil
		}
		return fmt.Errorf("failed to delete platform config: %w", err)
	}

	return nil
}
