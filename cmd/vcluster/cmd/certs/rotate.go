package certs

import (
	"context"
	"fmt"
	"os"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/certs"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/spf13/cobra"
)

const standaloneConfig = `
privateNodes:
  enabled: true
controlPlane:
  standalone:
    enabled: true
`

type rotateCmd struct {
	log        log.Logger
	pkiPath    string
	standalone bool
}

func rotate() *cobra.Command {
	cmd := &rotateCmd{
		log: log.GetInstance(),
	}

	rotateCmd := &cobra.Command{
		Use:   "rotate",
		Short: "Rotates control-plane client and server certs",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			return cmd.Run(cobraCmd.Context(), false)
		}}

	rotateCmd.Flags().StringVar(&cmd.pkiPath, "path", constants.PKIDir, "The path to the PKI directory")
	rotateCmd.Flags().BoolVar(&cmd.standalone, "standalone", false, "Signalizes if vCluster is running standalone")

	return rotateCmd
}

func rotateCA() *cobra.Command {
	cmd := &rotateCmd{
		log: log.GetInstance(),
	}

	rotateCACmd := &cobra.Command{
		Use:   "rotate-ca",
		Short: "Rotates the CA certificate",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			return cmd.Run(cobraCmd.Context(), true)
		}}

	rotateCACmd.Flags().StringVar(&cmd.pkiPath, "path", constants.PKIDir, "The path to the PKI directory")
	rotateCACmd.Flags().BoolVar(&cmd.standalone, "standalone", false, "Signalizes if vCluster is running standalone")

	return rotateCACmd
}

func (cmd *rotateCmd) Run(ctx context.Context, withCA bool) error {
	var vConfig *config.VirtualClusterConfig

	if cmd.standalone {
		cfg, err := config.ParseConfigBytes([]byte(standaloneConfig), os.Getenv("VCLUSTER_NAME"), nil)
		if err != nil {
			return fmt.Errorf("parsing vCluster config: %w", err)
		}
		vConfig = cfg
	} else {
		cfg, err := config.ParseConfig(constants.DefaultVClusterConfigLocation, os.Getenv("VCLUSTER_NAME"), nil)
		if err != nil {
			return fmt.Errorf("parsing vCluster config: %w", err)
		}
		vConfig = cfg
	}

	return certs.Rotate(ctx, vConfig, cmd.pkiPath, withCA, cmd.log)
}
