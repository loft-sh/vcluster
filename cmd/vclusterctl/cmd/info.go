package cmd

import (
	"os"
	"runtime"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/procli"
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
func NewInfoCmd() *cobra.Command {
	cobraCmd := &cobra.Command{
		Use:   "info",
		Short: "Displays informations about the cli and platform",
		Long: `
#######################################################
################### vcluster info ###################
#######################################################
Displays information about vCluster

Example:
vcluster info
#######################################################
	`,
		Args:   cobra.NoArgs,
		Hidden: true,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			infos := cliInfo{
				Version:   cobraCmd.Root().Version,
				OS:        runtime.GOOS,
				Arch:      runtime.GOARCH,
				MachineID: telemetry.GetMachineID(log.GetInstance()),
			}
			proClient, err := procli.CreateProClient()
			if err == nil {
				infos.InstanceID = proClient.Self().Status.InstanceID
			}
			return yaml.NewEncoder(os.Stdout).Encode(struct{ Info cliInfo }{infos})
		},
	}

	return cobraCmd
}
