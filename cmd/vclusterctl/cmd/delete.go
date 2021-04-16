package cmd

import (
	"fmt"
	"os/exec"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

// DeleteCmd holds the login cmd flags
type DeleteCmd struct {
	*flags.GlobalFlags

	Namespace string
	log       log.Logger
}

// NewDeleteCmd creates a new command
func NewDeleteCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &DeleteCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "delete",
		Short: "Deletes a virtual cluster",
		Long: `
#######################################################
################### vcluster delete ###################
#######################################################
Deletes a virtual cluster

Example:
vcluster delete test --namespace test
#######################################################
	`,
		Args: cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd, args)
		},
	}

	cobraCmd.Flags().StringVarP(&cmd.Namespace, "namespace", "n", "", "The namespace the vcluster was created in")
	return cobraCmd
}

// Run executes the functionality
func (cmd *DeleteCmd) Run(cobraCmd *cobra.Command, args []string) error {
	// test for helm
	_, err := exec.LookPath("helm")
	if err != nil {
		return fmt.Errorf("seems like helm is not installed. Helm is required for the deletion of a virtual cluster. Please visit https://helm.sh/docs/intro/install/ for install instructions")
	}

	output, err := exec.Command("helm", "version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("Seems like there are issues with your helm client: \n\n%s", output)
	}

	// first load the kube config
	kubeClientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})

	// load the raw config
	rawConfig, err := kubeClientConfig.RawConfig()
	if err != nil {
		return fmt.Errorf("there is an error loading your current kube config (%v), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}

	namespace, _, err := kubeClientConfig.Namespace()
	if err != nil {
		return err
	} else if namespace == "" {
		namespace = "default"
	}
	if cmd.Namespace != "" {
		namespace = cmd.Namespace
	}

	// we have to upgrade / install the chart
	err = helm.NewClient(&rawConfig, cmd.log).Delete(args[0], namespace)
	if err != nil {
		return err
	}

	cmd.log.Donef("Successfully deleted virtual cluster %s in namespace %s", args[0], namespace)
	return nil
}
