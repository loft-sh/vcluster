package pro

import (
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/spf13/cobra"
)

type aliasCmd struct {
	added       map[string]bool
	commands    map[string]*cobra.Command
	globalFlags *flags.GlobalFlags
}

func NewAliasCmd(globalFlags *flags.GlobalFlags) aliasCmd {
	return aliasCmd{
		added:       map[string]bool{},
		commands:    map[string]*cobra.Command{},
		globalFlags: globalFlags,
	}
}

func (a *aliasCmd) AddCmd(use, description string) {
	uses := strings.Split(use, " ")

	for i, currentUse := range uses {
		currentCommand := strings.Join(uses[:i+1], " ")

		if _, ok := a.commands[currentCommand]; !ok {
			a.commands[currentCommand] = &cobra.Command{
				Use:                currentUse,
				DisableFlagParsing: true,
			}

			if i != 0 && !a.added[currentCommand] {
				previousCommand := strings.Join(uses[:i], " ")
				a.commands[previousCommand].AddCommand(a.commands[currentCommand])
				a.commands[previousCommand].RunE = nil
				a.added[currentCommand] = true
			}
		}
	}

	a.commands[use].Short = description
	a.commands[use].RunE = func(cmd *cobra.Command, args []string) error { return a.runE(cmd, uses, args) }
}

func (a *aliasCmd) Commands() []*cobra.Command {
	cmds := []*cobra.Command{}

	for key, cmd := range a.commands {
		if strings.Count(key, " ") == 0 {
			cmds = append(cmds, cmd)
		}
	}

	return cmds
}

func (aliasCmd) runE(cobraCmd *cobra.Command, split, args []string) error {
	ctx := cobraCmd.Context()

	cobraCmd.SilenceUsage = true

	// check if we have a version
	lastUsedVersion, err := pro.LastUsedVersion()
	if err != nil {
		return fmt.Errorf("failed to get last user cli version from config: %w", err)
	}

	err = pro.RunLoftCli(ctx, lastUsedVersion, append(split, args...))
	if err != nil {
		return fmt.Errorf("failed to create vcluster pro: %w", err)
	}

	return nil
}
