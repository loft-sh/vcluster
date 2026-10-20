package cmd

import (
	"os"
	"runtime"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/telemetry"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type cliInfo struct {
	Version    string `yaml:"version,omitempty"`
	OS         string `yaml:"os,omitempty"`
	Arch       string `yaml:"arch,omitempty"`
	MachineID  string `yaml:"machineID,omitempty"`
	InstanceID string `yaml:"instanceID,omitempty"`
}

// NewInfoCmd creates a new info command
func NewInfoCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cobraCmd := &cobra.Command{
		Use:   "info",
		Short: "Displays informations about the cli and platform",
		Long: `#######################################################
################### vcluster info #####################
#######################################################
Displays information about vCluster

Example:
vcluster info
#######################################################
	`,
		Args:   cobra.NoArgs,
		Hidden: true,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			cfg := globalFlags.LoadedConfig(log.GetInstance())
			infos := cliInfo{
				Version:   cobraCmd.Root().Version,
				OS:        runtime.GOOS,
				Arch:      runtime.GOARCH,
				MachineID: telemetry.GetMachineID(cfg),
			}

			platformClient, err := platform.InitClientFromConfig(cobraCmd.Context(), cfg)
			if err == nil {
				infos.InstanceID = platformClient.Self().Status.InstanceID
			}

			return yaml.NewEncoder(os.Stdout).Encode(struct{ Info cliInfo }{infos})
		},
	}

	return cobraCmd
}
