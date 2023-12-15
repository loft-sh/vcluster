package pro

import (
	"context"
	"fmt"

	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/generate"
	"github.com/loft-sh/loftctl/v3/pkg/kubeconfig"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func NewGenerateCmd() *cobra.Command {
	description := `########################################################
################## vcluster pro generate ##################
########################################################
	`
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate configuration",
		Long:  description,
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(NewGenerateKubeConfigCmd())
	return cmd
}

// GenerateKubeConfigCmd holds the cmd flags
type GenerateKubeConfigCmd struct {
	log            log.Logger
	Namespace      string
	ServiceAccount string
}

// NewGenerateKubeConfigCmd creates a new command
func NewGenerateKubeConfigCmd() *cobra.Command {
	cmd := &GenerateKubeConfigCmd{
		log: log.GetInstance(),
	}

	c := &cobra.Command{
		Use:   "admin-kube-config",
		Short: "Generates a new kube config for connecting a cluster",
		Long: `
#######################################################
######### vcluster pro generate admin-kube-config ###########
#######################################################
Creates a new kube config that can be used to connect
a cluster to vCluster.Pro

Example:
vcluster pro generate admin-kube-config
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})
			c, err := loader.ClientConfig()
			if err != nil {
				return err
			}

			return cmd.Run(cobraCmd.Context(), c)
		},
	}

	c.Flags().StringVar(&cmd.Namespace, "namespace", "loft", "The namespace to generate the service account in. The namespace will be created if it does not exist")
	c.Flags().StringVar(&cmd.ServiceAccount, "service-account", "loft-admin", "The service account name to create")
	return c
}

// Run executes the command
func (cmd *GenerateKubeConfigCmd) Run(ctx context.Context, c *rest.Config) error {
	token, err := generate.GetAuthToken(ctx, c, cmd.Namespace, cmd.ServiceAccount)
	if err != nil {
		return fmt.Errorf("get auth token: %w", err)
	}

	// print kube config
	return kubeconfig.PrintTokenKubeConfig(c, string(token))
}
