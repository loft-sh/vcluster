package snapshot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/snapshot/volumes"
	csiVolumes "github.com/loft-sh/vcluster/pkg/snapshot/volumes/csi"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	RestoreControllerFinalizer = "vcluster.loft.sh/restore-controller"
	restoreControllerName      = "vcluster-restore-controller"
)

type RestoreReconciler struct {
	reconcilerBase
	volumesRestorer volumes.Restorer
}

func NewRestoreController(registerContext *synccontext.RegisterContext) (*RestoreReconciler, error) {
	logger := loghelper.New(restoreControllerName)

	if registerContext == nil {
		return nil, errors.New("register context is nil")
	}
	if registerContext.Config == nil {
		return nil, errors.New("virtual cluster config is nil")
	}
	isHostMode, err := IsSnapshotRequestCreatedInHostCluster(registerContext.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to check if snapshot request is created in host cluster: %w", err)
	}

	var requestsManager ctrl.Manager
	if isHostMode {
		requestsManager = registerContext.HostManager
		logger.Infof("vcluster-restore-controller will reconcile snapshot requests in the host cluster")
	} else {
		requestsManager = registerContext.VirtualManager
		logger.Infof("vcluster-restore-controller will reconcile snapshot requests in the virtual cluster")
	}

	var volumesManager ctrl.Manager
	if registerContext.Config.PrivateNodes.Enabled {
		logger.Infof("vcluster-restore-controller will create volume snapshots in the virtual cluster")
		volumesManager = registerContext.VirtualManager
	} else {
		logger.Infof("vcluster-restore-controller will create volume snapshots in the host cluster")
		volumesManager = registerContext.HostManager
	}
	kubeClient, snapshotsClient, err := createClients(volumesManager.GetConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create kuberenetes clients: %w", err)
	}
	eventRecorder := requestsManager.GetEventRecorderFor(controllerName)
	volumesRestorer, err := csiVolumes.NewRestorer(registerContext.Config, kubeClient, snapshotsClient, eventRecorder, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create volume snapshotter: %w", err)
	}

	reconciler := reconcilerBase{
		vConfig:            registerContext.Config,
		requestsKubeClient: requestsManager.GetClient(),
		requestsManager:    requestsManager,
		logger:             logger,
		eventRecorder:      eventRecorder,
		isHostMode:         isHostMode,
		kind:               restoreReconciler,
		finalizer:          RestoreControllerFinalizer,
		requestKey:         RestoreRequestKey,
	}

	return &RestoreReconciler{
		reconcilerBase:  reconciler,
		volumesRestorer: volumesRestorer,
	}, nil
}

