package token

import (
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"
)

func NewTokenCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	tokenCmd := &cobra.Command{
		Use:   "token",
		Short: "vCluster token subcommand",
		Long: `#######################################################
#################### vcluster token #####################
#######################################################
		`,
		Args: cobra.NoArgs,
	}

	tokenCmd.AddCommand(NewCreateCmd(globalFlags))
	tokenCmd.AddCommand(NewListCmd(globalFlags))
	tokenCmd.AddCommand(NewDeleteCmd(globalFlags))
	return tokenCmd
}
