package pro

import (
	"context"
	"fmt"

	loftctlflags "github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/start"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
	"github.com/loft-sh/log/terminal"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/find"
	"github.com/loft-sh/vcluster/pkg/procli"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

type StartCmd struct {
	start.Options
}

func NewStartCmd(loftctlGlobalFlags *loftctlflags.GlobalFlags) (*cobra.Command, error) {
	cmd := &StartCmd{
		Options: start.Options{
			GlobalFlags: loftctlGlobalFlags,
			Log:         log.GetInstance(),
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
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			return cmd.Run(cobraCmd.Context())
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

func (cmd *StartCmd) Run(ctx context.Context) error {
	// get version to deploy
	if cmd.Version == "latest" || cmd.Version == "" {
		cmd.Version = procli.MinimumVersionTag

		latestVersion, err := procli.LatestCompatibleVersion(context.TODO())
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
	if previousContext == "" {
		_, _, previousContext = find.VClusterProFromContext(rawConfig.CurrentContext)
	}
	if previousContext != "" {
		if terminal.IsTerminalIn {
			switchBackOption := "No, switch back to context " + previousContext
			out, err := cmd.Log.Question(&survey.QuestionOptions{
				Question:     "You are trying to create vCluster.Pro inside another vcluster, is this desired?",
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
			cmd.Log.Warnf("You are trying to create vCluster.Pro inside another vcluster, is this desired?")
		}
	}

	return start.NewLoftStarter(cmd.Options).Start(ctx)
}
