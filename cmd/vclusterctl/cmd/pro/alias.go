package pro

import (
	"fmt"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type AliasCmd struct{}

var (
	aliasCmd = AliasCmd{}
)

func NewAliasCmd(globalFlags *flags.GlobalFlags, use string) *cobra.Command {
	cmd := &cobra.Command{
		Use:                use,
		Short:              fmt.Sprintf("%s a new pro virtual cluster", cases.Title(language.English).String(use)),
		DisableFlagParsing: true,
		RunE:               aliasCmd.RunE,
	}

	return cmd
}

func (AliasCmd) RunE(cobraCmd *cobra.Command, args []string) error {
	ctx := cobraCmd.Context()

	cobraCmd.SilenceUsage = true

	calledAs := cobraCmd.CalledAs()

	// check if we have a version
	lastUsedVersion, err := pro.LastUsedVersion()
	if err != nil {
		return fmt.Errorf("failed to get last user cli version from config: %w", err)
	}

	err = pro.RunLoftCli(ctx, lastUsedVersion, append([]string{calledAs}, args...))
	if err != nil {
		return fmt.Errorf("failed to create vcluster pro: %w", err)
	}

	return nil
}
