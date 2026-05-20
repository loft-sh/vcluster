package snapshot

import (
	"encoding/json"
	"fmt"

	snapshotapi "github.com/loft-sh/api/v4/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/snapshot/volumes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	RestoreRequestKey                                     = "restoreRequest"
	RequestPhaseRestoringVolumes snapshotapi.RequestPhase = "RestoringVolumes"
)

// RestoreRequest specifies vCluster restore request.
type RestoreRequest struct {
	snapshotapi.RequestMetadata `json:"metadata,omitempty"`
	Spec                        RestoreRequestSpec   `json:"spec,omitempty"`
	Status                      RestoreRequestStatus `json:"status,omitempty"`
}

func (r *RestoreRequest) Done() bool {
	return r.Status.Phase == snapshotapi.RequestPhaseCompleted ||
		r.Status.Phase == snapshotapi.RequestPhaseFailed ||
		r.Status.Phase == snapshotapi.RequestPhasePartiallyFailed
}

func (r *RestoreRequest) GetPhase() snapshotapi.RequestPhase {
	return r.Status.Phase
}

type RestoreRequestSpec struct {
	URL            string                     `json:"url,omitempty"`
	IncludeVolumes bool                       `json:"includeVolumes,omitempty"`
	VolumesRestore volumes.RestoreRequestSpec `json:"volumesRestore,omitempty"`
	Options        snapshotapi.Options        `json:"-"`
}

type RestoreRequestStatus struct {
	Phase          snapshotapi.RequestPhase     `json:"phase,omitempty"`
	VolumesRestore volumes.RestoreRequestStatus `json:"volumesRestore,omitempty"`
	Error          snapshotapi.SnapshotError    `json:"error,omitempty"`
}

func NewRestoreRequest(snapshotRequest snapshotapi.Request) (RestoreRequest, error) {
	restoreRequest := RestoreRequest{
		RequestMetadata: snapshotapi.RequestMetadata{
			CreationTimestamp: metav1.Now(),
		},
		Spec: RestoreRequestSpec{
			URL:            snapshotRequest.Spec.URL,
			IncludeVolumes: true,
			VolumesRestore: volumes.RestoreRequestSpec{
				Requests: []volumes.RestoreRequest{},
			},
		},
		Status: RestoreRequestStatus{
			Phase: snapshotapi.RequestPhaseNotStarted,
			VolumesRestore: volumes.RestoreRequestStatus{
				Phase:                  snapshotapi.VolumeSnapshotPhaseNotStarted,
				PersistentVolumeClaims: map[string]volumes.RestoreStatus{},
			},
		},
	}

	for _, volumeSnapshotRequest := range snapshotRequest.Spec.VolumeSnapshots.Requests {
		pvcName := fmt.Sprintf("%s/%s", volumeSnapshotRequest.PersistentVolumeClaim.Namespace, volumeSnapshotRequest.PersistentVolumeClaim.Name)
		snapshotStatus, ok := snapshotRequest.Status.VolumeSnapshots.Snapshots[pvcName]
		if !ok {
			return RestoreRequest{}, fmt.Errorf("volume snapshot status for PVC %s is not set", pvcName)
		}
		if snapshotStatus.Phase != snapshotapi.VolumeSnapshotPhaseCompleted {
			// Volume snapshot was not successfully created
			continue
		}
		if snapshotStatus.SnapshotHandle == "" {
			return RestoreRequest{}, fmt.Errorf("snapshot handle for PVC %s is not set in the snapshot request status", pvcName)
		}

		// add volume restore request
		volumeRestoreRequest := volumes.RestoreRequest{
			PersistentVolumeClaim:   volumeSnapshotRequest.PersistentVolumeClaim,
			CSIDriver:               volumeSnapshotRequest.CSIDriver,
			VolumeSnapshotClassName: volumeSnapshotRequest.VolumeSnapshotClassName,
			SnapshotHandle:          snapshotStatus.SnapshotHandle,
		}
		restoreRequest.Spec.VolumesRestore.Requests = append(restoreRequest.Spec.VolumesRestore.Requests, volumeRestoreRequest)

		// set volume restore status
		restoreRequest.Status.VolumesRestore.PersistentVolumeClaims[pvcName] = volumes.RestoreStatus{
			Phase: snapshotapi.VolumeSnapshotPhaseNotStarted,
		}
	}

	return restoreRequest, nil
}

func UnmarshalRestoreRequest(configMap *corev1.ConfigMap) (*RestoreRequest, error) {
	if configMap == nil {
		return nil, fmt.Errorf("config map is nil")
	}
	// check if ConfigMap has the required restore request label
	if _, ok := configMap.Labels[constants.RestoreRequestLabel]; !ok {
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
				constants.RestoreRequestLabel:    "",
			},
		},
		Data: map[string]string{
			RestoreRequestKey: string(restoreRequestJSON),
		},
	}

	return configMap, nil
}
