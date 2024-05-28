package platform

import (
	"context"
	"fmt"

	"github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
	"github.com/loft-sh/log/terminal"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/start"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

type StartCmd struct {
	start.Options
}

func NewStartCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &StartCmd{
		Options: start.Options{
			GlobalFlags: globalFlags,
			Log:         log.GetInstance(),
		},
	}

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start a vCluster platform instance and connect via port-forwarding",
		Long: `########################################################
############# vcluster platform start ##################
########################################################

Starts a vCluster platform instance in your Kubernetes cluster
and then establishes a port-forwarding connection.

Please make sure you meet the following requirements
before running this command:

1. Current kube-context has admin access to the cluster
2. Helm v3 must be installed
3. kubectl must be installed

########################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	startCmd.Flags().StringVar(&cmd.Context, "context", "", "The kube context to use for installation")
	startCmd.Flags().StringVar(&cmd.Namespace, "namespace", "vcluster-platform", "The namespace to install vCluster platform into")
	startCmd.Flags().StringVar(&cmd.LocalPort, "local-port", "", "The local port to bind to if using port-forwarding")
	startCmd.Flags().StringVar(&cmd.Host, "host", "", "Provide a hostname to enable ingress and configure its hostname")
	startCmd.Flags().StringVar(&cmd.Password, "password", "", "The password to use for the admin account. (If empty this will be the namespace UID)")
	startCmd.Flags().StringVar(&cmd.Version, "version", "latest", "The vCluster platform version to install")
	startCmd.Flags().StringVar(&cmd.Values, "values", "", "Path to a file for extra vCluster platform helm chart values")
	startCmd.Flags().BoolVar(&cmd.ReuseValues, "reuse-values", true, "Reuse previous vCluster platform helm values on upgrade")
	startCmd.Flags().BoolVar(&cmd.Upgrade, "upgrade", false, "If true, vCluster platform will try to upgrade the release")
	startCmd.Flags().StringVar(&cmd.Email, "email", "", "The email to use for the installation")
	startCmd.Flags().BoolVar(&cmd.Reset, "reset", false, "If true, an existing vCluster platform instance will be deleted before installing vCluster platform")
	startCmd.Flags().BoolVar(&cmd.NoWait, "no-wait", false, "If true, vCluster platform will not wait after installing it")
	startCmd.Flags().BoolVar(&cmd.NoPortForwarding, "no-port-forwarding", false, "If true, vCluster platform will not do port forwarding after installing it")
	startCmd.Flags().BoolVar(&cmd.NoTunnel, "no-tunnel", false, "If true, vCluster platform will not create a loft.host tunnel for this installation")
	startCmd.Flags().BoolVar(&cmd.NoLogin, "no-login", false, "If true, vCluster platform will not login to a vCluster platform instance on start")
	startCmd.Flags().StringVar(&cmd.ChartPath, "chart-path", "", "The vCluster platform chart path to deploy vCluster platform")
	startCmd.Flags().StringVar(&cmd.ChartRepo, "chart-repo", "https://charts.loft.sh/", "The chart repo to deploy vCluster platform")
	startCmd.Flags().StringVar(&cmd.ChartName, "chart-name", "vcluster-platform", "The chart name to deploy vCluster platform")

	return startCmd
}

func (cmd *StartCmd) Run(ctx context.Context) error {
	// get version to deploy
	if cmd.Version == "latest" || cmd.Version == "" {
		cmd.Version = platform.MinimumVersionTag
		latestVersion, err := platform.LatestCompatibleVersion(ctx)
		if err == nil {
			cmd.Version = latestVersion
		}
	}
	// make sure we are in the correct context
	// first load the kube config
	kubeClientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{
		CurrentContext: cmd.Context,
	})
	// load the raw config
	rawConfig, err := kubeClientConfig.RawConfig()
	if err != nil {
		return fmt.Errorf("there is an error loading your current kube config (%w), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}
	if cmd.Context != "" {
		rawConfig.CurrentContext = cmd.Context
	}
	// check if vcluster in vcluster
	_, _, previousContext := find.VClusterFromContext(rawConfig.CurrentContext)
	if previousContext != "" {
		if terminal.IsTerminalIn {
			switchBackOption := "No, switch back to context " + previousContext
			out, err := cmd.Log.Question(&survey.QuestionOptions{
				Question:     "You are trying to create vCluster platform inside another vcluster, is this desired?",
				DefaultValue: switchBackOption,
				Options:      []string{switchBackOption, "Yes"},
			})
			if err != nil {
				return err
			}
			if out == switchBackOption {
				cmd.Context = previousContext
				kubeClientConfig = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{
					CurrentContext: cmd.Context,
				})
				rawConfig, err = kubeClientConfig.RawConfig()
				if err != nil {
					return fmt.Errorf("there is an error loading your current kube config (%w), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
				}
				err = find.SwitchContext(&rawConfig, cmd.Context)
				if err != nil {
					return fmt.Errorf("switch context: %w", err)
				}
			}
		} else {
			cmd.Log.Warnf("You are trying to create vCluster platform inside another vcluster, is this desired?")
		}
	}

	return start.NewLoftStarter(cmd.Options).Start(ctx)
}
