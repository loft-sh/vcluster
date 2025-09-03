package snapshot

import (
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	requestLabel = "vcluster.loft.sh/snapshot-request"
	requestKey   = "snapshotRequest"
	optionsKey   = "snapshotOptions"

	RequestPhaseInProgress RequestPhase = "InProgress"
	RequestPhaseCompleted  RequestPhase = "Completed"
	RequestPhaseFailed     RequestPhase = "Failed"
)

type RequestPhase string

type Request struct {
	Spec   RequestSpec   `json:"spec"`
	Status RequestStatus `json:"status"`
}

type RequestSpec struct {
	Options Options `json:"-"`
}

type RequestStatus struct {
	Phase RequestPhase `json:"phase,omitempty"`
}

func UnmarshalSnapshotRequest(configMap *corev1.ConfigMap, secret *corev1.Secret) (*Request, error) {
	if configMap == nil {
		return nil, fmt.Errorf("config map is nil")
	}
	if secret == nil {
		return nil, fmt.Errorf("secret is nil")
	}

	// check if both ConfigMap and Secret have the required snapshot request label
	if _, ok := configMap.Labels[requestLabel]; !ok {
		return nil, fmt.Errorf("config map does not have the snapshot request label")
	}
	if _, ok := secret.Labels[requestLabel]; !ok {
		return nil, fmt.Errorf("secret does not have the snapshot request label")
	}

	// snapshot request, part 1 - ConfigMap with snapshot request phase (volume snapshot details will be added here)
	snapshotRequestJSON, ok := configMap.Data[requestKey]
	if !ok {
		return nil, fmt.Errorf("config map does not have the snapshot request")
	}
	var snapshotRequest Request
	err := json.Unmarshal([]byte(snapshotRequestJSON), &snapshotRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot request from ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}

	// unmarshal snapshot request from the Secret
	optionsJSON, ok := secret.Data[optionsKey]
	if !ok {
		return nil, fmt.Errorf("secret does not have the snapshot options")
	}
	var options Options
	err = json.Unmarshal(optionsJSON, &options)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot options: %w", err)
	}
	snapshotRequest.Spec.Options = options
	return &snapshotRequest, nil
}

func MarshalSnapshotRequest(vClusterNamespace string, snapshotRequest *Request) (*corev1.ConfigMap, *corev1.Secret, error) {
	if vClusterNamespace == "" {
		return nil, nil, fmt.Errorf("vClusterNamespace is not set")
	}
	if snapshotRequest == nil {
		return nil, nil, fmt.Errorf("snapshotRequest is nil")
	}

	// snapshot request, part 1 - ConfigMap
	snapshotRequestJSON, err := json.Marshal(snapshotRequest)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal snapshot request: %w", err)
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vClusterNamespace,
			Labels: map[string]string{
				requestLabel: "",
			},
		},
		Data: map[string]string{
			requestKey: string(snapshotRequestJSON),
		},
	}

	// snapshot request, part 2 - Secret with snapshot options (might contain credentials, hence using a Secret)
	optionsJSON, err := json.Marshal(snapshotRequest.Spec.Options)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal snapshot options: %w", err)
	}

	// Write snapshot options
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vClusterNamespace,
			Labels: map[string]string{
				requestLabel: "",
			},
		},
		Data: map[string][]byte{
			optionsKey: optionsJSON,
		},
	}

	return configMap, secret, nil
}
