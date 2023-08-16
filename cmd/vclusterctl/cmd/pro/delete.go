package pro

import (
	"fmt"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/spf13/cobra"
)

type DeleteCmd struct{}

func NewDeleteCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := DeleteCmd{}

	deleteCmd := &cobra.Command{
		Use:   "delete [flags] vcluster_name",
		Short: "Delete a pro virtual cluster",
		Long: `
#######################################################
#################### vcluster pro delete #####################
#######################################################
Deletes a pro virtual cluster

Example:
vcluster pro delete test --namespace test
#######################################################
	`,
		DisableFlagParsing: true,
		RunE:               cmd.RunE,
	}

	return deleteCmd
}

func (dc *DeleteCmd) RunE(cobraCmd *cobra.Command, args []string) error {
	ctx := cobraCmd.Context()

	cobraCmd.SilenceUsage = true

	log.GetInstance().Info("Deleting pro virtual cluster ...")

	config, err := pro.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get vcluster pro config: %w", err)
	}

	// check if we have a version
	if config.LastUsedVersion == "" {
		return fmt.Errorf("no vcluster pro version found, please run 'vcluster pro login' first")
	}

	err = pro.RunLoftCli(ctx, config.LastUsedVersion, append([]string{"delete", "vcluster"}, args...))
	if err != nil {
		return fmt.Errorf("failed to delete vcluster pro: %w", err)
	}

	return nil
}
