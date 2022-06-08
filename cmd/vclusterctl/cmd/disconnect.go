package cmd

import (
	"fmt"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/find"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// DisconnectCmd holds the disconnect cmd flags
type DisconnectCmd struct {
	*flags.GlobalFlags

	rawConfig  *clientcmdapi.Config
	restConfig *rest.Config
	log        log.Logger
}

// NewDisconnectCmd creates a new command
func NewDisconnectCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &DisconnectCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "disconnect",
		Short: "Disconnects from a virtual cluster",
		Long: `
#######################################################
################# vcluster disconnect #################
#######################################################
Disconnect switches back the kube context if
vcluster connect --update-current was used

Example:
vcluster connect --update-current
vcluster disconnect
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd, args)
		},
	}

	return cobraCmd
}

// Run executes the functionality
func (cmd *DisconnectCmd) Run(cobraCmd *cobra.Command, args []string) error {
	if cmd.Context == "" {
		rawConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).RawConfig()
		if err != nil {
			return err
		}

		cmd.Context = rawConfig.CurrentContext
	}

	// get vcluster info from context
	vClusterName, vClusterNamespace, _ := find.VClusterFromContext(cmd.Context)
	if vClusterName == "" {
		return fmt.Errorf("current selected context is not a vcluster context")
	}

	// disconnect
	err := cmd.disconnect(vClusterName, vClusterNamespace)
	if err != nil {
		return err
	}

	return nil
}

func (cmd *DisconnectCmd) disconnect(vClusterName, vClusterNamespace string) error {
	vCluster, err := find.GetVCluster(cmd.Context, vClusterName, vClusterNamespace)
	if err != nil {
		return err
	}

	// load the raw config
	rawConfig, err := vCluster.ClientFactory.RawConfig()
	if err != nil {
		return fmt.Errorf("there is an error loading your current kube config (%v), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}
	err = switchContext(&rawConfig, vCluster.Context)
	if err != nil {
		return errors.Wrap(err, "delete kube context")
	}

	rawConfig.CurrentContext = vCluster.Context
	restConfig, err := vCluster.ClientFactory.ClientConfig()
	if err != nil {
		return err
	}

	cmd.Namespace = vCluster.Namespace
	cmd.rawConfig = &rawConfig
	cmd.restConfig = restConfig
	return nil
}

func switchContext(kubeConfig *clientcmdapi.Config, otherContext string) error {
	kubeConfig.CurrentContext = otherContext
	return clientcmd.ModifyConfig(clientcmd.NewDefaultClientConfigLoadingRules(), *kubeConfig, false)
}
