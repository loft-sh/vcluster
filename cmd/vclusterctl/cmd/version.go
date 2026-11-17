package cmd

import (
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the vCluster version",
	Run: func(cmd *cobra.Command, _ []string) {
		root := cmd.Root()
		root.SetArgs([]string{"--version"})
		_ = root.Execute()
	},
}
