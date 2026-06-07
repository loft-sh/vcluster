package snapshot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	snapshotapi "github.com/loft-sh/api/v4/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	ControllerFinalizer = "vcluster.loft.sh/snapshot-controller"
	controllerName      = "vcluster-snapshot-controller"
)

type Reconciler struct {
	reconcilerBase
	vConfig                    *config.VirtualClusterConfig
	snapshotRequestsKubeClient client.Client
	snapshotRequestsManager    ctrl.Manager
	logger                     loghelper.Logger
	eventRecorder              events.EventRecorder
	isHostMode                 bool
}

func NewController(registerContext *synccontext.RegisterContext) (*Reconciler, error) {
	logger := loghelper.New(controllerName)

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

	var snapshotRequestsManager ctrl.Manager
	if isHostMode {
		snapshotRequestsManager = registerContext.HostManager
		logger.Infof("vcluster-snapshot-controller will reconcile snapshot requests in the host cluster")
	} else {
		snapshotRequestsManager = registerContext.VirtualManager
		logger.Infof("vcluster-snapshot-controller will reconcile snapshot requests in the virtual cluster")
	}
	eventRecorder := snapshotRequestsManager.GetEventRecorder(controllerName)

	reconciler := reconcilerBase{
		vConfig:            registerContext.Config,
		requestsKubeClient: snapshotRequestsManager.GetClient(),
		requestsManager:    snapshotRequestsManager,
		logger:             logger,
		eventRecorder:      eventRecorder,
		isHostMode:         isHostMode,
		kind:               snapshotReconciler,
		finalizer:          ControllerFinalizer,
		requestKey:         snapshotapi.RequestKey,
	}
	return &Reconciler{
		reconcilerBase:             reconciler,
		vConfig:                    registerContext.Config,
		snapshotRequestsKubeClient: snapshotRequestsManager.GetClient(),
		snapshotRequestsManager:    snapshotRequestsManager,
		logger:                     logger,
		eventRecorder:              eventRecorder,
		isHostMode:                 isHostMode,
	}, nil
}

