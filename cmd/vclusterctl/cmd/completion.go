package cmd

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/find"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const compDesc = `
#######################################################
################### vcluster completion ###############	
#######################################################
Generates completion scripts for various shells

Example:
vcluster completion bash
vcluster completion zsh 
vcluster completion fish
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

const fishCompDesc = `
#######################################################
################### vcluster completion fish ###########
#######################################################
Generate the autocompletion script for fish

Example:
vcluster completion fish > "${fpath[1]}/_vcluster"
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
	completionCmd.AddCommand(NewBashCommand(), NewZshCommand(), NewFishCommand())
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

func NewFishCommand() *cobra.Command {
	fishCmd := &cobra.Command{
		Use:                   "fish",
		Short:                 "generate autocompletion script for fish",
		Long:                  fishCompDesc,
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenFishCompletion(os.Stdout, false)
		},
	}
	return fishCmd
}

// listVClusterForCompletion fetches the list of all the available vclusters
// for command autocompletion
func listVClusterForCompletion(cmd *flags.GlobalFlags) ([]string, cobra.ShellCompDirective) {
	if cmd.Context == "" {
		rawConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).RawConfig()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		cmd.Context = rawConfig.CurrentContext
	}

	namespace := metav1.NamespaceAll
	if cmd.Namespace != "" {
		namespace = cmd.Namespace
	}

	vClusters, err := find.ListVClusters(cmd.Context, "", namespace)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	listClusters := []string{}
	for _, v := range vClusters {
		listClusters = append(listClusters, v.Name)
	}
	return listClusters, cobra.ShellCompDirectiveNoFileComp
}

// registerNamespaceCompletionFunc registers namespace autocompletion function.
// which fetches and display all the namespaces based on current context.
func registerNamespaceCompletionFunc(rootCmd *cobra.Command) {
	_ = rootCmd.RegisterFlagCompletionFunc("namespace", listNamespacesForCompletion)
}

// listNamespacesForCompletion fetches all namespaces based on current context.
// this function will be called automatically when user press <tab><tab>
func listNamespacesForCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	kubeClientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})
	restConfig, err := kubeClientConfig.ClientConfig()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	namespaces, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	namespacesStr := []string{}
	for _, n := range namespaces.Items {
		namespacesStr = append(namespacesStr, n.Name)
	}
	return namespacesStr, cobra.ShellCompDirectiveNoFileComp
}
