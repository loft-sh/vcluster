package certs

import (
	"context"
	"encoding/json"
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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
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

	// certRotationExitDelayMin and certRotationExitDelayMax define the random
	// delay window before exiting in standalone mode after a successful
	// rotation.
	certRotationExitDelayMin = 5 * time.Second
	certRotationExitDelayMax = 5 * time.Minute

	initialPendingRolloutRetryDelay = time.Minute
	maxPendingRolloutRetryDelay     = time.Hour

	certRotationAnnotation = "vcluster.loft.sh/cert-rotation-at"
)

var certRotationRolloutBackoff = wait.Backoff{
	Duration: time.Second,
	Factor:   2.0,
	Jitter:   0.1,
	Steps:    5,
}

// StartCertWatcher starts a background goroutine that periodically checks
// whether leaf certificates are approaching expiry. If any cert is within the
// renewal threshold (90 days), it rotates all leaf certs and then either exits
// in standalone mode or triggers workload rollouts in clustered mode so the
// new certs are picked up everywhere.
//
// In HA deployments, a Lease is used to coordinate rotation so only one
// replica rotates at a time, avoiding concurrent regeneration.
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

	var (
		retryTimer <-chan time.Time
		// pendingRolloutAt means cert rotation already succeeded and only the
		// workload rollout still needs to converge. While this is set, the
		// watcher retries only propagation and never regenerates certs again.
		pendingRolloutAt  string
		pendingRetryDelay = initialPendingRolloutRetryDelay
	)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-retryTimer:
			retryTimer = nil
		}

		if pendingRolloutAt != "" {
			if err := finishPendingRollout(ctx, currentNamespace, currentNamespaceClient, options, pendingRolloutAt); err != nil {
				klog.Errorf("Error finishing pending cert rollout: %v", err)
				retryTimer = time.After(pendingRetryDelay)
				pendingRetryDelay = min(pendingRetryDelay*2, maxPendingRolloutRetryDelay)
				continue
			}

			pendingRolloutAt = ""
			pendingRetryDelay = initialPendingRolloutRetryDelay
			retryTimer = nil
			continue
		}

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

		if options.ControlPlane.Standalone.Enabled {
			if err := restartAfterRotation(ctx, currentNamespace, currentNamespaceClient, options, ""); err != nil {
				klog.Errorf("Error restarting after rotation: %v", err)
				continue
			}
			continue
		}

		// Keep the same rollout timestamp across retries so the patch stays
		// idempotent while we wait for the workload controllers to converge.
		pendingRolloutAt = time.Now().UTC().Format(time.RFC3339Nano)
		if err := finishPendingRollout(ctx, currentNamespace, currentNamespaceClient, options, pendingRolloutAt); err != nil {
			klog.Errorf("Error finishing pending cert rollout: %v", err)
			retryTimer = time.After(pendingRetryDelay)
			pendingRetryDelay = min(pendingRetryDelay*2, maxPendingRolloutRetryDelay)
			continue
		}

		pendingRolloutAt = ""
		pendingRetryDelay = initialPendingRolloutRetryDelay
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

func restartAfterRotation(
	ctx context.Context,
	currentNamespace string,
	currentNamespaceClient kubernetes.Interface,
	options *config.VirtualClusterConfig,
	rolloutAt string,
) error {
	if options.ControlPlane.Standalone.Enabled {
		delay := certRotationExitDelayMin
		if certRotationExitDelayMax > certRotationExitDelayMin {
			delay += time.Duration(rand.Int64N(int64(certRotationExitDelayMax - certRotationExitDelayMin)))
		}
		// Spread standalone restarts out so multiple instances do not all
		// disappear at exactly the same moment after rotating.
		klog.Infof("Standalone cert rotation complete, exiting after random delay %s", delay)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		klog.Infof("Leaf certificates rotated successfully in standalone mode, exiting for restart")
		osutil.Exit(0)
		return nil
	}

	return finishPendingRollout(ctx, currentNamespace, currentNamespaceClient, options, rolloutAt)
}

