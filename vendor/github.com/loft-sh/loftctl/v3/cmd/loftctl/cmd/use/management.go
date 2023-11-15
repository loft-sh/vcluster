package use

import (
	"os"

	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/loftctl/v3/pkg/kubeconfig"
	"github.com/loft-sh/loftctl/v3/pkg/upgrade"
	"github.com/loft-sh/log"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
)

// ManagementCmd holds the cmd flags
type ManagementCmd struct {
	*flags.GlobalFlags

	Print bool

	log log.Logger
}

// NewManagementCmd creates a new command
func NewManagementCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ManagementCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	description := product.ReplaceWithHeader("use management", `
Creates a new kube context to the Loft Management API.

Example:
loft use management
########################################################
	`)
	if upgrade.IsPlugin == "true" {
		description = `
########################################################
################ devspace use management ###############
########################################################
Creates a new kube context to the Loft Management API.

Example:
devspace use management
########################################################
	`
	}
	c := &cobra.Command{
		Use:   "management",
		Short: product.Replace("Creates a kube context to the Loft Management API"),
		Long:  description,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Check for newer version
			if !cmd.Print {
				upgrade.PrintNewerVersionWarning()
			}

			return cmd.Run(args)
		},
	}

	c.Flags().BoolVar(&cmd.Print, "print", false, "When enabled prints the context to stdout")
	return c
}

func (cmd *ManagementCmd) Run(args []string) error {
	baseClient, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return err
	}

	// create kube context options
	contextOptions, err := CreateManagementContextOptions(baseClient, cmd.Config, true, cmd.log)
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
		err = kubeconfig.UpdateKubeConfig(contextOptions)
		if err != nil {
			return err
		}

		cmd.log.Donef("Successfully updated kube context to use cluster %s", ansi.Color(contextOptions.Name, "white+b"))
	}

	return nil
}

func CreateManagementContextOptions(baseClient client.Client, config string, setActive bool, log log.Logger) (kubeconfig.ContextOptions, error) {
	contextOptions := kubeconfig.ContextOptions{
		Name:       kubeconfig.ManagementContextName(),
		ConfigPath: config,
		SetActive:  setActive,
	}

	contextOptions.Server = baseClient.Config().Host + "/kubernetes/management"
	contextOptions.InsecureSkipTLSVerify = baseClient.Config().Insecure

	return contextOptions, nil
}
