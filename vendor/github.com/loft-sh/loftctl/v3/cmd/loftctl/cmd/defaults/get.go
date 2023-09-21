package defaults

import (
	"fmt"
	"strings"

	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	pdefaults "github.com/loft-sh/loftctl/v3/pkg/defaults"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// GetCmd holds the cmd flags
type getCmd struct {
	*flags.GlobalFlags

	Log      log.Logger
	Defaults *pdefaults.Defaults
}

// NewGetCmd creates a new command
func NewGetCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	cmd := &getCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
		Defaults:    defaults,
	}

	description := fmt.Sprintf(`
#######################################################
################## loft defaults get ##################
#######################################################
Get retrieves a default value configured for lofctl
loft defaults get $KEY

Example:
loft defaults get project

Supported keys include:
%s
#######################################################
	`, strings.Join(pdefaults.DefaultKeys, "\n"))

	c := &cobra.Command{
		Use:   "get",
		Short: "Retrieve default value",
		Long:  description,
		Args: func(cobraCmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cobraCmd, args); err != nil {
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
func (cmd *getCmd) Run(args []string) error {
	cmd.Log.Infof("Retrieving default value for \"%s\"", args[0])
	key := args[0]

	if value, err := cmd.Defaults.Get(key, ""); err != nil {
		return err
	} else {
		if value == "" {
			value = "<empty>"
		}
		cmd.Log.Infof("%s: %s", key, value)
	}

	return nil
}
