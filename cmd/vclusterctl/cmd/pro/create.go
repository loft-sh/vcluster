package pro

import (
	"fmt"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/spf13/cobra"
)

type CreateCmd struct{}

func NewCreateCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := CreateCmd{}

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new pro virtual cluster",
		Long: `
#######################################################
#################### vcluster pro create #####################
#######################################################
Creates a new pro virtual cluster

Example:
vcluster pro create test --namespace test
#######################################################
	`,
		DisableFlagParsing: true,
		RunE:               cmd.RunE,
	}

	return createCmd
}

func (cc *CreateCmd) RunE(cobraCmd *cobra.Command, args []string) error {
	ctx := cobraCmd.Context()

	cobraCmd.SilenceUsage = true

	log.GetInstance().Info("Creating a new vcluster pro...")

	config, err := pro.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get vcluster pro config: %w", err)
	}

	// check if we have a version
	if config.LastUsedVersion == "" {
		return fmt.Errorf("no vcluster pro version found, please run 'vcluster pro login' first")
	}

	err = pro.RunLoftCli(ctx, config.LastUsedVersion, append([]string{"create", "vcluster"}, args...))
	if err != nil {
		return fmt.Errorf("failed to create vcluster pro: %w", err)
	}

	return nil
}
