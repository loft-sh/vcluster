package snapshot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	snapshotTypes "github.com/loft-sh/vcluster/pkg/snapshot/types"
	"github.com/loft-sh/vcluster/pkg/snapshot/volumes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// APIVersion is the snapshot request API version.
	APIVersion = "v1beta1"

	RequestKey = "snapshotRequest"
	OptionsKey = "snapshotOptions"

	RequestPhaseNotStarted              RequestPhase = ""
	RequestPhaseCreatingVolumeSnapshots RequestPhase = "CreatingVolumeSnapshots"
	RequestPhaseCreatingEtcdBackup      RequestPhase = "CreatingEtcdBackup"
	RequestPhaseCompleted               RequestPhase = "Completed"
	RequestPhasePartiallyFailed         RequestPhase = "PartiallyFailed"
	RequestPhaseFailed                  RequestPhase = "Failed"

	DefaultRequestTTL = 24 * time.Hour
)

type RequestPhase string

type Request struct {
	RequestMetadata `json:"metadata,omitempty"`
	Spec            RequestSpec   `json:"spec,omitempty"`
	Status          RequestStatus `json:"status,omitempty"`
}

func (r *Request) Done() bool {
	return r.Status.Phase == RequestPhaseCompleted || r.Status.Phase == RequestPhaseFailed
}

type RequestMetadata struct {
	Name              string      `json:"name"`
	CreationTimestamp metav1.Time `json:"creationTimestamp,omitempty"`
}

type RequestSpec struct {
	URL             string                   `json:"url,omitempty"`
	IncludeVolumes  bool                     `json:"includeVolumes,omitempty"`
	VolumeSnapshots volumes.SnapshotsRequest `json:"volumeSnapshots,omitempty"`
	Options         Options                  `json:"-"`
}

type RequestStatus struct {
	Phase           RequestPhase                `json:"phase,omitempty"`
	VolumeSnapshots volumes.SnapshotsStatus     `json:"volumeSnapshots,omitempty"`
	Error           snapshotTypes.SnapshotError `json:"error,omitempty"`
}

// CreateSnapshotRequestResources creates snapshot request ConfigMap and Secret in the cluster. It returns the created
// snapshot request.
func CreateSnapshotRequestResources(ctx context.Context, vClusterNamespace, vClusterName string, vConfig *config.VirtualClusterConfig, options *Options, kubeClient *kubernetes.Clientset) (*Request, error) {
	if vConfig == nil {
		return nil, fmt.Errorf("config is nil")
	}
	if vConfig.ControlPlane.Standalone.Enabled {
		return nil, errors.New("creating snapshot request resources is currently not supported in standalone mode")
	}

	// first create the snapshot options Secret
	secret, err := CreateSnapshotOptionsSecret(vClusterNamespace, vClusterName, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot options Secret: %w", err)
	}
	secret.GenerateName = fmt.Sprintf("%s-snapshot-request-", vClusterName)
	secret, err = kubeClient.CoreV1().Secrets(vClusterNamespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot options Secret: %w", err)
	}

	// then create the snapshot request that will be reconciled by the controller
	snapshotRequest := &Request{
		RequestMetadata: RequestMetadata{
			Name:              secret.Name,
			CreationTimestamp: metav1.Now(),
		},
		Spec: RequestSpec{
			URL:            options.GetURL(),
			IncludeVolumes: options.IncludeVolumes,
		},
	}
	configMap, err := CreateSnapshotRequestConfigMap(vClusterNamespace, vClusterName, snapshotRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot request ConfigMap: %w", err)
	}
	configMap.Name = secret.Name
	_, err = kubeClient.CoreV1().ConfigMaps(vClusterNamespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot request ConfigMap: %w", err)
	}

	return snapshotRequest, nil
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
	if _, ok := configMap.Labels[constants.SnapshotRequestLabel]; !ok {
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
	if _, ok := secret.Labels[constants.SnapshotRequestLabel]; !ok {
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
				constants.SnapshotRequestLabel:   "",
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
				constants.SnapshotRequestLabel:   "",
			},
		},
		Data: map[string][]byte{
			OptionsKey: optionsJSON,
		},
	}

	return secret, nil
}
