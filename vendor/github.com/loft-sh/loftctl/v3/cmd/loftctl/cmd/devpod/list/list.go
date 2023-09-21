package list

import (
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/spf13/cobra"
)

// NewListCmd creates a new cobra command
func NewListCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	c := &cobra.Command{
		Use:    "list",
		Hidden: true,
		Short:  "DevPod List commands",
		Long: `
#######################################################
################### loft devpod list ##################
#######################################################
	`,
		Args: cobra.NoArgs,
	}

	c.AddCommand(NewProjectsCmd(globalFlags))
	c.AddCommand(NewTemplatesCmd(globalFlags))
	c.AddCommand(NewTemplateOptionsCmd(globalFlags))
	c.AddCommand(NewTemplateOptionsVersionCmd(globalFlags))
	return c
}
