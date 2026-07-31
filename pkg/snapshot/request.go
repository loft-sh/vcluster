package snapshot

import (
	"context"
	"errors"
	"fmt"
	"time"

	snapshotapi "github.com/loft-sh/api/v4/pkg/snapshot"
	"github.com/loft-sh/api/v4/pkg/snapshot/storage/azure"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/table"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/client-go/kubernetes"
)

var (
	ErrSnapshotRequestNotFound = errors.New("snapshot request not found")
)

// CreateSnapshotRequestResources creates snapshot request ConfigMap and Secret in the cluster. It returns the created
// snapshot request.
func CreateSnapshotRequestResources(ctx context.Context, vClusterNamespace, vClusterName string, vConfig *config.VirtualClusterConfig, options *snapshotapi.Options, kubeClient *kubernetes.Clientset) (*snapshotapi.Request, error) {
	if vConfig == nil {
		return nil, fmt.Errorf("config is nil")
	}
	if vConfig.ControlPlane.Standalone.Enabled {
		// Standalone has no host namespace; use the virtual cluster's own kube-system.
		vClusterNamespace = constants.VClusterStandaloneSnapshotNamespace
	}

	// first create the snapshot options Secret
	secret, err := snapshotapi.NewSnapshotOptionsSecret(vClusterNamespace, vClusterName, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot options Secret: %w", err)
	}
	secret, err = kubeClient.CoreV1().Secrets(vClusterNamespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot options Secret: %w", err)
	}

	// then create the snapshot request that will be reconciled by the controller
	snapshotRequest := &snapshotapi.Request{
		RequestMetadata: snapshotapi.RequestMetadata{
			Name:              secret.Name,
			CreationTimestamp: metav1.Now(),
		},
		Spec: snapshotapi.RequestSpec{
			URL: options.GetURL(),
		},
	}
	configMap, err := snapshotapi.NewSnapshotRequestConfigMap(vClusterNamespace, vClusterName, snapshotRequest)
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
		// Standalone uses the virtual cluster's kube-system, not a host namespace.
		return false, nil
	}

	return true, nil
}

func GetSnapshots(ctx context.Context, vClusterNamespace string, snapshotOpts *snapshotapi.Options, kubeClient *kubernetes.Clientset, log log.Logger) error {
	// First, try to get saved snapshots
	restoreClient := NewRestoreClient(*snapshotOpts, false)

	savedSnapshotRequest, err := restoreClient.GetSnapshotRequest(ctx)
	if azure.IsAzureFlagNotSetError(err) {
		return fmt.Errorf("failed to get saved snapshot request for URL %s: %w", snapshotOpts.GetURL(), err)
	} else if errors.Is(err, ErrSnapshotRequestNotFound) {
		log.Debugf("Saved snapshot request not found for URL %s", snapshotOpts.GetURL())
	} else if err != nil {
		log.Debugf("Failed to get saved snapshot request for URL %s: %v", snapshotOpts.GetURL(), err)
	}
	if savedSnapshotRequest != nil {
		savedSnapshotRequest.Status.Phase = snapshotapi.RequestPhaseCompleted
	}

	var inProgressSnapshotRequest *snapshotapi.Request
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
			snapshotRequest, err := snapshotapi.UnmarshalRequest(&configMap)
			if err != nil {
				return fmt.Errorf("failed to unmarshal snapshot request from ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
			}
			if snapshotRequest.Spec.URL != snapshotOpts.GetURL() {
				continue
			}
			if !snapshotRequest.Done() {
				inProgressSnapshotRequest = snapshotRequest
				break
			}
		}
		if inProgressSnapshotRequest != nil {
			break
		}
		continueOption = snapshotRequestConfigMaps.Continue
		listRequests = snapshotRequestConfigMaps.Continue != ""
	}

	if savedSnapshotRequest == nil && inProgressSnapshotRequest == nil {
		log.Infof("No snapshot found for the URL %s", snapshotOpts.GetURL())
		return nil
	}

	var url string
	var saved string
	var status snapshotapi.RequestPhase
	var age string
	var snapshotRequestToShow *snapshotapi.Request
	if inProgressSnapshotRequest != nil {
		snapshotRequestToShow = inProgressSnapshotRequest
	} else {
		snapshotRequestToShow = savedSnapshotRequest
	}
	url = snapshotRequestToShow.Spec.URL
	status = snapshotRequestToShow.Status.Phase
	age = duration.HumanDuration(time.Since(snapshotRequestToShow.CreationTimestamp.Time))
	if savedSnapshotRequest != nil {
		saved = "Yes"
	} else {
		saved = "No"
	}

	header := []string{"SNAPSHOT", "SAVED", "STATUS", "AGE"}
	values := [][]string{
		{
			url,
			saved,
			string(status),
			age,
		},
	}
	table.PrintTable(log, header, values)
	return nil
}

func DeleteSnapshotRequestResources(ctx context.Context, vClusterNamespace, vClusterName string, vConfig *config.VirtualClusterConfig, options *snapshotapi.Options, kubeClient *kubernetes.Clientset) error {
	// First, try to get saved snapshots
	restoreClient := NewRestoreClient(*options, false)

	savedSnapshotRequest, err := restoreClient.GetSnapshotRequest(ctx)
	if errors.Is(err, ErrSnapshotRequestNotFound) {
		return fmt.Errorf("saved snapshot request not found for URL %s", options.GetURL())
	} else if err != nil {
		return fmt.Errorf("failed to get saved snapshot request for URL %s: %w", options.GetURL(), err)
	}

	if savedSnapshotRequest == nil {
		return fmt.Errorf("saved snapshot request not found for URL %s", options.GetURL())
	}

	// update the snapshot request status to indicate that the snapshot request is in the cleaning up phase
	savedSnapshotRequest.CreationTimestamp = metav1.Now()
	savedSnapshotRequest.Status.Phase = snapshotapi.RequestPhaseDeleting

	// first create the snapshot options Secret
	secret, err := snapshotapi.NewSnapshotDeleteOptionsSecret(vClusterNamespace, vClusterName, options)
	if err != nil {
		return fmt.Errorf("failed to create snapshot options Secret: %w", err)
	}
	secret, err = kubeClient.CoreV1().Secrets(vClusterNamespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create snapshot options Secret: %w", err)
	}

	savedSnapshotRequest.Name = secret.Name
	configMap, err := snapshotapi.NewSnapshotRequestConfigMap(vClusterNamespace, vClusterName, savedSnapshotRequest)
	if err != nil {
		return fmt.Errorf("failed to create snapshot request ConfigMap: %w", err)
	}
	configMap.Name = secret.Name
	_, err = kubeClient.CoreV1().ConfigMaps(vClusterNamespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create snapshot request ConfigMap: %w", err)
	}
	return nil
}
