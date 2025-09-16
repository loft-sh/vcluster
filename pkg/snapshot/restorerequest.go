package snapshot

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/snapshot/meta"
	"github.com/loft-sh/vcluster/pkg/snapshot/volumes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	RestoreRequestKey                         = "restoreRequest"
	RequestPhaseRestoringVolumes RequestPhase = "RestoringVolumes"
)

type RestoreRequest Request

func (r *RestoreRequest) Done() bool {
	return r.Status.Phase == RequestPhaseCompleted || r.Status.Phase == RequestPhaseFailed
}

func NewRestoreRequest(snapshotRequest Request) RestoreRequest {
	restoreRequest := RestoreRequest(snapshotRequest)
	restoreRequest.Spec.VolumeSnapshots.Spec.VolumeSnapshotConfigs = slices.Clone(snapshotRequest.Spec.VolumeSnapshots.Spec.VolumeSnapshotConfigs)
	restoreRequest.Spec.VolumeSnapshots.Status.Snapshots = maps.Clone(snapshotRequest.Spec.VolumeSnapshots.Status.Snapshots)

	// set volumes restore request phase to NotStarted only when the snapshot request
	// was successfully completed
	restoreRequest.Status.Phase = RequestPhaseNotStarted

	// reset overall volume snapshots status
	if restoreRequest.Spec.VolumeSnapshots.Status.Phase == volumes.RequestPhaseCompleted {
		restoreRequest.Spec.VolumeSnapshots.Status.Phase = volumes.RequestPhaseNotStarted
	}
	// reset volumes snapshot status for all volume snapshots
	for k, snapshotStatus := range restoreRequest.Spec.VolumeSnapshots.Status.Snapshots {
		if snapshotStatus.Phase == volumes.RequestPhaseCompleted {
			snapshotStatus.Phase = volumes.RequestPhaseNotStarted
			restoreRequest.Spec.VolumeSnapshots.Status.Snapshots[k] = snapshotStatus
		}
	}

	return restoreRequest
}

func UnmarshalRestoreRequest(configMap *corev1.ConfigMap) (*RestoreRequest, error) {
	if configMap == nil {
		return nil, fmt.Errorf("config map is nil")
	}
	// check if ConfigMap has the required restore request label
	if _, ok := configMap.Labels[meta.RestoreRequestLabel]; !ok {
		return nil, fmt.Errorf("config map does not have the restore request label")
	}

	// extract the restore request from the ConfigMap
	restoreRequestJSON, ok := configMap.Data[RestoreRequestKey]
	if !ok {
		return nil, fmt.Errorf("config map does not have the restore request")
	}
	var restoreRequest RestoreRequest
	err := json.Unmarshal([]byte(restoreRequestJSON), &restoreRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal restore request from ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}

	return &restoreRequest, nil
}

func CreateRestoreRequestConfigMap(vClusterNamespace, vClusterName string, restoreRequest RestoreRequest) (*corev1.ConfigMap, error) {
	if vClusterNamespace == "" {
		return nil, fmt.Errorf("vClusterNamespace is not set")
	}

	restoreRequestJSON, err := json.Marshal(restoreRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal restore request: %w", err)
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vClusterNamespace,
			Labels: map[string]string{
				constants.VClusterNamespaceLabel: vClusterNamespace,
				constants.VClusterNameLabel:      vClusterName,
				meta.RestoreRequestLabel:         "",
			},
		},
		Data: map[string]string{
			RestoreRequestKey: string(restoreRequestJSON),
		},
	}

	return configMap, nil
}
