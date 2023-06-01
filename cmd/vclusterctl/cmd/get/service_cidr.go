package get

import (
	"fmt"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/loft-sh/vcluster/pkg/util/servicecidr"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type serviceCIDRCmd struct {
	*flags.GlobalFlags
	log log.Logger
}

func getServiceCIDR(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &serviceCIDRCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "service-cidr",
		Short: "Prints Service CIDR of the cluster",
		Long: `
#######################################################
############### vcluster get service-cidr  ############
#######################################################
Prints Service CIDR of the cluster

Ex: 
vcluster get service-cidr
10.96.0.0/12
#######################################################
	`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd)
		}}

	return cobraCmd
}

func (cmd *serviceCIDRCmd) Run(cobraCmd *cobra.Command) error {
	// first load the kube config
	kubeClientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{
		CurrentContext: cmd.Context,
	})

	// load the rest config
	kubeConfig, err := kubeClientConfig.ClientConfig()
	if err != nil {
		return fmt.Errorf("there is an error loading your current kube config (%v), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}

	client, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}

	if cmd.Namespace == "" {
		cmd.Namespace, _, err = kubeClientConfig.Namespace()
		if err != nil {
			return err
		} else if cmd.Namespace == "" {
			cmd.Namespace = "default"
		}
	}

	cidr, warning := servicecidr.GetServiceCIDR(cobraCmd.Context(), client, cmd.Namespace)
	if warning != "" {
		cmd.log.Debugf(warning)
	}

	_, err = cmd.log.Write([]byte(cidr))
	if err != nil {
		return err
	}

	return nil
}
