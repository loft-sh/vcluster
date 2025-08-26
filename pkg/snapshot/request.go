package snapshot

import (
	"fmt"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
)

const (
	snapshotRequestAnnotation = "vcluster.loft.sh/snapshot-request"
	snapshotRequestKey        = "snapshotRequest"
)

type Request struct {
	Options Options `yaml:"options"`
}

func UnmarshalSnapshotRequest(configMap *corev1.ConfigMap) (*Request, error) {
	if configMap == nil {
		return nil, fmt.Errorf("config map is nil")
	}
	snapshotRequestYAML := configMap.Data[snapshotRequestKey]
	var snapshotRequest Request
	err := yaml.Unmarshal([]byte(snapshotRequestYAML), &snapshotRequest)
	if err != nil {
		return nil, err
	}
	return &snapshotRequest, nil
}

func MarshalSnapshotRequest(snapshotRequest *Request, configMap *corev1.ConfigMap) error {
	if snapshotRequest == nil {
		return fmt.Errorf("config is nil")
	}
	if configMap == nil {
		return fmt.Errorf("config map is nil")
	}
	snapshotRequestYAML, err := yaml.Marshal(snapshotRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot request: %w", err)
	}
	configMap.Data[snapshotRequestKey] = string(snapshotRequestYAML)
	return nil
}
