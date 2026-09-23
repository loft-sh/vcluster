package snapshot

import (
	"context"
	"fmt"

	snapshotapi "github.com/loft-sh/api/v4/pkg/snapshot"
	corev1 "k8s.io/api/core/v1"
)

// reconcileNewRequest updates the snapshot request phase to "InProgress".
func (c *Reconciler) reconcileDeleting(_ context.Context, configMap *corev1.ConfigMap, snapshotRequest *snapshotapi.Request) error {
	if snapshotRequest.Status.Phase != snapshotapi.RequestPhaseDeleting {
		return fmt.Errorf("invalid phase for snapshot deletion request %s, expected %s, got %s", snapshotRequest.Name, snapshotapi.RequestPhaseDeleting, snapshotRequest.Status.Phase)
	}
	c.logger.Debugf("Reconciling snapshot deletion request %s/%s", configMap.Namespace, configMap.Name)
	defer c.logger.Debugf("Reconciled snapshot deletion request %s/%s, new phase is %s", configMap.Namespace, configMap.Name, snapshotRequest.Status.Phase)
	snapshotRequest.Status.Phase = snapshotapi.RequestPhaseDeletingEtcdBackup
	c.eventRecorder.Eventf(
		configMap,
		nil,
		corev1.EventTypeNormal,
		"DeletingEtcdBackup",
		"ReconcileSnapShotRequest",
		"Started to delete etcd backup for snapshot request %s/%s",
		configMap.Namespace,
		configMap.Name,
	)
	return nil
}

func (c *Reconciler) reconcileDeletingEtcdBackup(ctx context.Context, configMap *corev1.ConfigMap, snapshotRequest *snapshotapi.Request) (bool, error) {
	if snapshotRequest.Status.Phase != snapshotapi.RequestPhaseDeletingEtcdBackup {
		return false, fmt.Errorf("invalid phase for snapshot deletion request %s, expected %s, got %s", snapshotRequest.Name, snapshotapi.RequestPhaseDeletingEtcdBackup, snapshotRequest.Status.Phase)
	}
	c.logger.Debugf("Deleting etcd backup at %s for snapshot deletion request %s/%s", snapshotRequest.Spec.URL, configMap.Namespace, configMap.Name)
	// Obtain the snapshot options (URL + credentials). Standalone, platform-connected tenants pull
	// them from the platform in memory; everyone else reads them from the pushed request Secret.
	snapshotOptions, requeue, err := c.resolveSnapshotOptions(ctx, configMap, snapshotRequest)
	if err != nil {
		return false, err
	}
	if requeue {
		return true, nil
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
