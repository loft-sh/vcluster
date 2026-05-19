package snapshot

import (
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewRequest(name string, creationTimestamp metav1.Time, options *Options) *Request {
	request := &Request{
		RequestMetadata: RequestMetadata{
			Name:              name,
			CreationTimestamp: creationTimestamp,
		},
	}
	if options != nil {
		request.Spec = RequestSpec{
			URL:            options.GetURL(),
			IncludeVolumes: options.IncludeVolumes,
		}
	}
	return request
}

func NewSnapshotOptionsSecret(vClusterNamespace, vClusterName string, options *Options) (*corev1.Secret, error) {
	secret, err := NewOptionsSecret(SnapshotRequestLabel, vClusterNamespace, vClusterName, options)
	if err != nil {
		return nil, err
	}
	secret.GenerateName = fmt.Sprintf("%s-snapshot-request-", vClusterName)
	return secret, nil
}

func NewSnapshotDeleteOptionsSecret(vClusterNamespace, vClusterName string, options *Options) (*corev1.Secret, error) {
	secret, err := NewOptionsSecret(SnapshotRequestLabel, vClusterNamespace, vClusterName, options)
	if err != nil {
		return nil, err
	}
	secret.GenerateName = fmt.Sprintf("%s-snapshot-request-delete-", vClusterName)
	return secret, nil
}

func NewOptionsSecret(requestLabel, vClusterNamespace, vClusterName string, options *Options) (*corev1.Secret, error) {
	if vClusterNamespace == "" {
		return nil, fmt.Errorf("vClusterNamespace is not set")
	}
	if options == nil {
		return nil, fmt.Errorf("snapshotOptions is nil")
	}

	optionsJSON, err := json.Marshal(options)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snapshot options: %w", err)
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vClusterNamespace,
			Labels: map[string]string{
				VClusterNamespaceLabel: vClusterNamespace,
				VClusterNameLabel:      vClusterName,
				requestLabel:           "",
			},
		},
		Data: map[string][]byte{
			OptionsKey: optionsJSON,
		},
	}, nil
}

func NewSnapshotRequestConfigMap(vClusterNamespace, vClusterName string, request *Request) (*corev1.ConfigMap, error) {
	if vClusterNamespace == "" {
		return nil, fmt.Errorf("vClusterNamespace is not set")
	}
	if request == nil {
		return nil, fmt.Errorf("snapshotRequest is nil")
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snapshot request: %w", err)
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vClusterNamespace,
			Labels: map[string]string{
				VClusterNamespaceLabel: vClusterNamespace,
				VClusterNameLabel:      vClusterName,
				SnapshotRequestLabel:   "",
			},
		},
		Data: map[string]string{
			RequestKey: string(requestJSON),
		},
	}, nil
}

func UnmarshalRequest(configMap *corev1.ConfigMap) (*Request, error) {
	if configMap == nil {
		return nil, fmt.Errorf("config map is nil")
	}
	if _, ok := configMap.Labels[SnapshotRequestLabel]; !ok {
		return nil, fmt.Errorf("config map does not have the snapshot request label")
	}

	requestJSON, ok := configMap.Data[RequestKey]
	if !ok {
		return nil, fmt.Errorf("config map does not have the snapshot request")
	}

	var request Request
	if err := json.Unmarshal([]byte(requestJSON), &request); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot request from ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}
	return &request, nil
}

func UnmarshalOptions(secret *corev1.Secret) (*Options, error) {
	if secret == nil {
		return nil, fmt.Errorf("secret is nil")
	}
	if !hasOptionsRequestLabel(secret.Labels) {
		return nil, fmt.Errorf("secret does not have a snapshot or restore request label")
	}

	optionsJSON, ok := secret.Data[OptionsKey]
	if !ok {
		return nil, fmt.Errorf("secret does not have the snapshot options")
	}

	var options Options
	if err := json.Unmarshal(optionsJSON, &options); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot options: %w", err)
	}
	return &options, nil
}

func hasOptionsRequestLabel(labels map[string]string) bool {
	if labels == nil {
		return false
	}
	if _, ok := labels[SnapshotRequestLabel]; ok {
		return true
	}
	if _, ok := labels[RestoreRequestLabel]; ok {
		return true
	}
	return false
}