func (c *RestoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, retErr error) {
	c.logger.Infof("Reconciling restore request ConfigMap %s", req.NamespacedName)

	var configMap corev1.ConfigMap
	err := c.client().Get(ctx, req.NamespacedName, &configMap)
	if kerrors.IsNotFound(err) {
		c.logger.Infof("Restore request ConfigMap %s not found", req.NamespacedName)
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get restore request ConfigMap %s/%s: %w", req.Namespace, req.Name, err)
	}
	c.logger.Infof("Found ConfigMap %s/%s with vcluster restore request", configMap.Namespace, configMap.Name)

	// Restore request ConfigMap deleted -> we've got some cleaning up to do 🧹
	if !configMap.DeletionTimestamp.IsZero() {
		err = c.reconcileDeletedRequest(ctx, &configMap)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile deletion of restore request ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
		return ctrl.Result{}, nil
	}

	// Extract restore request details from the ConfigMap 🔎
	restoreRequest, err := UnmarshalRestoreRequest(&configMap)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to unmarshal vcluster snapshot request from ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}

	// Not done? Add the finalizer if it's not already set! 🔒
	if !restoreRequest.Done() {
		updated, err := c.addFinalizer(ctx, &configMap)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to add vCluster restore controller finalizer to the restore request ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
		if updated {
			c.eventRecorder.Eventf(&configMap, corev1.EventTypeNormal, "Created", "Restore request %s/%s has been created", configMap.Namespace, configMap.Name)
			return ctrl.Result{}, nil
		}
	}

	// patch restore request ConfigMap after the reconciliation
	configMapBeforeChange := client.MergeFrom(configMap.DeepCopy())
	defer func() {
		if retErr != nil {
			// something went wrong, recorde error and update restore request phase to Failed
			c.eventRecorder.Eventf(&configMap, corev1.EventTypeWarning, "Failed", "Restore request %s/%s has failed with error: %v", configMap.Namespace, configMap.Name, retErr)
			restoreRequest.Status.Phase = RequestPhaseFailed
		}
		updateErr := c.updateRequest(ctx, configMapBeforeChange, &configMap, *restoreRequest)
		if updateErr != nil {
			retErr = fmt.Errorf("failed to update restore request %s/%s: %w", configMap.Namespace, configMap.Name, updateErr)
		}
		if retErr != nil {
			retErr = errors.Join(retErr, updateErr)
		} else {
			retErr = updateErr
		}
	}()

	switch restoreRequest.Status.Phase {
	case RequestPhaseNotStarted:
		err = c.reconcileNewRequest(ctx, &configMap, restoreRequest)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile new restore request %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
	case RequestPhaseRestoringVolumes:
		volumesRestoreRequest := &restoreRequest.Spec.VolumesRestore
		volumesRestoreStatus := &restoreRequest.Status.VolumesRestore
		previousVolumesRestoreRequestPhase := volumesRestoreStatus.Phase
		err = c.volumesRestorer.Reconcile(ctx, &configMap, restoreRequest.Name, volumesRestoreRequest, volumesRestoreStatus)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile volume snapshots: %w", err)
		}
		switch volumesRestoreStatus.Phase {
		case volumes.RequestPhaseInProgress:
			if previousVolumesRestoreRequestPhase == volumes.RequestPhaseNotStarted {
				// volume restore request just got initialized and moved to in-progress
				return ctrl.Result{
					RequeueAfter: 5 * time.Second,
				}, nil
			} else {
				// ongoing volumes restore reconciliation, this may take some time, wait a bit before reconciling again
				return ctrl.Result{
					RequeueAfter: time.Minute,
				}, nil
			}
		case volumes.RequestPhaseSkipped:
			fallthrough
		case volumes.RequestPhaseCompleted:
			restoreRequest.Status.Phase = RequestPhaseCompleted
		case volumes.RequestPhaseFailed:
			restoreRequest.Status.Phase = RequestPhaseFailed
			restoreRequest.Status.Error.Message = volumesRestoreStatus.Error.Message
		case volumes.RequestPhasePartiallyFailed:
			restoreRequest.Status.Phase = RequestPhasePartiallyFailed
			restoreRequest.Status.Error.Message = volumesRestoreStatus.Error.Message
		default:
			return ctrl.Result{}, fmt.Errorf("unexpected volume snapshots request phase %s", volumesRestoreStatus.Phase)
		}
	case RequestPhasePartiallyFailed:
		fallthrough
	case RequestPhaseCompleted:
		err = c.reconcileCompletedRequest(ctx, &configMap, restoreRequest.RequestMetadata)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile completed restore request %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
	case RequestPhaseFailed:
		err = c.reconcileFailedRequest(ctx, &configMap, restoreRequest.RequestMetadata)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile failed restore request %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
	default:
		return ctrl.Result{}, fmt.Errorf("invalid restore request phase %s", restoreRequest.Status.Phase)
	}

	return ctrl.Result{}, nil
}

func (c *RestoreReconciler) Register() error {
	isRestoreRequest := predicate.NewPredicateFuncs(func(obj client.Object) bool {
		if obj.GetNamespace() != c.getRestoreRequestNamespace() {
			return false
		}
		objLabels := obj.GetLabels()
		if objLabels == nil {
			return false
		}
		_, ok := objLabels[constants.RestoreRequestLabel]
		return ok
	})

	return ctrl.NewControllerManagedBy(c.requestsManager).
		WithOptions(controller.Options{
			CacheSyncTimeout:        constants.DefaultCacheSyncTimeout,
			MaxConcurrentReconciles: 1,
		}).
		Named(restoreControllerName).
		For(&corev1.ConfigMap{}, builder.WithPredicates(isRestoreRequest)).
		Complete(c)
}

// reconcileNewRequest updates the snapshot request phase to "InProgress".
func (c *RestoreReconciler) reconcileNewRequest(_ context.Context, configMap *corev1.ConfigMap, restoreRequest *RestoreRequest) error {
	restoreRequest.Status.Phase = RequestPhaseRestoringVolumes
	c.eventRecorder.Eventf(configMap, corev1.EventTypeNormal, "RestoringVolumes", "Started restoring volumes for restore request %s/%s", configMap.Namespace, configMap.Name)
	return nil
}

func (c *RestoreReconciler) updateRequest(ctx context.Context, previousConfigMapState client.Patch, configMap *corev1.ConfigMap, restoreRequest RestoreRequest) error {
	restoreRequestJSON, err := json.Marshal(restoreRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal restore request to JSON: %w", err)
	}
	configMap.Data[RestoreRequestKey] = string(restoreRequestJSON)

	// patch restore request ConfigMap
	err = c.client().Patch(ctx, configMap, previousConfigMapState)
	if err != nil {
		return fmt.Errorf("failed to patch restore request ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}
	c.logger.Infof("Patched restore request %s/%s", configMap.Namespace, configMap.Name)
	return nil
}

func (c *RestoreReconciler) getRestoreRequestNamespace() string {
	if c.isHostMode {
		return c.vConfig.HostNamespace
	}
	return "kube-system"
}
