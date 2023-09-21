package defaults

import (
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	pdefaults "github.com/loft-sh/loftctl/v3/pkg/defaults"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

type viewCmd struct {
	*flags.GlobalFlags

	Log      log.Logger
	Defaults *pdefaults.Defaults
}

// NewViewCmd creates a new command
func NewViewCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	cmd := &viewCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
		Defaults:    defaults,
	}

	description := `
#######################################################
################# loft defaults view ##################
#######################################################
View shows all default values configured for lofctl

Example:
loft defaults view
#######################################################
	`

	c := &cobra.Command{
		Use:   "view",
		Short: "View all defaults values",
		Long:  description,
		Args:  cobra.NoArgs,
		RunE:  func(cobraCmd *cobra.Command, args []string) error { return cmd.Run() },
	}

	return c
}

// Run executes the functionality
func (cmd *viewCmd) Run() error {
	cmd.Log.Infof("Showing all default values configured for loftctl:")
	for _, key := range pdefaults.DefaultKeys {
		value, err := cmd.Defaults.Get(key, "")
		if err != nil {
			continue
		}
		if value == "" {
			value = "<empty>"
		}
		cmd.Log.Infof("\t%s: %s", key, value)
	}

	return nil
}
