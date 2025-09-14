package snapshot

import (
	"encoding/json"
	"fmt"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/snapshot/meta"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	RestoreRequestKey = "restoreRequest"
)

type RestoreRequest Request

func NewRestoreRequest(snapshotRequest Request) RestoreRequest {
	restoreRequest := RestoreRequest(snapshotRequest)
	restoreRequest.Status.Phase = RequestPhaseNotStarted
	return restoreRequest
}

func CreateRestoreRequestConfigMap(vClusterNamespace, vClusterName string, snapshotRequest Request) (*corev1.ConfigMap, error) {
	if vClusterNamespace == "" {
		return nil, fmt.Errorf("vClusterNamespace is not set")
	}
	restoreRequest := NewRestoreRequest(snapshotRequest)

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
