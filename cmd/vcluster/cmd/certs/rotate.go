package certs

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/certs"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/setup"
	"github.com/loft-sh/vcluster/pkg/util/servicecidr"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
)

type excludeFunc func(name string) bool

type rotateCmd struct {
	PKIPath         string
	excludeFiles    excludeFunc
	setCertValidity func(kubeadmConfig *kubeadmapi.InitConfiguration, duration time.Duration)
	log             log.Logger
}

func rotate() *cobra.Command {
	cmd := &rotateCmd{
		excludeFiles: excludeCAandSAFiles,
		setCertValidity: func(kubeadmConfig *kubeadmapi.InitConfiguration, duration time.Duration) {
			kubeadmConfig.CertificateValidityPeriod = &metav1.Duration{Duration: duration}
		},
		log: log.GetInstance(),
	}

	rotateCmd := &cobra.Command{
		Use:   "rotate",
		Short: "Rotates control-plane client and server certs",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			return cmd.Run(cobraCmd.Context())
		}}

	rotateCmd.Flags().StringVar(&cmd.PKIPath, "pki-path", constants.PKIDir, "Destination path to the PKI directory")

	return rotateCmd
}

func rotateCA() *cobra.Command {
	cmd := &rotateCmd{
		excludeFiles: excludeSAFiles,
		setCertValidity: func(kubeadmConfig *kubeadmapi.InitConfiguration, duration time.Duration) {
			kubeadmConfig.CACertificateValidityPeriod = &metav1.Duration{Duration: duration}
			kubeadmConfig.CertificateValidityPeriod = &metav1.Duration{Duration: duration}
		},
		log: log.GetInstance(),
	}

	rotateCACmd := &cobra.Command{
		Use:   "rotate-ca",
		Short: "Rotates the CA certificate",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			return cmd.Run(cobraCmd.Context())
		}}

	rotateCACmd.Flags().StringVar(&cmd.PKIPath, "pki-path", constants.PKIDir, "Destination path to the PKI directory")

	return rotateCACmd
}

func (cmd *rotateCmd) Run(ctx context.Context) error {
	vConfig, err := config.ParseConfig(constants.DefaultVClusterConfigLocation, os.Getenv("VCLUSTER_NAME"), nil)
	if err != nil {
		return fmt.Errorf("parsing vCluster config: %w", err)
	}

	vConfig.ControlPlaneConfig, vConfig.ControlPlaneNamespace, vConfig.ControlPlaneService, vConfig.WorkloadConfig, vConfig.WorkloadNamespace, vConfig.WorkloadService, err = pro.GetRemoteClient(vConfig)
	if err != nil {
		return fmt.Errorf("getting remote client: %w", err)
	}

	if err := setup.InitClients(vConfig); err != nil {
		return fmt.Errorf("initializing clients: %w", err)
	}

	serviceCIDR, err := servicecidr.GetServiceCIDR(ctx, &vConfig.Config, vConfig.WorkloadClient, vConfig.WorkloadService, vConfig.WorkloadNamespace)
	if err != nil {
		return fmt.Errorf("getting service cidr: %w", err)
	}

	kubeadmConfig, err := setup.GenerateInitKubeadmConfig(serviceCIDR, cmd.PKIPath, vConfig)
	if err != nil {
		return fmt.Errorf("generating kubeadm config: %w", err)
	}

	validityPeriod := os.Getenv("VCLUSTER_CERTS_VALIDITYPERIOD")
	if os.Getenv("DEVELOPMENT") == "true" && validityPeriod != "" {
		duration, err := time.ParseDuration(validityPeriod)
		if err != nil {
			return fmt.Errorf("parsing duration format: %w", err)
		}

		cmd.log.Info("Setting custom cert validity period")
		cmd.setCertValidity(kubeadmConfig, duration)
	}

	cmd.log.Info("Backing up previous PKI directory")
	backupDirName := fmt.Sprintf("%d", time.Now().Unix())
	backupDir := filepath.Join(cmd.PKIPath, "../pki.bak/"+backupDirName)
	if err := backupDirectory(cmd.PKIPath, backupDir); err != nil {
		return fmt.Errorf("backing up PKI directory")
	}
	cmd.log.Infof("Backup available at %s", backupDir)

	cmd.log.Info("Removing client and server leaf certificates from PKI directory")
	if err := removeFiles(cmd.PKIPath, cmd.excludeFiles); err != nil {
		return fmt.Errorf("removing files from PKI directory: %w", err)
	}

	if err := certs.GenerateCertificates(cmd.PKIPath, kubeadmConfig); err != nil {
		return fmt.Errorf("creating pki assets: %w", err)
	}

	// We cannot update the host secret when running standalone.
	if vConfig.ControlPlane.Standalone.Enabled {
		return nil
	}

	// Update the secret so in case of a restart without persistence we don't loose data.
	return updateSecret(ctx, vConfig.ControlPlaneNamespace, certs.CertSecretName(vConfig.Name), cmd.PKIPath, vConfig.ControlPlaneClient)
}

func backupDirectory(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}

	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if d.IsDir() {
			return os.MkdirAll(dstPath, 0755)
		}

		return copyFile(path, dstPath)
	})
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	return os.Chmod(dst, srcInfo.Mode())
}

func removeFiles(path string, f excludeFunc) error {
	return filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !f(d.Name()) {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("removing %s: %w", path, err)
			}
		}

		return nil
	})
}

func excludeCAandSAFiles(name string) bool {
	if name == certs.CACertName {
		return true
	}
	if name == certs.CAKeyName {
		return true
	}
	if strings.HasSuffix(name, fmt.Sprintf("-%s", certs.CACertName)) {
		return true
	}
	if strings.HasSuffix(name, fmt.Sprintf("-%s", certs.CAKeyName)) {
		return true
	}

	return excludeSAFiles(name)
}

func excludeSAFiles(name string) bool {
	if name == certs.ServiceAccountPublicKeyName {
		return true
	}
	if name == certs.ServiceAccountPrivateKeyName {
		return true
	}
	return false
}

func updateSecret(ctx context.Context, secretNamespace, secretName, pkiPath string, client kubernetes.Interface) error {
	secret, err := client.CoreV1().Secrets(secretNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("getting cert secret %s: %w", secretName, err)
	}

	data := map[string][]byte{}
	for k, v := range certs.CertMap() {
		d, err := os.ReadFile(filepath.Join(pkiPath, k))
		if err != nil {
			return fmt.Errorf("reading file %s: %w", k, err)
		}

		data[v] = d
	}
	secret.Data = data

	_, err = client.CoreV1().Secrets(secretNamespace).Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("updating cert secret %s: %w", secretName, err)
	}

	return nil
}
