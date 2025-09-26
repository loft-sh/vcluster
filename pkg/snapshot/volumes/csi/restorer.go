package csi

import (
	"context"
	"errors"
	"fmt"

	snapshotsv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/clientset/versioned"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/snapshot/volumes"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
)

type Restorer struct {
	snapshotHandler
	vConfig *config.VirtualClusterConfig
}

func NewRestorer(vConfig *config.VirtualClusterConfig, kubeClient *kubernetes.Clientset, snapshotsClient *snapshotsv1.Clientset, eventRecorder record.EventRecorder, logger loghelper.Logger) (*Restorer, error) {
	if vConfig == nil {
		return nil, errors.New("virtual cluster config is required")
	}
	if kubeClient == nil {
		return nil, errors.New("kubernetes client is required")
	}
	if snapshotsClient == nil {
		return nil, errors.New("snapshot client is required")
	}
	if eventRecorder == nil {
		return nil, errors.New("event recorder is required")
	}
	if logger == nil {
		return nil, errors.New("logger is required")
	}

	restorer := &Restorer{
		snapshotHandler: snapshotHandler{
			kubeClient:      kubeClient,
			snapshotsClient: snapshotsClient,
			eventRecorder:   eventRecorder,
			logger:          logger,
		},
		vConfig: vConfig,
	}
	return restorer, nil
}

// Reconcile volumes restore request.
func (r *Restorer) Reconcile(ctx context.Context, requestObj runtime.Object, requestName string, request *volumes.RestoreRequestSpec, status *volumes.RestoreRequestStatus) error {
	r.logger.Infof("Restore volumes for restore request %s", requestName)
	var err error

	switch status.Phase {
	case volumes.RequestPhaseNotStarted:
		status.Phase = volumes.RequestPhaseInProgress
		fallthrough
	case volumes.RequestPhaseInProgress:
		err = r.reconcileInProgress(ctx, requestObj, requestName, request, status)
		if err != nil {
			return fmt.Errorf("failed to reconcile failed volumes snapshot request %s: %w", requestName, err)
		}
	case volumes.RequestPhaseCompleted:
		fallthrough
	case volumes.RequestPhasePartiallyFailed:
		fallthrough
	case volumes.RequestPhaseFailed:
		fallthrough
	case volumes.RequestPhaseSkipped:
		err = r.reconcileDone(ctx, requestName, status)
		if err != nil {
			return fmt.Errorf("failed to reconcile failed volumes snapshot request %s: %w", requestName, err)
		}
	default:
		return fmt.Errorf("invalid snapshot request phase: %s", status.Phase)
	}

	return nil
}
