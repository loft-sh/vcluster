package space

import (
	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/defaults"
	"github.com/spf13/cobra"
)

func NewRootCmd(globalFlags *flags.GlobalFlags, defaults *defaults.Defaults) *cobra.Command {
	short := "Management operations on space resources"

	spaceCmd := &cobra.Command{
		Use:   "space",
		Short: short,
		Long:  product.ReplaceWithHeader("space", short),

		Aliases: []string{"spaces"},

		Example: product.Replace(`  # List all spaces
  vcluster pro spaces list

  # Create a new space
  vcluster pro spaces create myspace

  # Delete a space
  vcluster pro spaces delete myspace

  # Use a space
  vcluster pro space use myspace
`),
		Args: cobra.NoArgs,
	}
	spaceCmd.AddCommand(NewSpaceListCmd(globalFlags, defaults))
	spaceCmd.AddCommand(NewSpaceCreateCmd(globalFlags, defaults))
	spaceCmd.AddCommand(NewSpaceDeleteCmd(globalFlags, defaults))
	spaceCmd.AddCommand(NewSpaceUseCmd(globalFlags, defaults))

	return spaceCmd
}
