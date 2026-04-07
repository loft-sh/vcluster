package certs

import (
	"context"
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/util/osutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
)

const (
	// DefaultCheckInterval is how often the background watcher checks for expiring certs.
	DefaultCheckInterval = 24 * time.Hour
)

// StartCertWatcher starts a background goroutine that periodically checks
// whether leaf certificates are approaching expiry. If any cert is within the
// renewal threshold (90 days), it rotates all leaf certs and exits the process
// so the pod restarts with fresh certificates.
//
// This covers the case where a pod runs continuously for longer than
// (cert lifetime - 90 days) without restarting, which would otherwise cause
// certificate expiration since the startup-only check in EnsureCerts never re-runs.
func StartCertWatcher(
	ctx context.Context,
	interval time.Duration,
	currentNamespace string,
	currentNamespaceClient kubernetes.Interface,
	certificateDir string,
	options *config.VirtualClusterConfig,
	kubeadmConfig *kubeadmapi.InitConfiguration,
) {
	go runCertWatcher(ctx, interval, currentNamespace, currentNamespaceClient, certificateDir, options, kubeadmConfig)
}

func runCertWatcher(
	ctx context.Context,
	interval time.Duration,
	currentNamespace string,
	currentNamespaceClient kubernetes.Interface,
	certificateDir string,
	options *config.VirtualClusterConfig,
	kubeadmConfig *kubeadmapi.InitConfiguration,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			expiring, err := checkCertsExpiring(ctx, currentNamespace, currentNamespaceClient, certificateDir)
			if err != nil {
				klog.Errorf("Error checking certificate expiry: %v", err)
				continue
			}
			if !expiring {
				continue
			}

			klog.Infof("Leaf certificates are expiring soon, rotating and restarting")
			if err := rotateLeafCerts(ctx, currentNamespace, currentNamespaceClient, certificateDir, kubeadmConfig, options); err != nil {
				klog.Errorf("Error rotating certificates: %v", err)
				continue
			}

			klog.Infof("Leaf certificates rotated successfully, exiting for pod restart")
			osutil.Exit(0)
		}
	}
}

// checkCertsExpiring checks whether any leaf cert is within the renewal window.
func checkCertsExpiring(
	ctx context.Context,
	currentNamespace string,
	currentNamespaceClient kubernetes.Interface,
	certificateDir string,
) (bool, error) {
	// Standalone mode: check certs on disk
	if currentNamespaceClient == nil {
		return diskCertsExpiringSoon(certificateDir), nil
	}

	// Platform-managed: we need the vcluster name to look up the secret,
	// but we can derive it from the certificate dir's secret name convention.
	// Instead, just check the certs on disk since they were downloaded at startup.
	return diskCertsExpiringSoon(certificateDir), nil
}

// rotateLeafCerts rotates all leaf certificates, preserving CA and SA keys.
func rotateLeafCerts(
	ctx context.Context,
	currentNamespace string,
	currentNamespaceClient kubernetes.Interface,
	certificateDir string,
	kubeadmConfig *kubeadmapi.InitConfiguration,
	options *config.VirtualClusterConfig,
) error {
	// Remove only leaf certs; preserve SA keys and CA keys
	if err := removeFiles(certificateDir, excludeSAFiles, excludeCAFiles); err != nil {
		return fmt.Errorf("remove expiring leaf certs: %w", err)
	}

	// Regenerate missing certs
	if err := generateCertificates(certificateDir, kubeadmConfig); err != nil {
		return fmt.Errorf("regenerate certs: %w", err)
	}

	// For non-standalone mode, sync renewed certs back to the host secret
	if currentNamespaceClient != nil {
		secretName := CertSecretName(options.Name)
		// Verify the secret exists before syncing
		_, err := currentNamespaceClient.CoreV1().Secrets(currentNamespace).Get(ctx, secretName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("get cert secret before sync: %w", err)
		}

		if err := SyncSecret(ctx, currentNamespace, secretName, certificateDir, currentNamespaceClient); err != nil {
			return fmt.Errorf("sync renewed certs to secret: %w", err)
		}
	}

	return nil
}
