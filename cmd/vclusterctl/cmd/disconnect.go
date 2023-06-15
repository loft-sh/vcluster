package cmd

import (
	"fmt"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/find"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log/survey"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// DisconnectCmd holds the disconnect cmd flags
type DisconnectCmd struct {
	*flags.GlobalFlags

	log log.Logger
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
		return fmt.Errorf("current selected context \"%s\" is not a vcluster context. If you've used a custom context name you will need to switch manually using kubectl", otherContext)
	}

	if otherContext == "" {
		otherContext, err = cmd.selectContext(&rawConfig, otherContext)
		if err != nil {
			return err
		}
	}

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

func (cmd *DisconnectCmd) selectContext(kubeConfig *clientcmdapi.Config, currentContext string) (string, error) {
	availableContexts := []string{}
	for context := range kubeConfig.Contexts {
		if context != currentContext {
			availableContexts = append(availableContexts, context)
		}
	}

	cmd.log.Warn("Unable to determine old context")
	options := &survey.QuestionOptions{
		Question: "Please select a new context to switch to:",
		Options:  availableContexts,
	}
	answer, err := cmd.log.Question(options)
	if err != nil {
		return "", err
	}

	return answer, nil
}
