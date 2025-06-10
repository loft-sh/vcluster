package cmd

import (
	"cmp"
	"fmt"
	"os"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	pdefaults "github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/spf13/cobra"
)

// DescribeCmd holds the describe cmd flags
type DescribeCmd struct {
	*flags.GlobalFlags

	output  string
	log     log.Logger
	project string
}

// NewDescribeCmd creates a new command
func NewDescribeCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	cmd := &DescribeCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}
	driver := ""

	cobraCmd := &cobra.Command{
		Use:   "describe",
		Short: "Describes a virtual cluster",
		Long: `#######################################################
################## vcluster describe ##################
#######################################################
describes a virtual cluster

Example:
vcluster describe test
vcluster describe -o json test
#######################################################
	`,
		Args: cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd, driver, args[0])
		},
	}
	p, _ := defaults.Get(pdefaults.KeyProject, "")

	cobraCmd.Flags().StringVar(&driver, "driver", "", "The driver to use for managing the virtual cluster, can be either helm or platform.")
	cobraCmd.Flags().StringVarP(&cmd.output, "output", "o", "", "The format to use to display the information, can either be json or yaml")
	cobraCmd.Flags().StringVarP(&cmd.project, "project", "p", p, "The project to use")

	return cobraCmd
}

// Run executes the functionality
func (cmd *DescribeCmd) Run(cobraCmd *cobra.Command, driver, name string) error {
	cfg := cmd.LoadedConfig(cmd.log)

	// If driver has been passed as flag use it, otherwise read it from the config file
	driverType, err := config.ParseDriverType(cmp.Or(driver, string(cfg.Driver.Type)))
	if err != nil {
		return fmt.Errorf("parse driver type: %w", err)
	}
	if driverType == config.PlatformDriver {
		return cli.DescribePlatform(cobraCmd.Context(), cmd.GlobalFlags, os.Stdout, cmd.log, name, cmd.project, cmd.output)
	}

	return cli.DescribeHelm(cobraCmd.Context(), cmd.GlobalFlags, os.Stdout, name, cmd.output)
}
