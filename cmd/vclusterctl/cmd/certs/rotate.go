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
of the given virtual cluster. The CA is left untouched, so the
new leaf certificates are signed by the current CA.

To move to a new CA instead (e.g. a renewed CA issued by an
external PKI), replace ca.crt and ca.key in the PKI directory
(either /data/pki or /var/lib/vcluster/pki) and delete
server-ca.{crt,key} and client-ca.{crt,key} before running
this command. These are copies of the old CA that are only
recreated from ca.{crt,key} when missing.
If the ca.crt file is a bundle containing multiple certificates
the signing CA cert must be the first one in the bundle.

Examples:
vcluster -n test certs rotate test
##############################################################
	`,
		Args:              nameValidator,
		ValidArgsFunction: completion.NewValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return certs.Rotate(cobraCmd.Context(), args[0], certs.RotationCmdCerts, false, cmd.GlobalFlags, cmd.log)
		}}

	return rotateCmd
}

type rotateCACmd struct {
	*flags.GlobalFlags
	log   log.Logger
	force bool
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
Rotates the whole PKI of the given virtual cluster: a new
self-signed CA certificate is generated and all leaf
certificates are re-issued from it.

If the current CA is not self-signed (i.e. it was supplied by
an external PKI), the command refuses to run because rotating
would replace the external CA with a self-signed one. Use
--force to replace it anyway. To move to a renewed external
CA instead, see "vcluster certs rotate --help".

Examples:
vcluster certs rotate-ca test
##############################################################
	`,
		Args:              nameValidator,
		ValidArgsFunction: completion.NewValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return certs.Rotate(cobraCmd.Context(), args[0], certs.RotationCmdCACerts, cmd.force, cmd.GlobalFlags, cmd.log)
		}}

	rotateCACmd.Flags().BoolVar(&cmd.force, "force", false, "Rotate the CA even if the current CA certificate is not self-signed, e.g. supplied by an external PKI")

	return rotateCACmd
}
