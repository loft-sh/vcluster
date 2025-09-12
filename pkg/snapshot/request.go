package snapshot

import (
	"encoding/json"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	RequestLabel = "vcluster.loft.sh/snapshot-request"
	RequestKey   = "snapshotRequest"
	OptionsKey   = "snapshotOptions"

	RequestPhaseInProgress RequestPhase = "InProgress"
	RequestPhaseCompleted  RequestPhase = "Completed"
	RequestPhaseFailed     RequestPhase = "Failed"
)

type RequestPhase string

type Request struct {
	Name   string        `json:"name"`
	Spec   RequestSpec   `json:"spec"`
	Status RequestStatus `json:"status"`
}

func (r *Request) Done() bool {
	return r.Status.Phase == RequestPhaseCompleted || r.Status.Phase == RequestPhaseFailed
}

type RequestSpec struct {
	Options Options `json:"-"`
}

type RequestStatus struct {
	Phase RequestPhase `json:"phase,omitempty"`
}

// IsSnapshotRequestCreatedInHostCluster checks if the snapshot request resources are created in
// the host cluster.
func IsSnapshotRequestCreatedInHostCluster(config *config.VirtualClusterConfig) (bool, error) {
	if config == nil {
		return false, fmt.Errorf("config is nil")
	}
	if config.ControlPlane.Standalone.Enabled {
		return false, fmt.Errorf("creating snapshot requests is currently not supported in standalone mode")
	}

	return true, nil
}

func UnmarshalSnapshotRequest(configMap *corev1.ConfigMap) (*Request, error) {
	if configMap == nil {
		return nil, fmt.Errorf("config map is nil")
	}
	// check if ConfigMap has the required snapshot request label
	if _, ok := configMap.Labels[RequestLabel]; !ok {
		return nil, fmt.Errorf("config map does not have the snapshot request label")
	}

	// extract the snapshot request from the ConfigMap (volume snapshot details will be added here)
	snapshotRequestJSON, ok := configMap.Data[RequestKey]
	if !ok {
		return nil, fmt.Errorf("config map does not have the snapshot request")
	}
	var snapshotRequest Request
	err := json.Unmarshal([]byte(snapshotRequestJSON), &snapshotRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot request from ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}

	return &snapshotRequest, nil
}

func UnmarshalSnapshotOptions(secret *corev1.Secret) (*Options, error) {
	if secret == nil {
		return nil, fmt.Errorf("secret is nil")
	}

	// check if Secret has the required snapshot request label
	if _, ok := secret.Labels[RequestLabel]; !ok {
		return nil, fmt.Errorf("secret does not have the snapshot request label")
	}

	// extract snapshot options from the Secret
	optionsJSON, ok := secret.Data[OptionsKey]
	if !ok {
		return nil, fmt.Errorf("secret does not have the snapshot options")
	}
	var snapshotOptions Options
	err := json.Unmarshal(optionsJSON, &snapshotOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot options: %w", err)
	}

	return &snapshotOptions, nil
}

func CreateSnapshotRequestConfigMap(vClusterNamespace, vClusterName string, snapshotRequest *Request) (*corev1.ConfigMap, error) {
	if vClusterNamespace == "" {
		return nil, fmt.Errorf("vClusterNamespace is not set")
	}
	if snapshotRequest == nil {
		return nil, fmt.Errorf("snapshotRequest is nil")
	}

	// snapshot request, part 1 - ConfigMap
	snapshotRequestJSON, err := json.Marshal(snapshotRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snapshot request: %w", err)
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vClusterNamespace,
			Labels: map[string]string{
				constants.VClusterNamespaceLabel: vClusterNamespace,
				constants.VClusterNameLabel:      vClusterName,
				RequestLabel:                     "",
			},
		},
		Data: map[string]string{
			RequestKey: string(snapshotRequestJSON),
		},
	}

	return configMap, nil
}

func CreateSnapshotOptionsSecret(vClusterNamespace, vClusterName string, snapshotOptions *Options) (*corev1.Secret, error) {
	if vClusterNamespace == "" {
		return nil, fmt.Errorf("vClusterNamespace is not set")
	}
	if snapshotOptions == nil {
		return nil, fmt.Errorf("snapshotOptions is nil")
	}

	optionsJSON, err := json.Marshal(snapshotOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snapshot options: %w", err)
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vClusterNamespace,
			Labels: map[string]string{
				constants.VClusterNamespaceLabel: vClusterNamespace,
				constants.VClusterNameLabel:      vClusterName,
				RequestLabel:                     "",
			},
		},
		Data: map[string][]byte{
			OptionsKey: optionsJSON,
		},
	}

	return secret, nil
}
