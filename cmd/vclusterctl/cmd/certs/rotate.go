package certs

import (
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/certs"
	"github.com/loft-sh/vcluster/pkg/cli/completion"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/spf13/cobra"
)

type rotateCmd struct {
	*flags.GlobalFlags
	log log.Logger
}

func rotate(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &rotateCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	useLine, nameValidator := util.NamedPositionalArgsValidator(true, false, "VCLUSTER_NAME")
	rotateCmd := &cobra.Command{
		Use:   "rotate" + useLine,
		Short: "Rotates control-plane client and server certs",
		Long: `##############################################################
################### vcluster certs rotate ####################
##############################################################
Rotates the control-plane client and server leaf certificates
of the given virtual cluster.

Examples:
vcluster -n test certs rotate test
##############################################################
	`,
		Args:              nameValidator,
		ValidArgsFunction: completion.NewValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return certs.Rotate(cobraCmd.Context(), args[0], certs.RotationCmdCerts, cmd.GlobalFlags, cmd.log)
		}}

	return rotateCmd
}

type rotateCACmd struct {
	*flags.GlobalFlags
	log log.Logger
}

func rotateCA(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &rotateCACmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	useLine, nameValidator := util.NamedPositionalArgsValidator(true, false, "VCLUSTER_NAME")
	rotateCACmd := &cobra.Command{
		Use:   "rotate-ca" + useLine,
		Short: "Rotates the CA certificate",
		Long: `##############################################################
################## vcluster certs rotate-ca ##################
##############################################################
Rotates the CA certificates of the given virtual cluster using
the current CA certificates.
The CA files (ca.{crt,key}) can be placed in the PKI directory
(either /data/pki or /var/lib/vcluster/pki) to issue new leaf
certificates to be signed by that CA.
If the ca.crt file is a bundle containing multiple certificates
the new CA cert must be the first one in the bundle.

Examples:
vcluster certs rotate-ca test
##############################################################
	`,
		Args:              nameValidator,
		ValidArgsFunction: completion.NewValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return certs.Rotate(cobraCmd.Context(), args[0], certs.RotationCmdCACerts, cmd.GlobalFlags, cmd.log)
		}}

	return rotateCACmd
}
