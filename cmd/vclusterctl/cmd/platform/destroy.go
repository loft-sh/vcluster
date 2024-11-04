package platform

import (
	"context"
	"errors"
	"fmt"

	"github.com/loft-sh/log"
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
		Short: "Destroy a vCluster platform instance",
		Long: `########################################################
############# vcluster platform destroy ##################
########################################################

Destroys a vCluster platform instance in your Kubernetes cluster.

Please make sure you meet the following requirements
before running this command:

1. Current kube-context has admin access to the cluster
2. Helm v3 must be installed


VirtualClusterInstances managed with driver helm will be deleted, but the underlying virtual cluster will not be uninstalled

########################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	destroyCmd.Flags().StringVar(&cmd.Context, "context", "", "The kube context to use for installation")
	destroyCmd.Flags().StringVar(&cmd.Namespace, "namespace", "", "The namespace vCluster platform is installed in")
	destroyCmd.Flags().BoolVar(&cmd.DeleteNamespace, "delete-namespace", true, "Whether to delete the namespace or not")
	destroyCmd.Flags().BoolVar(&cmd.IgnoreNotFound, "ignore-not-found", false, "Exit successfully if platform installation is not found")
	destroyCmd.Flags().BoolVar(&cmd.Force, "force", false, "Try uninstalling even if the platform is not installed. '--namespace' is required if true")
	destroyCmd.Flags().IntVar(&cmd.TimeoutMinutes, "timeout-minutes", 5, "How long to try deleting the platform before giving up")

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
			return fmt.Errorf("vCluster platform may not be installed: %w", err)
		}
		cmd.Log.Infof("found platform installation in namespace %q", namespace)
		cmd.Namespace = namespace
	}

	found, err := clihelper.IsLoftAlreadyInstalled(ctx, cmd.KubeClient, cmd.Namespace)
	if err != nil {
		return fmt.Errorf("vCluster platform may not be installed: %w", err)
	}
	shouldForce := cmd.Force && cmd.Namespace != ""
	if !found && !shouldForce {
		if cmd.IgnoreNotFound {
			cmd.Log.Info("no platform installation found")
			return nil
		}
		return fmt.Errorf("platform not installed in namespace %q", cmd.Namespace)
	}

	err = destroy.Destroy(ctx, cmd.DeleteOptions)
	if err != nil {
		return fmt.Errorf("failed to destroy platform: %w", err)
	}
	return nil
}
