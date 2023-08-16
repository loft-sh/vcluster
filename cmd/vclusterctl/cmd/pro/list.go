package pro

import (
	"fmt"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/spf13/cobra"
)

type ListCmd struct{}

func NewListCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := ListCmd{}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all pro virtual clusters",
		Long: `
#######################################################
#################### vcluster pro list #####################
#######################################################
List all pro virtual cluster

Example:
vcluster list
vcluster list --output json
vcluster list --namespace test
#######################################################
	`,
		DisableFlagParsing: true,
		RunE:               cmd.RunE,
	}

	return listCmd
}

func (lc *ListCmd) RunE(cobraCmd *cobra.Command, args []string) error {
	ctx := cobraCmd.Context()

	cobraCmd.SilenceUsage = true

	log.GetInstance().Info("Listing pro virtual clusters ...")

	config, err := pro.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get vcluster pro config: %w", err)
	}

	// check if we have a version
	if config.LastUsedVersion == "" {
		return fmt.Errorf("no vcluster pro version found, please run 'vcluster pro login' first")
	}

	err = pro.RunLoftCli(ctx, config.LastUsedVersion, append([]string{"list", "vclusters"}, args...))
	if err != nil {
		return fmt.Errorf("failed to list vcluster pro: %w", err)
	}

	return nil
}
