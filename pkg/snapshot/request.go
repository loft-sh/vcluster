package snapshot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/log/table"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	snapshotTypes "github.com/loft-sh/vcluster/pkg/snapshot/types"
	"github.com/loft-sh/vcluster/pkg/snapshot/volumes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/client-go/kubernetes"
)

const (
	// APIVersion is the snapshot request API version.

	RequestKey = "snapshotRequest"
	OptionsKey = "snapshotOptions"

	DefaultRequestTTL = 24 * time.Hour
)

type Request struct {
	RequestMetadata `json:"metadata,omitempty"`
	Spec            RequestSpec   `json:"spec,omitempty"`
	Status          RequestStatus `json:"status,omitempty"`
}

func (r *Request) Done() bool {
	return r.Status.Phase == RequestPhaseCompleted || r.Status.Phase == RequestPhaseFailed
}

func (r *Request) GetPhase() RequestPhase {
	return r.Status.Phase
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

func GetSnapshots(ctx context.Context, args []string, globalFlags *flags.GlobalFlags, vClusterNamespace string, snapshotOpts *Options, kubeClient *kubernetes.Clientset, log log.Logger) error {
	// First, try to get saved snapshots
	restoreClient := RestoreClient{
		Snapshot: *snapshotOpts,
	}

	var snapshotRequests []Request
	savedSnapshotRequest, err := restoreClient.GetSnapshotRequest(ctx)
	if err != nil {
		log.Debugf("failed to get saved snapshot request: %v", err)
	}
	if savedSnapshotRequest != nil {
		// The snapshot request has been saved while it was in progress (it's
		// set to Completed/PartiallyFailed after the upload). Therefore, here
		// we update the phase to the correct final state.
		if savedSnapshotRequest.Spec.IncludeVolumes {
			if savedSnapshotRequest.Status.VolumeSnapshots.Phase == volumes.RequestPhaseCompleted {
				savedSnapshotRequest.Status.Phase = RequestPhaseCompleted
			} else {
				savedSnapshotRequest.Status.Phase = RequestPhasePartiallyFailed
			}
		} else {
			savedSnapshotRequest.Status.Phase = RequestPhaseCompleted
		}
		snapshotRequests = append(snapshotRequests, *savedSnapshotRequest)
	}

	listRequests := true
	var continueOption string
	for listRequests {
		listOptions := metav1.ListOptions{
			LabelSelector: constants.SnapshotRequestLabel,
			Continue:      continueOption,
		}
		snapshotRequestConfigMaps, err := kubeClient.CoreV1().ConfigMaps(vClusterNamespace).List(ctx, listOptions)
		if err != nil {
			return fmt.Errorf("failed to list snapshot request ConfigMaps: %w", err)
		}
		for _, configMap := range snapshotRequestConfigMaps.Items {
			snapshotRequest, err := UnmarshalSnapshotRequest(&configMap)
			if err != nil {
				return fmt.Errorf("failed to unmarshal snapshot request from ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
			}
			if snapshotRequest.Spec.URL != snapshotOpts.GetURL() {
				continue
			}
			if savedSnapshotRequest != nil &&
				(snapshotRequest.Name == savedSnapshotRequest.Name ||
					snapshotRequest.Spec.URL == savedSnapshotRequest.Spec.URL &&
						snapshotRequest.CreationTimestamp.Time.Before(savedSnapshotRequest.CreationTimestamp.Time)) {
				// Skip the local snapshot request if:
				// 1. it's the same request as the uploaded one, or
				// 2. it's older than the saved one.
				continue
			}

			snapshotRequests = append(snapshotRequests, *snapshotRequest)
		}

		continueOption = snapshotRequestConfigMaps.Continue
		listRequests = snapshotRequestConfigMaps.Continue != ""
	}

	if len(snapshotRequests) == 0 {
		log.Errorf("vCluster snapshot %q not found", snapshotOpts.GetURL())
		return nil
	}

	header := []string{"SNAPSHOT", "STATUS", "AGE"}
	values := make([][]string, len(snapshotRequests))
	for i, snapshotRequest := range snapshotRequests {
		values[i] = []string{
			snapshotRequest.Spec.URL,
			string(snapshotRequest.Status.Phase),
			duration.HumanDuration(time.Since(snapshotRequest.CreationTimestamp.Time)),
		}
	}
	table.PrintTable(log, header, values)
	return nil
}
