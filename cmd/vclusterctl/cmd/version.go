package cmd

import (
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of vcluster",
	Long:  `All software has versions. This is Vcluster's.`,
	Run: func(cmd *cobra.Command, _ []string) {
		root := cmd.Root()
		root.SetArgs([]string{"--version"})
		_ = root.Execute()
	},
}
