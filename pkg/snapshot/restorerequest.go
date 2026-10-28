package snapshot

import (
	"encoding/json"
	"fmt"

	snapshotapi "github.com/loft-sh/api/v4/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	RestoreRequestKey                                        = "restoreRequest"
	RequestPhaseRestoringEtcdBackup snapshotapi.RequestPhase = "RestoringEtcdBackup"
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
	URL     string              `json:"url,omitempty"`
	Options snapshotapi.Options `json:"-"`
}

type RestoreRequestStatus struct {
	Phase snapshotapi.RequestPhase  `json:"phase,omitempty"`
	Error snapshotapi.SnapshotError `json:"error,omitempty"`
}

func NewRestoreRequest(snapshotRequest snapshotapi.Request) (RestoreRequest, error) {
	restoreRequest := RestoreRequest{
		RequestMetadata: snapshotapi.RequestMetadata{
			CreationTimestamp: metav1.Now(),
		},
		Spec: RestoreRequestSpec{
			URL: snapshotRequest.Spec.URL,
		},
		Status: RestoreRequestStatus{
			Phase: snapshotapi.RequestPhaseNotStarted,
		},
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
