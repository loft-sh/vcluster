package cmd

import (
	"os"
	"runtime"
	"text/template"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/telemetry"
	"github.com/spf13/cobra"
)

const cliInfoTemplate = `CLI Info:
- Version: {{ .Version }}
- OS: {{ .OS }}
- Arch: {{ .Arch }}
- Machine ID: {{ .MachineID }}
{{- if .InstanceID }}
- Instance ID: {{ .InstanceID }}
{{ end }}`

type cliInfo struct {
	Version    string
	OS         string
	Arch       string
	MachineID  string
	InstanceID string
}

// NewCreateCmd creates a new command
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
		Args: cobra.NoArgs,
		Run: func(cobraCmd *cobra.Command, args []string) {
			cliInfo := cliInfo{
				Version:   cobraCmd.Root().Version,
				OS:        runtime.GOOS,
				Arch:      runtime.GOARCH,
				MachineID: telemetry.GetMachineID(log.GetInstance()),
			}
			proClient, err := pro.CreateProClient()
			if err == nil {
				cliInfo.InstanceID = proClient.Self().Status.InstanceID
			}
			tmpl := template.Must(template.New("info").Parse(string(cliInfoTemplate)))
			_ = tmpl.Execute(os.Stdout, cliInfo)
		},
	}

	return cobraCmd
}
