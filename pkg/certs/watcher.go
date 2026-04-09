package certs

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"time"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/util/osutil"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	coordinationv1 "k8s.io/api/coordination/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
)

const (
	// DefaultCheckInterval is how often the background watcher checks for expiring certs.
	DefaultCheckInterval = 24 * time.Hour

	// certRotationLeaseDuration is how long the rotation lease is held.
	// Other replicas that see a held lease skip rotation.
	certRotationLeaseDuration = 5 * time.Minute
)

// StartCertWatcher starts a background goroutine that periodically checks
// whether leaf certificates are approaching expiry. If any cert is within the
// renewal threshold (90 days), it rotates all leaf certs and exits the process
// so the pod restarts with fresh certificates.
//
// In HA deployments, a Lease is used to coordinate rotation so only one
// replica rotates and exits at a time, avoiding simultaneous restarts.
func StartCertWatcher(
	ctx context.Context,
	interval time.Duration,
	serviceCIDR string,
	currentNamespace string,
	currentNamespaceClient kubernetes.Interface,
	certificateDir string,
	options *config.VirtualClusterConfig,
) {
	kubeadmConfig, err := GenerateInitKubeadmConfig(serviceCIDR, certificateDir, options)
	if err != nil {
		klog.Errorf("Failed to create kubeadm config for cert watcher: %v", err)
		return
	}

	// Allow overriding the check interval for testing.
	if dev, period := os.Getenv("DEVELOPMENT"), os.Getenv("VCLUSTER_CERT_CHECK_INTERVAL"); dev == "true" && period != "" {
		if d, err := time.ParseDuration(period); err == nil {
			interval = d
			klog.Infof("Using custom cert check interval: %s", d)
		}
	}

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
	// Add random jitter (up to 10% of interval) so HA replicas don't all
	// check at exactly the same time.
	jitter := time.Duration(rand.Int64N(int64(interval / 10)))
	ticker := time.NewTicker(interval + jitter)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			expiring, err := checkCertsExpiring(certificateDir)
			if err != nil {
				klog.Errorf("Error checking certificate expiry: %v", err)
				continue
			}
			if !expiring {
				continue
			}

			// In HA deployments (replicas > 1), acquire a lease so only one
			// replica rotates and restarts at a time. Single-replica deployments
			// skip the lease — they may not have coordination.k8s.io/leases RBAC
			// (it's only granted when HA or privateNodes are enabled in the chart).
			if options.ControlPlane.StatefulSet.HighAvailability.Replicas > 1 {
				if currentNamespaceClient == nil || !tryAcquireRotationLease(ctx, currentNamespaceClient, currentNamespace, options.Name) {
					klog.Infof("Another replica is handling cert rotation, skipping")
					continue
				}
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

// tryAcquireRotationLease attempts to create or update a Lease object to
// coordinate cert rotation across HA replicas. Returns true if this replica
// acquired the lease (and should perform rotation), false if another replica
// holds it.
func tryAcquireRotationLease(ctx context.Context, client kubernetes.Interface, namespace, vclusterName string) bool {
	leaseName := translate.SafeConcatName("vcluster", vclusterName, "cert-rotation")
	now := metav1.NewMicroTime(time.Now())

	holderID, err := os.Hostname()
	if err != nil {
		klog.Errorf("Failed to get hostname for lease: %v", err)
		return false
	}

	leaseDuration := int32(certRotationLeaseDuration.Seconds())

	existing, err := client.CoordinationV1().Leases(namespace).Get(ctx, leaseName, metav1.GetOptions{})
	if kerrors.IsNotFound(err) {
		// No lease exists — create one and acquire it.
		_, err = client.CoordinationV1().Leases(namespace).Create(ctx, &coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      leaseName,
				Namespace: namespace,
			},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity:       &holderID,
				LeaseDurationSeconds: &leaseDuration,
				AcquireTime:          &now,
				RenewTime:            &now,
			},
		}, metav1.CreateOptions{})
		if err != nil {
			klog.Errorf("Failed to create cert rotation lease: %v", err)
			return false
		}
		return true
	}
	if err != nil {
		klog.Errorf("Failed to get cert rotation lease: %v", err)
		return false
	}

	// Lease exists — check if it's expired.
	if existing.Spec.RenewTime != nil && existing.Spec.LeaseDurationSeconds != nil {
		expiry := existing.Spec.RenewTime.Time.Add(time.Duration(*existing.Spec.LeaseDurationSeconds) * time.Second)
		if time.Now().Before(expiry) {
			// Lease is still held by another replica.
			return false
		}
	}

	// Lease is expired — take it over.
	existing.Spec.HolderIdentity = &holderID
	existing.Spec.AcquireTime = &now
	existing.Spec.RenewTime = &now
	existing.Spec.LeaseDurationSeconds = &leaseDuration
	_, err = client.CoordinationV1().Leases(namespace).Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("Failed to update cert rotation lease: %v", err)
		return false
	}
	return true
}

