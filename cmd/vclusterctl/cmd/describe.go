package cmd

import (
	"cmp"
	"fmt"
	"os"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/constants"
	pdefaults "github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// DescribeCmd holds the describe cmd flags
type DescribeCmd struct {
	*flags.GlobalFlags
	cli.DescribeOptions

	log log.Logger
}

// NewDescribeCmd creates a new command
func NewDescribeCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	cmd := &DescribeCmd{
		GlobalFlags: globalFlags,
		log:         log.NewStdoutLogger(os.Stdin, os.Stderr, os.Stderr, logrus.InfoLevel),
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
	cobraCmd.Flags().StringVarP(&cmd.OutputFormat, "output", "o", "", "The format to use to display the information, can either be json or yaml")
	cobraCmd.Flags().StringVarP(&cmd.Project, "project", "p", p, "The project to use")

	cobraCmd.Flags().BoolVar(&cmd.AllValues, "all", true, "If set, return the complete vcluster.yaml, otherwise return only the user supplied vcluster.yaml")

	cobraCmd.Flags().BoolVar(&cmd.GenerateUserSuppliedConfigIfMissing, "generate-config", true, "Attempt to generate the user supplied config if missing.")
	cobraCmd.Flags().StringVar(&cmd.ChartName, "chart-name", "vcluster", "The virtual cluster chart name to use")
	cobraCmd.Flags().StringVar(&cmd.ChartRepo, "chart-repo", constants.LoftChartRepo, "The virtual cluster chart repo to use")

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
		return cli.DescribePlatform(cobraCmd.Context(), cmd.GlobalFlags, os.Stdout, cmd.log, name, cmd.Project, cmd.OutputFormat)
	}

	outputBytes, err := cli.DescribeHelm(cobraCmd.Context(), &cmd.DescribeOptions, cmd.GlobalFlags, name, cmd.log)
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(outputBytes)
	return err
}
