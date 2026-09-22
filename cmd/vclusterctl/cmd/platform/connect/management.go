package connect

import (
	"os"

	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/kubeconfig"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
)

// ManagementCmd holds the cmd flags
type ManagementCmd struct {
	*flags.GlobalFlags
	log log.Logger

	Print bool
}

// NewManagementCmd creates a new command
func newManagementCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ManagementCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	description := product.ReplaceWithHeader("connect management", `
Creates a new kube context to the vCluster platform Management API.

Example:
vcluster platform connect management
########################################################
	`)
	c := &cobra.Command{
		Use:   "management",
		Short: product.Replace("Creates a kube context to the vCluster platform Management API"),
		Long:  description,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			// Check for newer version
			if !cmd.Print {
				upgrade.PrintNewerVersionWarning()
			}

			return cmd.run(cobraCmd)
		},
	}

	c.Flags().BoolVar(&cmd.Print, "print", false, "When enabled prints the context to stdout")
	return c
}

func (cmd *ManagementCmd) run(cobraCmd *cobra.Command) error {
	cfg := cmd.LoadedConfig(cmd.log)
	platformClient, err := platform.InitClientFromConfig(cobraCmd.Context(), cfg)
	if err != nil {
		return err
	}

	// create kube context options
	contextOptions, err := createManagementContextOptions(platformClient, cmd.Config, true)
	if err != nil {
		return err
	}

	// check if we should print or update the config
	if cmd.Print {
		err = kubeconfig.PrintKubeConfigTo(contextOptions, os.Stdout)
		if err != nil {
			return err
		}
	} else {
		// update kube config
		err = kubeconfig.UpdateKubeConfig(contextOptions, cfg)
		if err != nil {
			return err
		}

		cmd.log.Donef("Successfully updated kube context to use cluster %s", ansi.Color(contextOptions.Name, "white+b"))
	}

	return nil
}

func createManagementContextOptions(platformClient platform.Client, config string, setActive bool) (kubeconfig.ContextOptions, error) {
	contextOptions := kubeconfig.ContextOptions{
		Name:       kubeconfig.ManagementContextName(),
		ConfigPath: config,
		SetActive:  setActive,
	}

	contextOptions.Server = platformClient.Config().Platform.Host + "/kubernetes/management"
	contextOptions.InsecureSkipTLSVerify = platformClient.Config().Platform.Insecure

	return contextOptions, nil
}
