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

	rawConfig  *clientcmdapi.Config // nolint:unused
	restConfig *rest.Config         // nolint:unused
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
	rawConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{
		CurrentContext: cmd.Context,
	}).RawConfig()
	if err != nil {
		return err
	}
	if cmd.Context == "" {
		cmd.Context = rawConfig.CurrentContext
	}

	// get vcluster info from context
	vClusterName, _, otherContext := find.VClusterFromContext(cmd.Context)
	if vClusterName == "" {
		return fmt.Errorf("current selected context is not a vcluster context")
	}

	// disconnect
	err = switchContext(&rawConfig, otherContext)
	if err != nil {
		return errors.Wrap(err, "switch kube context")
	}

	cmd.log.Infof("Successfully disconnected from vcluster: %s and switched back to the original context: %s", vClusterName, otherContext)
	return nil
}

func switchContext(kubeConfig *clientcmdapi.Config, otherContext string) error {
	kubeConfig.CurrentContext = otherContext
	return clientcmd.ModifyConfig(clientcmd.NewDefaultClientConfigLoadingRules(), *kubeConfig, false)
}
