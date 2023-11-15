package devpod

import (
	"os"

	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/devpod/list"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewDevPodCmd creates a new cobra command
func NewDevPodCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	c := &cobra.Command{
		Use:    "devpod",
		Hidden: true,
		Short:  "DevPod commands",
		Long: `
########################################################
##################### loft devpod ######################
########################################################
	`,
		PersistentPreRunE: func(cobraCmd *cobra.Command, args []string) error {
			if os.Getenv("DEVPOD_DEBUG") == "true" {
				log.Default.SetLevel(logrus.DebugLevel)
			}
			if globalFlags.Config == "" && os.Getenv("LOFT_CONFIG") != "" {
				globalFlags.Config = os.Getenv("LOFT_CONFIG")
			}

			log.Default.SetFormat(log.JSONFormat)
			return nil
		},
		Args: cobra.NoArgs,
	}

	c.AddCommand(list.NewListCmd(globalFlags))
	c.AddCommand(NewUpCmd(globalFlags))
	c.AddCommand(NewStopCmd(globalFlags))
	c.AddCommand(NewSshCmd(globalFlags))
	c.AddCommand(NewStatusCmd(globalFlags))
	c.AddCommand(NewDeleteCmd(globalFlags))
	return c
}