func finishPendingRollout(
	ctx context.Context,
	currentNamespace string,
	currentNamespaceClient kubernetes.Interface,
	options *config.VirtualClusterConfig,
	rolloutAt string,
) error {
	if currentNamespaceClient == nil {
		return errors.New("clustered cert rotation requires a host cluster client")
	}

	// Restart deployed etcd first so the control plane comes back against the
	// refreshed etcd serving/client certs. Only attempt this when the user has
	// explicitly enabled deployed etcd in the configuration.
	if options.ControlPlane.BackingStore.Etcd.Deploy.Enabled {
		if err := rolloutDeployedEtcdWithRetry(ctx, currentNamespaceClient, currentNamespace, options.Name, rolloutAt); err != nil {
			return err
		}
	}

	if err := rolloutControlPlaneWithRetry(ctx, currentNamespaceClient, currentNamespace, options.Name, rolloutAt); err != nil {
		return err
	}

	klog.Infof("Leaf certificates rotated successfully, triggered workload rollout at %s", rolloutAt)
	return nil
}

func rolloutControlPlaneWithRetry(ctx context.Context, client kubernetes.Interface, namespace, name, rolloutAt string) error {
	return retryWithBackoff(ctx, "trigger control-plane rollout", func(ctx context.Context) error {
		// Patching spec.template annotations triggers a rollout for Deployments
		// and StatefulSets using RollingUpdate. StatefulSets configured with
		// OnDelete will not restart pods automatically.
		err := patchStatefulSetTemplateAnnotation(ctx, client, namespace, name, certRotationAnnotation, rolloutAt)
		if err == nil {
			return nil
		}
		if !kerrors.IsNotFound(err) {
			return err
		}

		err = patchDeploymentTemplateAnnotation(ctx, client, namespace, name, certRotationAnnotation, rolloutAt)
		if err == nil {
			return nil
		}
		if kerrors.IsNotFound(err) {
			return fmt.Errorf("control-plane workload %s/%s not found", namespace, name)
		}
		return err
	})
}

func rolloutDeployedEtcdWithRetry(ctx context.Context, client kubernetes.Interface, namespace, vclusterName, rolloutAt string) error {
	etcdName := vclusterName + "-etcd"
	return retryWithBackoff(ctx, "trigger deployed etcd rollout", func(ctx context.Context) error {
		err := patchStatefulSetTemplateAnnotation(ctx, client, namespace, etcdName, certRotationAnnotation, rolloutAt)
		if kerrors.IsNotFound(err) {
			return nil
		}
		return err
	})
}

func retryWithBackoff(ctx context.Context, description string, fn func(context.Context) error) error {
	var lastErr error

	err := wait.ExponentialBackoffWithContext(ctx, certRotationRolloutBackoff, func(ctx context.Context) (bool, error) {
		if err := fn(ctx); err != nil {
			lastErr = err
			klog.Infof("%s failed, retrying: %v", description, err)
			return false, nil
		}
		return true, nil
	})
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return fmt.Errorf("%s interrupted: %w", description, err)
	}
	if lastErr != nil {
		return fmt.Errorf("%s after retries: %w", description, lastErr)
	}
	return fmt.Errorf("%s interrupted: %w", description, err)
}

func patchStatefulSetTemplateAnnotation(ctx context.Context, client kubernetes.Interface, namespace, name, key, value string) error {
	patchBytes, err := podTemplateAnnotationPatch(key, value)
	if err != nil {
		return err
	}

	_, err = client.AppsV1().StatefulSets(namespace).Patch(ctx, name, types.MergePatchType, patchBytes, metav1.PatchOptions{})
	return err
}

func patchDeploymentTemplateAnnotation(ctx context.Context, client kubernetes.Interface, namespace, name, key, value string) error {
	patchBytes, err := podTemplateAnnotationPatch(key, value)
	if err != nil {
		return err
	}

	_, err = client.AppsV1().Deployments(namespace).Patch(ctx, name, types.MergePatchType, patchBytes, metav1.PatchOptions{})
	return err
}

func podTemplateAnnotationPatch(key, value string) ([]byte, error) {
	return json.Marshal(map[string]any{
		"spec": map[string]any{
			"template": map[string]any{
				"metadata": map[string]any{
					"annotations": map[string]string{
						key: value,
					},
				},
			},
		},
	})
}

