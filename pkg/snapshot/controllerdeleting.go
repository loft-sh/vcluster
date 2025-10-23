package snapshot

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// reconcileNewRequest updates the snapshot request phase to "InProgress".
func (c *Reconciler) reconcileDeleting(_ context.Context, configMap *corev1.ConfigMap, snapshotRequest *Request) error {
	if snapshotRequest.Status.Phase != RequestPhaseDeleting {
		return fmt.Errorf("invalid phase for snapshot deletion request %s, expected %s, got %s", snapshotRequest.Name, RequestPhaseDeleting, snapshotRequest.Status.Phase)
	}
	c.logger.Debugf("Reconciling snapshot deletion request %s/%s", configMap.Namespace, configMap.Name)
	defer c.logger.Debugf("Reconciled snapshot deletion request %s/%s, new phase is %s", configMap.Namespace, configMap.Name, snapshotRequest.Status.Phase)

	if snapshotRequest.Spec.IncludeVolumes {
		snapshotRequest.Status.Phase = RequestPhaseDeletingVolumeSnapshots
		c.eventRecorder.Eventf(configMap, corev1.EventTypeNormal, "DeletingVolumeSnapshots", "Started to delete volume snapshots for snapshot request %s/%s", configMap.Namespace, configMap.Name)
	} else {
		snapshotRequest.Status.Phase = RequestPhaseDeletingEtcdBackup
		c.eventRecorder.Eventf(configMap, corev1.EventTypeNormal, "DeletingEtcdBackup", "Started to delete etcd backup for snapshot request %s/%s", configMap.Namespace, configMap.Name)
	}
	return nil
}

func (c *Reconciler) reconcileDeletingEtcdBackup(ctx context.Context, configMap *corev1.ConfigMap, snapshotRequest *Request) (bool, error) {
	if snapshotRequest.Status.Phase != RequestPhaseDeletingEtcdBackup {
		return false, fmt.Errorf("invalid phase for snapshot deletion request %s, expected %s, got %s", snapshotRequest.Name, RequestPhaseDeletingEtcdBackup, snapshotRequest.Status.Phase)
	}
	c.logger.Debugf("Deleting etcd backup at %s for snapshot deletion request %s/%s", snapshotRequest.Spec.URL, configMap.Namespace, configMap.Name)
	// Find snapshot request secret, it contains snapshot options (with the storage credentials) 🪪
	var secret corev1.Secret
	secretObjectKey := client.ObjectKey{
		Namespace: configMap.Namespace,
		Name:      configMap.Name,
	}
	err := c.client().Get(ctx, secretObjectKey, &secret)
	if kerrors.IsNotFound(err) {
		// Too soon and the client cache is not up to date? Requeue if this is a recently created snapshot request.
		if time.Since(configMap.CreationTimestamp.Time) < 10*time.Second {
			return true, nil
		}
		return false, fmt.Errorf("can't find snapshot deletion request Secret %s/%s: %w", configMap.Namespace, configMap.Name, err)
	} else if err != nil {
		return false, fmt.Errorf("failed to get snapshot deletion request Secret %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}

	// Extract snapshot options from the Secret 🔎
	snapshotOptions, err := UnmarshalSnapshotOptions(&secret)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal vcluster snapshot request from ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}

	// Create and save the snapshot! 💾
	c.logger.Debugf("Deleting vCluster snapshot in storage type %q", snapshotOptions.Type)
	snapshotClient := &Client{
		Request: snapshotRequest,
		Options: *snapshotOptions,
	}

	err = snapshotClient.Delete(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to delete etcd backup: %w", err)
	}
	c.logger.Infof("Deleted vCluster etcd backup at %s for the snapshot deletion request %s/%s", snapshotRequest.Spec.URL, configMap.Namespace, configMap.Name)
	snapshotRequest.Status.Phase = snapshotRequest.Status.Phase.Next()
	return false, nil
}
