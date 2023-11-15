package vars

import (
	"errors"
	"os"
	"strings"

	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	ErrNotLoftContext = errors.New("current context is not a loft context, but predefined var LOFT_CLUSTER is used")
)

type clusterCmd struct {
	*flags.GlobalFlags
}

func newClusterCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &clusterCmd{
		GlobalFlags: globalFlags,
	}

	return &cobra.Command{
		Use:   "cluster",
		Short: "Prints the current cluster",
		Args:  cobra.NoArgs,
		RunE:  cmd.Run,
	}
}

// Run executes the command logic
func (*clusterCmd) Run(cobraCmd *cobra.Command, args []string) error {
	kubeConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).RawConfig()
	if err != nil {
		return err
	}

	kubeContext := os.Getenv("DEVSPACE_PLUGIN_KUBE_CONTEXT_FLAG")
	if kubeContext == "" {
		kubeContext = kubeConfig.CurrentContext
	}

	cluster, ok := kubeConfig.Clusters[kubeContext]
	if !ok {
		return ErrNotLoftContext
	}

	server := strings.TrimSuffix(cluster.Server, "/")
	splitted := strings.Split(server, "/")
	if len(splitted) < 3 {
		return ErrNotLoftContext
	} else if splitted[len(splitted)-2] != "cluster" || splitted[len(splitted)-3] != "kubernetes" {
		return ErrNotLoftContext
	}

	_, err = os.Stdout.Write([]byte(splitted[len(splitted)-1]))
	return err
}