// checkCertsExpiring checks whether any leaf cert is within the renewal window.
// Both standalone and platform-managed modes check certs on disk — in
// platform-managed mode, certs were downloaded from the secret at startup.
func checkCertsExpiring(certificateDir string) (bool, error) {
	return diskCertsExpiringSoon(certificateDir), nil
}

// rotateLeafCerts rotates all leaf certificates.
// In standalone mode, the local PKI directory is the source of truth and gets
// updated in place.
// In clustered mode, a temporary PKI directory is generated and synced to the
// secret, which becomes the source of truth for the subsequent rollout.
func rotateLeafCerts(
	ctx context.Context,
	currentNamespace string,
	currentNamespaceClient kubernetes.Interface,
	certificateDir string,
	kubeadmConfig *kubeadmapi.InitConfiguration,
	options *config.VirtualClusterConfig,
) error {
	// Back up the current on-disk PKI before making changes or preparing a
	// replacement source of truth.
	backupDir, err := backupPKI(certificateDir)
	if err != nil {
		return fmt.Errorf("backing up PKI directory: %w", err)
	}
	klog.Infof("PKI backup available at %s", backupDir)

	if options.ControlPlane.Standalone.Enabled {
		return rotateLeafCertsStandalone(certificateDir, kubeadmConfig)
	}

	return rotateLeafCertsInCluster(ctx, currentNamespace, currentNamespaceClient, kubeadmConfig, options)
}

func rotateLeafCertsStandalone(certificateDir string, kubeadmConfig *kubeadmapi.InitConfiguration) error {
	// Remove only leaf certs; preserve SA keys and CA keys.
	if err := removeFiles(certificateDir, excludeSAFiles, excludeCAFiles); err != nil {
		return fmt.Errorf("remove expiring leaf certs: %w", err)
	}

	// Regenerate missing certs.
	if err := generateCertificates(certificateDir, kubeadmConfig); err != nil {
		return fmt.Errorf("regenerate certs: %w", err)
	}

	// Standalone etcd server and peer certs are regenerated dynamically at runtime.
	if err := removeStandaloneEtcdCerts(certificateDir); err != nil {
		return err
	}

	return nil
}

func rotateLeafCertsInCluster(
	ctx context.Context,
	currentNamespace string,
	currentNamespaceClient kubernetes.Interface,
	kubeadmConfig *kubeadmapi.InitConfiguration,
	options *config.VirtualClusterConfig,
) error {
	if currentNamespaceClient == nil {
		return errors.New("clustered cert rotation requires a host cluster client")
	}

	// Build renewed certs in a temporary PKI tree first. In clustered mode the
	// Secret is the commit point; the current pod's live /pki stays untouched
	// until the rollout restarts pods and EnsureCerts rehydrates from the Secret.
	tempDir, err := os.MkdirTemp("/tmp", "pki-rotate-*")
	if err != nil {
		return fmt.Errorf("create temp pki dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	secretName := CertSecretName(options.Name)
	secret, err := currentNamespaceClient.CoreV1().Secrets(currentNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get cert secret: %w", err)
	}

	if err := downloadCertsFromSecret(secret, tempDir); err != nil {
		return fmt.Errorf("download certs before renewal: %w", err)
	}

	if err := removeFiles(tempDir, excludeSAFiles, excludeCAFiles); err != nil {
		return fmt.Errorf("remove expiring leaf certs: %w", err)
	}

	tempKubeadmConfig := *kubeadmConfig
	tempKubeadmConfig.CertificatesDir = tempDir

	if err := generateCertificates(tempDir, &tempKubeadmConfig); err != nil {
		return fmt.Errorf("regenerate certs: %w", err)
	}

	// Syncing the Secret is the safety boundary for in-cluster rotation. If this
	// fails, we leave existing pods on the old committed cert set.
	if err := SyncSecret(ctx, currentNamespace, secretName, tempDir, currentNamespaceClient); err != nil {
		return fmt.Errorf("sync renewed certs to secret: %w", err)
	}

	return nil
}
