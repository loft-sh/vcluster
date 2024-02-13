package cmd

import (
	"context"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/find"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const completionTimeout = time.Second * 3

// defining as a type purely for readability purposes.
// this is the type accepted by cobra.Command.ValidArgsFunc and cobra.Command.RegisterFlagCompletionFunc
type completionFunc func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective)

type completionResult struct {
	completions []string
	directive   cobra.ShellCompDirective
}

// wrapper to add a timeout to completionFuncs
func wrapCompletionFuncWithTimeout(defaultDirective cobra.ShellCompDirective, compFunc completionFunc) completionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// initialize a buffered channel to receive results
		resultChan := make(chan completionResult, 1)

		// run completion function in the background and send result to resultChan
		go func(c chan completionResult) {
			completions, directive := compFunc(cmd, args, toComplete)
			r := completionResult{completions: completions, directive: directive}
			c <- r
		}(resultChan)

		// wait for results or timeout
		select {
		case result := <-resultChan:
			return result.completions, result.directive
		case <-time.After(completionTimeout):
			return []string{}, defaultDirective | cobra.ShellCompDirectiveError
		}
	}
}

// newValidVClusterNameFunc returns a function that handles shell completion when the argument is vcluster_name
// It takes into account the namespace if specified by the --namespace flag.
func newValidVClusterNameFunc(globalFlags *flags.GlobalFlags) completionFunc {
	fn := func(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		vclusters, _, err := find.ListVClusters(cmd.Context(), nil, globalFlags.Context, "", globalFlags.Namespace, "", log.Default.ErrorStreamOnly())
		if err != nil {
			return []string{}, cobra.ShellCompDirectiveError | cobra.ShellCompDirectiveNoFileComp
		}
		names := make([]string, len(vclusters))
		for i := range vclusters {
			names[i] = vclusters[i].Name
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
	return wrapCompletionFuncWithTimeout(cobra.ShellCompDirectiveNoFileComp, fn)
}

// newNamespaceCompletionFunc handles shell completions for the namespace flag
func newNamespaceCompletionFunc(ctx context.Context) completionFunc {
	fn := func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		restConfig, err := config.GetConfig()
		if err != nil {
			return []string{}, cobra.ShellCompDirectiveError | cobra.ShellCompDirectiveNoFileComp
		}
		kubeClient, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return []string{}, cobra.ShellCompDirectiveError | cobra.ShellCompDirectiveNoFileComp
		}

		namespaces, err := kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		if err != nil {
			return []string{}, cobra.ShellCompDirectiveError | cobra.ShellCompDirectiveNoFileComp
		}
		names := make([]string, len(namespaces.Items))
		for i := range namespaces.Items {
			ns := namespaces.Items[i].Name
			if ns != metav1.NamespaceSystem {
				names[i] = ns
			}
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
	return wrapCompletionFuncWithTimeout(cobra.ShellCompDirectiveNoFileComp, fn)
}
