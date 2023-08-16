package pro

import (
	"fmt"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/spf13/cobra"
)

type ImportCmd struct{}

func NewImportCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := ImportCmd{}

	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Import an existing pro virtual cluster to the vcluster pro server",
		Long: `
#######################################################
#################### vcluster pro import #####################
#######################################################
Import an existing pro virtual cluster to the vcluster pro server
#######################################################
	`,
		DisableFlagParsing: true,
		RunE:               cmd.RunE,
	}

	return importCmd
}

func (ic *ImportCmd) RunE(cobraCmd *cobra.Command, args []string) error {
	ctx := cobraCmd.Context()

	cobraCmd.SilenceUsage = true

	log.GetInstance().Info("Importing pro virtual cluster ...")

	config, err := pro.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get vcluster pro config: %w", err)
	}

	// check if we have a version
	if config.LastUsedVersion == "" {
		return fmt.Errorf("no vcluster pro version found, please run 'vcluster pro login' first")
	}

	err = pro.RunLoftCli(ctx, config.LastUsedVersion, append([]string{"import", "vcluster"}, args...))
	if err != nil {
		return fmt.Errorf("failed to import vcluster pro: %w", err)
	}

	return nil
}