// checkCertsExpiring checks whether any leaf cert is within the renewal window.
// Both standalone and platform-managed modes check certs on disk — in
// platform-managed mode, certs were downloaded from the secret at startup.
func checkCertsExpiring(certificateDir string) (bool, error) {
	return diskCertsExpiringSoon(certificateDir), nil
}

// rotateLeafCerts rotates all leaf certificates, preserving CA and SA keys.
// This aligns with the manual Rotate function in rotate.go:
// - backs up PKI before deletion
// - downloads certs from secret (non-standalone) to ensure CA/SA keys are on disk
// - handles standalone etcd cert cleanup
func rotateLeafCerts(
	ctx context.Context,
	currentNamespace string,
	currentNamespaceClient kubernetes.Interface,
	certificateDir string,
	kubeadmConfig *kubeadmapi.InitConfiguration,
	options *config.VirtualClusterConfig,
) error {
	// Back up the PKI directory before making changes so recovery is possible.
	backupDir, err := backupPKI(certificateDir)
	if err != nil {
		return fmt.Errorf("backing up PKI directory: %w", err)
	}
	klog.Infof("PKI backup available at %s", backupDir)

	// For non-standalone mode, download certs from the secret first to ensure
	// CA and SA keys are present on disk before we regenerate leaf certs.
	if !options.ControlPlane.Standalone.Enabled {
		if currentNamespaceClient == nil {
			return errors.New("non-standalone cert rotation requires a host cluster client")
		}

		secretName := CertSecretName(options.Name)
		secret, err := currentNamespaceClient.CoreV1().Secrets(currentNamespace).Get(ctx, secretName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("get cert secret: %w", err)
		}

		if err := downloadCertsFromSecret(secret, certificateDir); err != nil {
			return fmt.Errorf("download certs before renewal: %w", err)
		}
	}

	// Remove only leaf certs; preserve SA keys and CA keys.
	if err := removeFiles(certificateDir, excludeSAFiles, excludeCAFiles); err != nil {
		return fmt.Errorf("remove expiring leaf certs: %w", err)
	}

	// Regenerate missing certs.
	if err := generateCertificates(certificateDir, kubeadmConfig); err != nil {
		return fmt.Errorf("regenerate certs: %w", err)
	}

	// Handle standalone etcd: remove etcd server/peer certs so vCluster
	// standalone regenerates them dynamically at runtime.
	if options.ControlPlane.Standalone.Enabled {
		return removeStandaloneEtcdCerts(certificateDir)
	}

	// Non-standalone: sync renewed certs back to the host secret.
	secretName := CertSecretName(options.Name)
	if err := SyncSecret(ctx, currentNamespace, secretName, certificateDir, currentNamespaceClient); err != nil {
		return fmt.Errorf("sync renewed certs to secret: %w", err)
	}

	return nil
}
