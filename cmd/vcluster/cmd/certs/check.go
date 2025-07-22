package certs

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/certs"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/spf13/cobra"
	certutil "k8s.io/client-go/util/cert"
)

type checkCmd struct {
	pkiPath string
	log     log.Logger
}

func check() *cobra.Command {
	cmd := &checkCmd{
		log: log.GetInstance(),
	}

	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "Checks the current certificates",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run()
		}}

	checkCmd.Flags().StringVar(&cmd.pkiPath, "path", constants.PKIDir, "The path to the PKI directory")

	return checkCmd
}

// Run checks the current certificates in the PKI directory and returns base information about those.
func (cmd *checkCmd) Run() error {
	now := time.Now()
	var certificateInfos []certs.Info
	err := filepath.WalkDir(cmd.pkiPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".crt" {
			return nil
		}

		c, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading file %s", path)
		}

		crts, err := certutil.ParseCertsPEM(c)
		if err != nil {
			return err
		}

		for _, crt := range crts {
			certificateInfos = append(certificateInfos, certs.Info{
				Filename:   d.Name(),
				Subject:    crt.Subject.CommonName,
				Issuer:     crt.Issuer.CommonName,
				ExpiryTime: crt.NotAfter,
				Status:     certStatus(crt, now),
			})
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("finding certificate information: %w", err)
	}

	if err := json.NewEncoder(os.Stdout).Encode(certificateInfos); err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}

	return nil
}

func certStatus(cert *x509.Certificate, now time.Time) string {
	if now.Before(cert.NotBefore) {
		return "NOT YET VALID"
	}
	if now.After(cert.NotAfter) {
		return "EXPIRED"
	}

	return "OK"
}
