package cmd

import (
	"fmt"

	loftctlImport "github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/importcmd"
	loftctlUtil "github.com/loft-sh/loftctl/v3/pkg/util"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/spf13/cobra"
)

func NewImportCmd(globalFlags *flags.GlobalFlags) (*cobra.Command, error) {
	loftctlGlobalFlags, err := pro.GlobalFlags(globalFlags)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pro flags: %w", err)
	}

	cmd := &loftctlImport.VClusterCmd{
		GlobalFlags: loftctlGlobalFlags,
		Log:         log.GetInstance(),
	}

	description := `########################################################
################### vcluster import ####################
########################################################
Imports a vcluster into a vCluster.Pro project.

Example:
vcluster import my-vcluster --cluster connected-cluster \
--namespace vcluster-my-vcluster --project my-project --importname my-vcluster
#######################################################
	`

	importCmd := &cobra.Command{
		Use:   "import" + loftctlUtil.VClusterNameOnlyUseLine,
		Short: "Imports a vcluster into a vCluster.Pro project",
		Long:  description,
		Args:  loftctlUtil.VClusterNameOnlyValidator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	importCmd.Flags().StringVar(&cmd.VClusterClusterName, "cluster", "", "Cluster name of the cluster the virtual cluster is running on")
	importCmd.Flags().StringVar(&cmd.VClusterNamespace, "namespace", "", "The namespace of the vcluster")
	importCmd.Flags().StringVar(&cmd.Project, "project", "", "The project to import the vcluster into")
	importCmd.Flags().StringVar(&cmd.ImportName, "importname", "", "The name of the vcluster under projects. If unspecified, will use the vcluster name")

	_ = importCmd.MarkFlagRequired("cluster")
	_ = importCmd.MarkFlagRequired("namespace")
	_ = importCmd.MarkFlagRequired("project")

	return importCmd, nil
}
