package cmd

import (
	"context"
	"fmt"

	loftctlUtil "github.com/loft-sh/loftctl/v4/pkg/util"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/spf13/cobra"
)

// ConnectCmd holds the cmd flags
type ConnectCmd struct {
	*flags.GlobalFlags
	cli.ConnectOptions

	Log log.Logger
}

// NewConnectCmd creates a new command
func NewConnectCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ConnectCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	useLine, nameValidator := loftctlUtil.NamedPositionalArgsValidator(true, false, "VCLUSTER_NAME")

	cobraCmd := &cobra.Command{
		Use:   "connect" + useLine,
		Short: "Connect to a virtual cluster",
		Long: `
#######################################################
################## vcluster connect ###################
#######################################################
Connect to a virtual cluster

Example:
vcluster connect test --namespace test
# Open a new bash with the vcluster KUBECONFIG defined
vcluster connect test -n test -- bash
vcluster connect test -n test -- kubectl get ns
#######################################################
	`,
		Args:              nameValidator,
		ValidArgsFunction: newValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.Manager, "manager", "", "The manager to use for managing the virtual cluster, can be either helm or platform.")

	cobraCmd.Flags().StringVar(&cmd.KubeConfigContextName, "kube-config-context-name", "", "If set, will override the context name of the generated virtual cluster kube config with this name")
	cobraCmd.Flags().StringVar(&cmd.KubeConfig, "kube-config", "./kubeconfig.yaml", "Writes the created kube config to this file")
	cobraCmd.Flags().BoolVar(&cmd.UpdateCurrent, "update-current", true, "If true updates the current kube config")
	cobraCmd.Flags().BoolVar(&cmd.Print, "print", false, "When enabled prints the context to stdout")
	cobraCmd.Flags().StringVar(&cmd.PodName, "pod", "", "The pod to connect to")
	cobraCmd.Flags().StringVar(&cmd.Server, "server", "", "The server to connect to")
	cobraCmd.Flags().IntVar(&cmd.LocalPort, "local-port", 0, "The local port to forward the virtual cluster to. If empty, vCluster will use a random unused port")
	cobraCmd.Flags().StringVar(&cmd.Address, "address", "", "The local address to start port forwarding under")
	cobraCmd.Flags().StringVar(&cmd.ServiceAccount, "service-account", "", "If specified, vCluster will create a service account token to connect to the virtual cluster instead of using the default client cert / key. Service account must exist and can be used as namespace/name.")
	cobraCmd.Flags().StringVar(&cmd.ServiceAccountClusterRole, "cluster-role", "", "If specified, vCluster will create the service account if it does not exist and also add a cluster role binding for the given cluster role to it. Requires --service-account to be set")
	cobraCmd.Flags().IntVar(&cmd.ServiceAccountExpiration, "token-expiration", 0, "If specified, vCluster will create the service account token for the given duration in seconds. Defaults to eternal")
	cobraCmd.Flags().BoolVar(&cmd.Insecure, "insecure", false, "If specified, vCluster will create the kube config with insecure-skip-tls-verify")
	cobraCmd.Flags().BoolVar(&cmd.BackgroundProxy, "background-proxy", false, "If specified, vCluster will create the background proxy in docker [its mainly used for vclusters with no nodeport service.]")

	// platform
	cobraCmd.Flags().StringVar(&cmd.Project, "project", "", "[PLATFORM] The platform project the vCluster is in")

	return cobraCmd
}

// Run executes the functionality
func (cmd *ConnectCmd) Run(ctx context.Context, args []string) error {
	vClusterName := ""
	if len(args) > 0 {
		vClusterName = args[0]
	}

	// validate flags
	err := cmd.validateFlags()
	if err != nil {
		return err
	}

	// get manager
	manager, err := platform.GetManager(cmd.Manager)
	if err != nil {
		return err
	}

	// is platform manager?
	if manager == platform.ManagerPlatform {
		return cli.ConnectPlatform(ctx, &cmd.ConnectOptions, cmd.GlobalFlags, vClusterName, args[1:], cmd.Log)
	}

	return cli.ConnectHelm(ctx, &cmd.ConnectOptions, cmd.GlobalFlags, vClusterName, args[1:], cmd.Log)
}

func (cmd *ConnectCmd) validateFlags() error {
	if cmd.ServiceAccountClusterRole != "" && cmd.ServiceAccount == "" {
		return fmt.Errorf("expected --service-account to be defined as well")
	}

	return nil
}
