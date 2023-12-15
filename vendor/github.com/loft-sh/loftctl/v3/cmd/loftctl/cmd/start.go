package cmd

import (
	"context"

	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/start"
	"github.com/loft-sh/loftctl/v3/pkg/upgrade"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// StartCmd holds the cmd flags
type StartCmd struct {
	start.Options
}

// NewStartCmd creates a new command
func NewStartCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &StartCmd{
		Options: start.Options{
			GlobalFlags: globalFlags,
			Log:         log.GetInstance(),
		},
	}

	startCmd := &cobra.Command{
		Use:   "start",
		Short: product.Replace("Start a loft instance and connect via port-forwarding"),
		Long: product.ReplaceWithHeader("start", `
Starts a loft instance in your Kubernetes cluster and
then establishes a port-forwarding connection.

Please make sure you meet the following requirements
before running this command:

1. Current kube-context has admin access to the cluster
2. Helm v3 must be installed
3. kubectl must be installed

########################################################
	`),
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.Run(cobraCmd.Context())
		},
	}

	startCmd.Flags().BoolVar(&cmd.Docker, "docker", false, product.Replace("If enabled will try to deploy Loft to the local docker installation."))
	startCmd.Flags().StringVar(&cmd.DockerImage, "docker-image", "", product.Replace("The docker image to install."))
	startCmd.Flags().StringArrayVar(&cmd.DockerArgs, "docker-arg", []string{}, product.Replace("Extra docker args for running Loft."))
	startCmd.Flags().StringVar(&cmd.Context, "context", "", "The kube context to use for installation")
	startCmd.Flags().StringVar(&cmd.Namespace, "namespace", "loft", product.Replace("The namespace to install loft into"))
	startCmd.Flags().StringVar(&cmd.LocalPort, "local-port", "", "The local port to bind to if using port-forwarding")
	startCmd.Flags().StringVar(&cmd.Host, "host", "", "Provide a hostname to enable ingress and configure its hostname")
	startCmd.Flags().StringVar(&cmd.Password, "password", "", "The password to use for the admin account. (If empty this will be the namespace UID)")
	startCmd.Flags().StringVar(&cmd.Version, "version", upgrade.GetVersion(), product.Replace("The loft version to install"))
	startCmd.Flags().StringVar(&cmd.Values, "values", "", product.Replace("Path to a file for extra loft helm chart values"))
	startCmd.Flags().BoolVar(&cmd.ReuseValues, "reuse-values", true, product.Replace("Reuse previous Loft helm values on upgrade"))
	startCmd.Flags().BoolVar(&cmd.Upgrade, "upgrade", false, product.Replace("If true, Loft will try to upgrade the release"))
	startCmd.Flags().StringVar(&cmd.Email, "email", "", "The email to use for the installation")
	startCmd.Flags().BoolVar(&cmd.Reset, "reset", false, product.Replace("If true, an existing loft instance will be deleted before installing loft"))
	startCmd.Flags().BoolVar(&cmd.NoWait, "no-wait", false, product.Replace("If true, loft will not wait after installing it"))
	startCmd.Flags().BoolVar(&cmd.NoPortForwarding, "no-port-forwarding", false, product.Replace("If true, loft will not do port forwarding after installing it"))
	startCmd.Flags().BoolVar(&cmd.NoTunnel, "no-tunnel", false, product.Replace("If true, loft will not create a loft.host tunnel for this installation"))
	startCmd.Flags().BoolVar(&cmd.NoLogin, "no-login", false, product.Replace("If true, loft will not login to a loft instance on start"))
	startCmd.Flags().StringVar(&cmd.ChartPath, "chart-path", "", product.Replace("The local chart path to deploy Loft"))
	startCmd.Flags().StringVar(&cmd.ChartRepo, "chart-repo", "https://charts.loft.sh/", product.Replace("The chart repo to deploy Loft"))
	startCmd.Flags().StringVar(&cmd.ChartName, "chart-name", "loft", product.Replace("The chart name to deploy Loft"))
	startCmd.Flags().StringVar(&cmd.Product, "product", "", product.Replace("The Loft product to install"))
	return startCmd
}

// Run executes the functionality "loft start"
func (cmd *StartCmd) Run(ctx context.Context) error {
	return start.NewLoftStarter(cmd.Options).Start(ctx)
}
