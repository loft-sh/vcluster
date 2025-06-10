package add

import (
	"context"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"
)

type VClusterCmd struct {
	*flags.GlobalFlags
	cli.AddVClusterOptions

	Log log.Logger
}

func NewVClusterCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &VClusterCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	description := `###############################################
############# vcluster platform add vcluster ##############
###############################################
Adds a vCluster to the vCluster platform.

Example:
vcluster platform add vcluster my-vcluster --namespace vcluster-my-vcluster --project my-project --import-name my-vcluster

Add all vCluster instances in the host cluster:
vcluster platform add vcluster --project my-project --all

###############################################
	`

	addCmd := &cobra.Command{
		Use:   "vcluster",
		Short: "Adds an existing vCluster to the vCluster platform",
		Long:  description,
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	addCmd.Flags().StringVar(&cmd.Project, "project", "", "The project to import the vCluster into")
	addCmd.Flags().StringVar(&cmd.ImportName, "import-name", "", "The name of the vCluster under projects. If unspecified, will use the vcluster name")
	addCmd.Flags().BoolVar(&cmd.Restart, "restart", true, "Restart the vCluster control-plane after creating the platform secret")
	addCmd.Flags().StringVar(&cmd.AccessKey, "access-key", "", "The access key for the vCluster to connect to the platform. If empty, the CLI will generate one")
	addCmd.Flags().StringVar(&cmd.Host, "host", "", "The host where to reach the platform")
	addCmd.Flags().BoolVar(&cmd.Insecure, "insecure", false, "If the platform host is insecure")
	addCmd.Flags().BytesBase64Var(&cmd.CertificateAuthorityData, "ca-data", []byte{}, "additional, base64 encoded certificate authority data that will be passed to the platform secret")
	addCmd.Flags().BoolVar(&cmd.All, "all", false, "all will try to add Virtual Cluster found in all namespaces in the host cluster. If this flag is set, any provided vCluster name argument is ignored")

	return addCmd
}

// Run executes the functionality
func (cmd *VClusterCmd) Run(ctx context.Context, args []string) error {
	return cli.AddVClusterHelm(ctx, &cmd.AddVClusterOptions, cmd.GlobalFlags, args, cmd.Log)
}
