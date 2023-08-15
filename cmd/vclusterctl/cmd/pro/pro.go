package pro

import (
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/spf13/cobra"
)

func NewProCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	proCmd := &cobra.Command{
		Use:   "pro",
		Short: "vcluster.pro subcommands",
		Long: `
#######################################################
#################### vcluster get #####################
#######################################################
		`,
		Args: cobra.NoArgs,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Download loft cli here

			log.GetInstance().Info("Download loft cli")

			return nil
		},
	}

	proCmd.AddCommand(NewStartCmd(globalFlags))
	proCmd.AddCommand(NewStopCmd(globalFlags))
	proCmd.AddCommand(NewLoginCmd(globalFlags))
	proCmd.AddCommand(NewCreateCmd(globalFlags))
	proCmd.AddCommand(NewImportCmd(globalFlags))
	proCmd.AddCommand(NewDeleteCmd(globalFlags))
	proCmd.AddCommand(NewListCmd(globalFlags))

	return proCmd
}
