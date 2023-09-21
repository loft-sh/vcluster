package defaults

import (
	"fmt"
	"strings"

	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	pdefaults "github.com/loft-sh/loftctl/v3/pkg/defaults"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

type setCmd struct {
	*flags.GlobalFlags

	Log      log.Logger
	Defaults *pdefaults.Defaults
}

// NewSetCmd creates a new command
func NewSetCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	cmd := &setCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
		Defaults:    defaults,
	}

	description := fmt.Sprintf(`
#######################################################
################## loft defaults set ##################
#######################################################
Set sets a default value for lofctl
loft defaults set $KEY $VALUE

Example:
loft defaults set project your-project

Supported keys include:
%s
#######################################################
	`, strings.Join(pdefaults.DefaultKeys, "\n"))

	c := &cobra.Command{
		Use:   "set",
		Short: "Set default value",
		Long:  description,
		Args: func(cobraCmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(2)(cobraCmd, args); err != nil {
				return err
			}
			if !pdefaults.IsSupportedKey(args[0]) {
				return fmt.Errorf("unknown key %s, supported keys are: \n\t%s", args[0], strings.Join(pdefaults.DefaultKeys, "\n\t"))
			}

			return nil
		},
		RunE: func(cobraCmd *cobra.Command, args []string) error { return cmd.Run(args) },
	}

	return c
}

// Run executes the functionality
func (cmd *setCmd) Run(args []string) error {
	cmd.Log.Infof("Setting default value for \"%s\"", args[0])
	key := args[0]
	value := args[1]

	if err := cmd.Defaults.Set(key, value); err != nil {
		return err
	} else {
		if value == "" {
			value = "<empty>"
		}
		cmd.Log.Infof("%s: %s", key, value)
	}

	return nil
}
