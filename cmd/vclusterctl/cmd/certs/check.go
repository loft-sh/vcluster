package certs

import (
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/certs"
	"github.com/loft-sh/vcluster/pkg/cli/completion"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/spf13/cobra"
)

type checkCmd struct {
	*flags.GlobalFlags

	Output string
	log    log.Logger
}

func check(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &checkCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	useLine, nameValidator := util.NamedPositionalArgsValidator(true, false, "VCLUSTER_NAME")
	checkCmd := &cobra.Command{
		Use:   "check" + useLine,
		Short: "Checks the current certificates",
		Long: `##############################################################
################### vcluster certs check #####################
##############################################################
Checks the current certificates.

Examples:
vcluster -n test certs check test
##############################################################
	`,
		Args:              nameValidator,
		ValidArgsFunction: completion.NewValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return certs.Check(cobraCmd.Context(), args[0], cmd.GlobalFlags, cmd.Output, cmd.log)
		}}

	checkCmd.Flags().StringVar(&cmd.Output, "output", "table", "Choose the format of the output. [table|json]")

	return checkCmd
}