func (c *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, retErr error) {
	c.logger.Debugf("Reconciling snapshot request ConfigMap %s", req.NamespacedName)

	var configMap corev1.ConfigMap
	err := c.client().Get(ctx, req.NamespacedName, &configMap)
	if kerrors.IsNotFound(err) {
		c.logger.Debugf("Snapshot request ConfigMap %s not found", req.NamespacedName)
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get snapshot request ConfigMap %s/%s: %w", req.Namespace, req.Name, err)
	}
	c.logger.Debugf("Found ConfigMap %s/%s with vcluster snapshot request", configMap.Namespace, configMap.Name)

	// Snapshot request ConfigMap deleted -> we've got some cleaning up to do 🧹
	if !configMap.DeletionTimestamp.IsZero() {
		err = c.reconcileDeletedRequest(ctx, &configMap)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile deletion of snapshot request ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
		return ctrl.Result{}, nil
	}

	// Extract snapshot request details from the ConfigMap and the Secret 🔎
	snapshotRequest, err := snapshotapi.UnmarshalRequest(&configMap)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to unmarshal vcluster snapshot request from ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}

	// Not done? Add the finalizer if it's not already set! 🔒
	if !snapshotRequest.Done() {
		updated, err := c.addFinalizer(ctx, &configMap)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to add vCluster snapshot controller finalizer to the snapshot request ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
		if updated {
			c.eventRecorder.Eventf(
				&configMap,
				nil,
				corev1.EventTypeNormal,
				"Created",
				"CreateSnapShotRequest",
				"Snapshot request %s/%s has been created",
				configMap.Namespace,
				configMap.Name,
			)
			return ctrl.Result{}, nil
		}
	}
	canContinue, err := c.cancelPreviousRequests(ctx, snapshotRequest)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to cancel previous snapshot requests: %w", err)
	}
	if !canContinue {
		return ctrl.Result{
			RequeueAfter: 30 * time.Second,
		}, nil
	}

	// patch snapshot request ConfigMap after the reconciliation
	configMapBeforeChange := client.MergeFrom(configMap.DeepCopy())
	defer func() {
		if retErr != nil {
			// something went wrong, recorde error and update snapshot request phase to Failed
			snapshotRequest.Status.Phase = snapshotapi.RequestPhaseFailed
			snapshotRequest.Status.Error.Message = retErr.Error()
			c.eventRecorder.Eventf(
				&configMap,
				nil,
				corev1.EventTypeWarning,
				"SnapshotRequestFailed",
				"ReconcileSnapShotRequest",
				"Snapshot request %s/%s has failed with error: %v",
				configMap.Namespace,
				configMap.Name, retErr)
		}
		updateErr := c.updateRequest(ctx, configMapBeforeChange, &configMap, *snapshotRequest)
		if updateErr != nil {
			retErr = fmt.Errorf("failed to update snapshot request %s/%s: %w", configMap.Namespace, configMap.Name, updateErr)
		}
		if retErr != nil {
			retErr = errors.Join(retErr, updateErr)
		} else {
			retErr = updateErr
		}
	}()

	switch snapshotRequest.Status.Phase {
	case snapshotapi.RequestPhaseNotStarted:
		err = c.reconcileNewRequest(ctx, &configMap, snapshotRequest)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile new snapshot request %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
	case snapshotapi.RequestPhaseCanceling:
		snapshotRequest.Status.Phase = snapshotRequest.Status.Phase.Next()
		return ctrl.Result{}, nil
	case snapshotapi.RequestPhaseCreatingEtcdBackup:
		requeue, err := c.reconcileCreatingEtcdBackup(ctx, &configMap, snapshotRequest)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile etcd backup creation for snapshot request %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
		if requeue {
			return ctrl.Result{
				RequeueAfter: 10 * time.Second,
			}, nil
		}
	case snapshotapi.RequestPhaseCanceled:
		fallthrough
	case snapshotapi.RequestPhaseDeleted:
		fallthrough
	case snapshotapi.RequestPhasePartiallyFailed:
		fallthrough
	case snapshotapi.RequestPhaseCompleted:
		err = c.reconcileCompletedRequest(ctx, &configMap, snapshotRequest.RequestMetadata)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile completed snapshot request %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
	case snapshotapi.RequestPhaseFailed:
		err = c.reconcileFailedRequest(ctx, &configMap, snapshotRequest.RequestMetadata)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile failed snapshot request %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
	case snapshotapi.RequestPhaseDeleting:
		err = c.reconcileDeleting(ctx, &configMap, snapshotRequest)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile snapshot deletion request %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
	case snapshotapi.RequestPhaseDeletingEtcdBackup:
		requeue, err := c.reconcileDeletingEtcdBackup(ctx, &configMap, snapshotRequest)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile snapshot deletion request %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
		if requeue {
			return ctrl.Result{
				RequeueAfter: 10 * time.Second,
			}, nil
		}
	default:
		return ctrl.Result{}, fmt.Errorf("invalid snapshot request phase %s", snapshotRequest.Status.Phase)
	}

	return ctrl.Result{}, nil
}

func (c *Reconciler) Register() error {
	isSnapshotRequestConfig := predicate.NewPredicateFuncs(func(obj client.Object) bool {
		if obj.GetNamespace() != c.getRequestNamespace() {
			return false
		}

		objLabels := obj.GetLabels()
		if objLabels == nil {
			return false
		}
		_, ok := objLabels[constants.SnapshotRequestLabel]
		return ok
	})

	return ctrl.NewControllerManagedBy(c.snapshotRequestsManager).
		WithOptions(controller.Options{
			CacheSyncTimeout:        constants.DefaultCacheSyncTimeout,
			MaxConcurrentReconciles: 1,
		}).
		Named("snapshot-requests-controller").
		For(&corev1.ConfigMap{}, builder.WithPredicates(isSnapshotRequestConfig)).
		Complete(c)
}

