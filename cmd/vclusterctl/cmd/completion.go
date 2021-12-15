package cmd

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
)

const compDesc = `
#######################################################
################### vcluster completion ###############	
#######################################################
Generates completion scripts for various shells

Example:
vcluster completion bash
vcluster completion zsh 
#######################################################
`

const bashCompDesc = `
#######################################################
################### vcluster completion bash ##########
#######################################################
Generate the autocompletion script for bash

Example:
- Linux:
vcluster completion bash > /etc/bash_completion.d/vcluster
- MacOS:
vcluster completion bash > \
/usr/local/etc/bash_completion.d/vcluster
#######################################################
`

const zshCompDesc = `
#######################################################
################### vcluster completion zsh ###########
#######################################################
Generate the autocompletion script for zsh

Example:
vcluster completion zsh > "${fpath[1]}/_vcluster"
#######################################################
`

func NewCompletionCmd() *cobra.Command {
	completionCmd := &cobra.Command{
		Use:                   "completion",
		Short:                 "Generates completion scripts for various shells",
		Long:                  compDesc,
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := cmd.Help()
			if err != nil {
				return err
			}
			return errors.New("subcommand is required")
		},
	}
	completionCmd.AddCommand(NewBashCommand(), NewZshCommand())
	return completionCmd
}

func NewBashCommand() *cobra.Command {
	bashCmd := &cobra.Command{
		Use:                   "bash",
		Short:                 "generate autocompletion script for bash",
		Long:                  bashCompDesc,
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenBashCompletion(os.Stdout)
		},
	}
	return bashCmd
}

func NewZshCommand() *cobra.Command {
	zshCmd := &cobra.Command{
		Use:                   "zsh",
		Short:                 "generate autocompletion script for zsh",
		Long:                  zshCompDesc,
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenZshCompletion(os.Stdout)
		},
	}
	return zshCmd
}
