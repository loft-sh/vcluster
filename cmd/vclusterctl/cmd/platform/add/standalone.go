package add

import (
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/standalone"
	"github.com/spf13/cobra"
)

func NewStandaloneCmd() *cobra.Command {
	options := &standalone.AddToPlatformOptions{}

	description := `################################################
####### vcluster platform add standalone #######
################################################
Adds a vCluster Standalone cluster to the vCluster platform.

Example:
vcluster platform add standalone my-cluster --project my-project --access-key my-access-key --host https://my-vcluster-platform.com

################################################
`

	addCmd := &cobra.Command{
		Use:   "standalone",
		Short: "Adds an existing vCluster Standalone cluster to the vCluster platform",
		Long:  description,
		Args:  cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			options.InstanceName = args[0]
			return standalone.AddToPlatform(cobraCmd.Context(), log.GetInstance(), options)
		},
	}

	addCmd.Flags().StringVar(&options.ProjectName, "project", "", "The project to import the vCluster into")
	addCmd.Flags().StringVar(&options.AccessKey, "access-key", "", "The access key for the vCluster to connect to the platform. If empty, the CLI will generate one")
	addCmd.Flags().StringVar(&options.Host, "host", "", "The host where to reach the platform")
	addCmd.Flags().BoolVar(&options.Insecure, "insecure", false, "If the platform host is insecure")

	return addCmd
}
