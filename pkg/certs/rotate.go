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
	"github.com/loft-sh/vcluster/pkg/config"
	setupconfig "github.com/loft-sh/vcluster/pkg/setup/config"
	"github.com/loft-sh/vcluster/pkg/util/servicecidr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Info struct {
	Filename   string    `json:"filename,omitempty"`
	Subject    string    `json:"subject,omitempty"`
	Issuer     string    `json:"issuer,omitempty"`
	ExpiryTime time.Time `json:"expiryTime"`
	Status     string    `json:"status,omitempty"` // "OK", "EXPIRED"
}

// Rotate rotates the certificates in the PKI directory.
// If running non-standalone it also updates the cert secret to contain the newly created certificates.
// Depending on the withCA argument this either means rotation the leaf certificates (withCA=false)
// or the whole PKI infra (withCA=true). In both cases the current SA pub and private keys are untouched.
func Rotate(ctx context.Context,
	vConfig *config.VirtualClusterConfig,
	pkiPath string,
	withCA bool,
	log log.Logger) error {
	var err error
	vConfig.HostConfig, vConfig.HostNamespace, err = setupconfig.InitClientConfig()
	if err != nil {
		return fmt.Errorf("getting remote client: %w", err)
	}

	if err := setupconfig.InitClients(vConfig); err != nil {
		return fmt.Errorf("initializing clients: %w", err)
	}

	serviceCIDR, err := servicecidr.GetServiceCIDR(ctx, &vConfig.Config, vConfig.HostClient, vConfig.Name, vConfig.HostNamespace)
	if err != nil {
		return fmt.Errorf("getting service cidr: %w", err)
	}

	kubeadmConfig, err := GenerateInitKubeadmConfig(serviceCIDR, pkiPath, vConfig)
	if err != nil {
		return fmt.Errorf("generating kubeadm config: %w", err)
	}

	var validityPeriod time.Duration
	dev, period := os.Getenv("DEVELOPMENT"), os.Getenv("VCLUSTER_CERTS_VALIDITYPERIOD")
	if dev == "true" && period != "" {
		validityPeriod, err = time.ParseDuration(period)
		if err != nil {
			return fmt.Errorf("parsing duration format: %w", err)
		}

		log.Info("Setting custom cert validity period")
		kubeadmConfig.CertificateValidityPeriod = &metav1.Duration{Duration: validityPeriod}
		if withCA {
			kubeadmConfig.CACertificateValidityPeriod = &metav1.Duration{Duration: validityPeriod}
		}
	}

	log.Info("Backing up previous PKI directory")
	backupDirName := fmt.Sprintf("%d", time.Now().Unix())
	backupDir := filepath.Join(pkiPath, "../pki.bak/"+backupDirName)
	if err := backupDirectory(pkiPath, backupDir); err != nil {
		return fmt.Errorf("backing up PKI directory: %w", err)
	}
	log.Infof("Backup available at %s", backupDir)

	excludeFuncs := []excludeFunc{excludeSAFiles}
	if !withCA {
		excludeFuncs = append(excludeFuncs, excludeCAFiles)
	}

	log.Info("Removing relevant certificate files from PKI directory")
	if err := removeFiles(pkiPath, excludeFuncs...); err != nil {
		return fmt.Errorf("removing files from PKI directory: %w", err)
	}

	if err := generateCertificates(pkiPath, kubeadmConfig); err != nil {
		return fmt.Errorf("creating pki assets: %w", err)
	}

	// In standalone there is no host secret so we skip updating the secret.
	if vConfig.ControlPlane.Standalone.Enabled {
		return nil
	}

	// Patch the secret so in case of a restart without persistence we don't loose data.
	return patchSecret(ctx, vConfig.HostNamespace, CertSecretName(vConfig.Name), pkiPath, vConfig.HostClient)
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

type excludeFunc func(name string) bool

func removeFiles(path string, excludeFuncs ...excludeFunc) error {
	return filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		for _, shouldExclude := range excludeFuncs {
			if shouldExclude(d.Name()) {
				return nil
			}
		}

		if err := os.Remove(path); err != nil {
			return fmt.Errorf("removing %s: %w", path, err)
		}

		return nil
	})
}

func excludeCAFiles(name string) bool {
	if name == CACertName {
		return true
	}
	if name == CAKeyName {
		return true
	}
	if strings.HasSuffix(name, fmt.Sprintf("-%s", CACertName)) {
		return true
	}
	if strings.HasSuffix(name, fmt.Sprintf("-%s", CAKeyName)) {
		return true
	}
	return false
}

func excludeSAFiles(name string) bool {
	if name == ServiceAccountPublicKeyName {
		return true
	}
	if name == ServiceAccountPrivateKeyName {
		return true
	}
	return false
}

func patchSecret(ctx context.Context, secretNamespace, secretName, pkiPath string, client kubernetes.Interface) error {
	secret, err := client.CoreV1().Secrets(secretNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("getting cert secret %s: %w", secretName, err)
	}

	data := map[string][]byte{}
	for k, v := range certMap {
		d, err := os.ReadFile(filepath.Join(pkiPath, k))
		if err != nil {
			return fmt.Errorf("reading file %s: %w", k, err)
		}

		data[v] = d
	}

	oldSecret := secret.DeepCopy()
	secret.Data = data
	patch := crclient.MergeFrom(oldSecret)
	patchBytes, err := patch.Data(secret)
	if err != nil {
		return fmt.Errorf("creating patch for secret %s: %w", secretName, err)
	}

	_, err = client.CoreV1().Secrets(secretNamespace).Patch(ctx, secretName, patch.Type(), patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("patching cert secret %s: %w", secretName, err)
	}

	return nil
}