// reconcileNewRequest updates the snapshot request phase to "InProgress".
func (c *Reconciler) reconcileNewRequest(_ context.Context, configMap *corev1.ConfigMap, snapshotRequest *snapshotapi.Request) error {
	snapshotRequest.Status.Phase = snapshotapi.RequestPhaseCreatingEtcdBackup
	c.eventRecorder.Eventf(
		configMap,
		nil,
		corev1.EventTypeNormal,
		"CreatingEtcdBackup",
		"ReconcileSnapShotRequest",
		"Started to create etcd backup for snapshot request %s/%s",
		configMap.Namespace,
		configMap.Name,
	)
	return nil
}

// reconcileCreatingEtcdBackup creates the snapshot, uploads it to the specified storage, and updates
// the snapshot request phase to "Completed".
func (c *Reconciler) reconcileCreatingEtcdBackup(ctx context.Context, configMap *corev1.ConfigMap, snapshotRequest *snapshotapi.Request) (bool, error) {
	// Find snapshot request secret, it contains snapshot options (with the storage credentials) 🪪
	var secret corev1.Secret
	secretObjectKey := client.ObjectKey{
		Namespace: configMap.Namespace,
		Name:      configMap.Name,
	}
	err := c.client().Get(ctx, secretObjectKey, &secret)
	if kerrors.IsNotFound(err) {
		// Too soon? Requeue if this is a recently created snapshot request.
		if time.Since(configMap.CreationTimestamp.Time) < 10*time.Second {
			return true, nil
		}
		return false, fmt.Errorf("can't find snapshot request Secret %s/%s: %w", configMap.Namespace, configMap.Name, err)
	} else if err != nil {
		return false, fmt.Errorf("failed to get snapshot request Secret %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}
	c.logger.Infof("Found snapshot request Secret %s/%s", secret.Namespace, secret.Name)

	// Extract snapshot options from the Secret 🔎
	snapshotOptions, err := snapshotapi.UnmarshalOptions(&secret)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal vcluster snapshot request from ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}
	snapshotRequest.Spec.Options = *snapshotOptions

	// Create and save the snapshot! 💾
	c.logger.Infof("Creating vCluster snapshot in storage type %q", snapshotOptions.Type)
	snapshotClient := &Client{
		Request: snapshotRequest,
		Options: *snapshotOptions,
	}
	if !c.isHostMode {
		configMapsToSkip, secretsToSkip, err := c.getOngoingSnapshotRequestsResourceNames(ctx)
		if err != nil {
			return false, fmt.Errorf("failed to get ongoing snapshot requests resource names: %w", err)
		}
		for _, configMapNamespacedName := range configMapsToSkip {
			snapshotClient.addResourceToSkip(string(corev1.ResourceConfigMaps), configMapNamespacedName.String())
		}
		for _, secretNamespacedName := range secretsToSkip {
			snapshotClient.addResourceToSkip(string(corev1.ResourceSecrets), secretNamespacedName.String())
		}
	}
	err = snapshotClient.Run(ctx, c.vConfig)
	if err != nil {
		return false, fmt.Errorf("failed to run snapshot client: %w", err)
	}
	c.logger.Infof("Created vCluster snapshot in storage type %q", snapshotOptions.Type)

	// All done, now update the snapshot request phase to "Completed"! ✅
	snapshotRequest.Status.Phase = snapshotapi.RequestPhaseCompleted

	if snapshotRequest.Status.Phase == snapshotapi.RequestPhaseCompleted {
		c.eventRecorder.Eventf(
			configMap,
			nil,
			corev1.EventTypeNormal,
			"Completed",
			"ReconcileSnapShotRequest",
			"Snapshot request %s/%s has been completed",
			configMap.Namespace,
			configMap.Name,
		)
	} else {
		c.eventRecorder.Eventf(
			configMap,
			nil,
			corev1.EventTypeNormal,
			"PartiallyFailed",
			"ReconcileSnapShotRequest",
			"Snapshot request %s/%s has partially failed",
			configMap.Namespace,
			configMap.Name,
		)
	}
	return false, nil
}

func (c *Reconciler) updateRequest(ctx context.Context, previousConfigMapState client.Patch, configMap *corev1.ConfigMap, snapshotRequest snapshotapi.Request) error {
	snapshotRequestJSON, err := json.Marshal(snapshotRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot request to JSON: %w", err)
	}
	configMap.Data[snapshotapi.RequestKey] = string(snapshotRequestJSON)

	// patch snapshot request ConfigMap
	err = c.client().Patch(ctx, configMap, previousConfigMapState)
	if err != nil {
		return fmt.Errorf("failed to patch snapshot request ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}
	c.logger.Debugf("Patched snapshot request %s/%s", configMap.Namespace, configMap.Name)
	return nil
}

func (c *Reconciler) getOngoingSnapshotRequestsResourceNames(ctx context.Context) ([]types.NamespacedName, []types.NamespacedName, error) {
	// list options with label selector
	var configMaps corev1.ConfigMapList
	listOptions := &client.ListOptions{
		Namespace: c.getRequestNamespace(),
		LabelSelector: labels.SelectorFromSet(map[string]string{
			constants.SnapshotRequestLabel: "",
		}),
	}
	err := c.client().List(ctx, &configMaps, listOptions)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list snapshot requests: %w", err)
	}

	var ongoingRequestConfigMaps []types.NamespacedName
	for _, configMap := range configMaps.Items {
		snapshotRequest, err := snapshotapi.UnmarshalRequest(&configMap)
		if err != nil {
			c.logger.Errorf("Failed to unmarshal vcluster snapshot request from ConfigMap %s/%s: %v", configMap.Namespace, configMap.Name, err)
		}
		if !snapshotRequest.Done() {
			namespacedName := types.NamespacedName{
				Namespace: configMap.Namespace,
				Name:      configMap.Name,
			}
			ongoingRequestConfigMaps = append(ongoingRequestConfigMaps, namespacedName)
		}
	}

	var ongoingRequestSecrets []types.NamespacedName
	var secrets corev1.SecretList
	err = c.client().List(ctx, &secrets, listOptions)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list snapshot request Secrets: %w", err)
	}
	for _, secret := range secrets.Items {
		namespacedName := types.NamespacedName{
			Namespace: secret.Namespace,
			Name:      secret.Name,
		}
		ongoingRequestSecrets = append(ongoingRequestSecrets, namespacedName)
	}

	return ongoingRequestConfigMaps, ongoingRequestSecrets, nil
}

func (c *Reconciler) cancelPreviousRequests(ctx context.Context, request *snapshotapi.Request) (bool, error) {
	if request.Status.Phase != snapshotapi.RequestPhaseNotStarted {
		// the current request has already started, previous requests should be already canceled
		return true, nil
	}

	var configMaps corev1.ConfigMapList
	listOptions := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			constants.SnapshotRequestLabel: "",
		}),
		Namespace: c.getRequestNamespace(),
	}
	err := c.client().List(ctx, &configMaps, listOptions)
	if err != nil {
		return false, fmt.Errorf("failed to list snapshot request ConfigMaps: %w", err)
	}
	currentRequestCanContinue := true

	for _, configMap := range configMaps.Items {
		otherRequest, err := snapshotapi.UnmarshalRequest(&configMap)
		if err != nil {
			c.logger.Errorf("Failed to unmarshal previous snapshot request from ConfigMap %s/%s: %v", configMap.Namespace, configMap.Name, err)
			continue
		}
		if !request.ShouldCancel(otherRequest) {
			if otherRequest.Status.Phase == snapshotapi.RequestPhaseCanceling {
				// the other request is still being canceled, so this one can't continue
				currentRequestCanContinue = false
			}
			continue
		}

		// cancel the previous request
		otherRequest.Status.Phase = snapshotapi.RequestPhaseCanceling
		oldValue := client.MergeFrom(configMap.DeepCopy())
		err = c.updateRequest(ctx, oldValue, &configMap, *otherRequest)
		if err != nil {
			return false, fmt.Errorf("failed to update snapshot request %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
		// the other request has been just canceled, so this one can't continue yet
		currentRequestCanContinue = false
	}

	return currentRequestCanContinue, nil
}
