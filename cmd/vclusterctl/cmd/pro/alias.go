package pro

import (
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type AliasCmd struct{}

var (
	aliasCmd = AliasCmd{}

	cmdMap = map[string]*cobra.Command{}
)

func GetRootCmds() []*cobra.Command {
	cmds := []*cobra.Command{}

	for key, cmd := range cmdMap {
		if strings.Count(key, " ") == 0 {
			cmds = append(cmds, cmd)
		}
	}

	return cmds
}

func AddAliasCmd(globalFlags *flags.GlobalFlags, use string) {
	split := strings.Split(use, " ")

	for i, currentUse := range split {
		concattedPrev := strings.Join(split[:i], " ")
		concatted := strings.Join(split[:i+1], " ")

		if _, ok := cmdMap[concatted]; !ok {

			cmdMap[concatted] = &cobra.Command{
				Use:                currentUse,
				Short:              cases.Title(language.English).String(currentUse),
				DisableFlagParsing: true,
			}

			if i != 0 {
				cmdMap[concattedPrev].AddCommand(cmdMap[concatted])
			}
		}
	}

	cmdMap[use].RunE = func(cmd *cobra.Command, args []string) error { return aliasCmd.RunE(cmd, split) }

}

func (AliasCmd) RunE(cobraCmd *cobra.Command, args []string) error {
	ctx := cobraCmd.Context()

	cobraCmd.SilenceUsage = true

	// check if we have a version
	lastUsedVersion, err := pro.LastUsedVersion()
	if err != nil {
		return fmt.Errorf("failed to get last user cli version from config: %w", err)
	}

	err = pro.RunLoftCli(ctx, lastUsedVersion, args)
	if err != nil {
		return fmt.Errorf("failed to create vcluster pro: %w", err)
	}

	return nil
}
