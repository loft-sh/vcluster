package procli

import (
	"fmt"

	loftctlflags "github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
)

// GlobalFlags converts vcluster global flags to vcluster pro global flags
func GlobalFlags(globalFlags *flags.GlobalFlags) (*loftctlflags.GlobalFlags, error) {
	loftctlGlobalFlags := &loftctlflags.GlobalFlags{
		Silent:    globalFlags.Silent,
		Debug:     globalFlags.Debug,
		LogOutput: globalFlags.LogOutput,
	}

	if globalFlags.Config != "" {
		loftctlGlobalFlags.Config = globalFlags.Config
	} else {
		var err error
		loftctlGlobalFlags.Config, err = ConfigFilePath()
		if err != nil {
			return nil, fmt.Errorf("failed to get vcluster pro configuration file path: %w", err)
		}
	}

	return loftctlGlobalFlags, nil
}
