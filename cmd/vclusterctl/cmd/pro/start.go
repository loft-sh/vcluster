package pro

import (
	"context"

	loftctl "github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd"
	loftctlflags "github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/start"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/spf13/cobra"
)

func NewStartCmd(loftctlGlobalFlags *loftctlflags.GlobalFlags) (*cobra.Command, error) {
	cmd := &loftctl.StartCmd{
		Options: start.Options{
			GlobalFlags: loftctlGlobalFlags,
			Log:         log.GetInstance(),
			Product:     "vcluster-pro",
		},
	}

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start a vCluster.Pro instance and connect via port-forwarding",
		Long: `########################################################
################## vcluster pro start ##################
########################################################

Starts a vCluster.Pro instance in your Kubernetes cluster
and then establishes a port-forwarding connection.

Please make sure you meet the following requirements
before running this command:

1. Current kube-context has admin access to the cluster
2. Helm v3 must be installed
3. kubectl must be installed

########################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			if cmd.Version == "latest" || cmd.Version == "" {
				cmd.Version = pro.MinimumVersionTag

				latestVersion, err := pro.LatestCompatibleVersion(context.TODO())
				if err == nil {
					cmd.Version = latestVersion
				}
			}

			return start.NewLoftStarter(cmd.Options).Start(cobraCmd.Context())
		},
	}

	startCmd.Flags().StringVar(&cmd.Context, "context", "", "The kube context to use for installation")
	startCmd.Flags().StringVar(&cmd.Namespace, "namespace", "vcluster-pro", "The namespace to install vCluster.Pro into")
	startCmd.Flags().StringVar(&cmd.LocalPort, "local-port", "", "The local port to bind to if using port-forwarding")
	startCmd.Flags().StringVar(&cmd.Host, "host", "", "Provide a hostname to enable ingress and configure its hostname")
	startCmd.Flags().StringVar(&cmd.Password, "password", "", "The password to use for the admin account. (If empty this will be the namespace UID)")
	startCmd.Flags().StringVar(&cmd.Version, "version", "latest", "The vCluster.Pro version to install")
	startCmd.Flags().StringVar(&cmd.Values, "values", "", "Path to a file for extra vCluster.Pro helm chart values")
	startCmd.Flags().BoolVar(&cmd.ReuseValues, "reuse-values", true, "Reuse previous vCluster.Pro helm values on upgrade")
	startCmd.Flags().BoolVar(&cmd.Upgrade, "upgrade", false, "If true, vCluster.Pro will try to upgrade the release")
	startCmd.Flags().StringVar(&cmd.Email, "email", "", "The email to use for the installation")
	startCmd.Flags().BoolVar(&cmd.Reset, "reset", false, "If true, an existing loft instance will be deleted before installing vCluster.Pro")
	startCmd.Flags().BoolVar(&cmd.NoWait, "no-wait", false, "If true, vCluster.Pro will not wait after installing it")
	startCmd.Flags().BoolVar(&cmd.NoPortForwarding, "no-port-forwarding", false, "If true, vCluster.Pro will not do port forwarding after installing it")
	startCmd.Flags().BoolVar(&cmd.NoTunnel, "no-tunnel", false, "If true, vCluster.Pro will not create a loft.host tunnel for this installation")
	startCmd.Flags().BoolVar(&cmd.NoLogin, "no-login", false, "If true, vCluster.Pro will not login to a vCluster.Pro instance on start")
	startCmd.Flags().StringVar(&cmd.ChartPath, "chart-path", "", "The vCluster.Pro chart path to deploy vCluster.Pro")
	startCmd.Flags().StringVar(&cmd.ChartRepo, "chart-repo", "https://charts.loft.sh/", "The chart repo to deploy vCluster.Pro")
	startCmd.Flags().StringVar(&cmd.ChartName, "chart-name", "vcluster-control-plane", "The chart name to deploy vCluster.Pro")

	return startCmd, nil
}
