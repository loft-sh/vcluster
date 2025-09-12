package snapshot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	ControllerFinalizer = "vcluster.loft.sh/snapshot-controller"
	controllerName      = "vcluster-snapshot-controller"
)

type Reconciler struct {
	vConfig       *config.VirtualClusterConfig
	manager       ctrl.Manager
	logger        loghelper.Logger
	eventRecorder record.EventRecorder
	isHostMode    bool
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

	var manager ctrl.Manager
	if isHostMode {
		logger.Infof("Registering vcluster-snapshot-controller to watch for volume snapshot requests in the host cluster")
		manager = registerContext.HostManager
	} else {
		logger.Infof("Registering vcluster-snapshot-controller to watch for volume snapshot requests in the virtual cluster")
		manager = registerContext.VirtualManager
	}

	return &Reconciler{
		vConfig:       registerContext.Config,
		manager:       manager,
		logger:        logger,
		eventRecorder: manager.GetEventRecorderFor(controllerName),
		isHostMode:    isHostMode,
	}, nil
}

func (c *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, retErr error) {
	c.logger.Infof("Reconciling snapshot request ConfigMap %s", req.NamespacedName)

	var configMap corev1.ConfigMap
	err := c.client().Get(ctx, req.NamespacedName, &configMap)
	if kerrors.IsNotFound(err) {
		c.logger.Infof("Snapshot request ConfigMap %s not found", req.NamespacedName)
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get snapshot request ConfigMap %s/%s: %w", req.Namespace, req.Name, err)
	}
	c.logger.Infof("Found ConfigMap %s/%s with vcluster snapshot request", configMap.Namespace, configMap.Name)

	// Snapshot request ConfigMap deleted -> we've got some cleaning up to do 🧹
	if !configMap.DeletionTimestamp.IsZero() {
		err = c.reconcileDeletedRequest(ctx, &configMap)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile deletion of snapshot request ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
		return ctrl.Result{}, nil
	}

	// Extract snapshot request details from the ConfigMap and the Secret 🔎
	snapshotRequest, err := UnmarshalSnapshotRequest(&configMap)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to unmarshal vcluster snapshot request from ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}

	// Not done? Add finalizer! 🔒
	if !snapshotRequest.Done() {
		updated, err := c.addFinalizer(ctx, &configMap)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to add vCluster snapshot controller finalizer to the snapshot request ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
		if updated {
			return ctrl.Result{}, nil
		}
	}

	defer func() {
		if retErr == nil {
			return
		}
		// something went wrong, recorde error and update snapshot request phase to Failed
		c.eventRecorder.Eventf(&configMap, corev1.EventTypeWarning, "SnapshotRequestFailed", "Snapshot request %s/%s has failed with error: %v", configMap.Namespace, configMap.Name, retErr)
		_, err = c.updateRequestPhase(ctx, &configMap, *snapshotRequest, RequestPhaseFailed)
		if err != nil {
			retErr = fmt.Errorf("failed to update snapshot request phase to %s: %w", RequestPhaseFailed, err)
		}
	}()

	switch snapshotRequest.Status.Phase {
	case "":
		err = c.reconcileNewRequest(ctx, &configMap, snapshotRequest)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile new snapshot request %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
	case RequestPhaseInProgress:
		requeue, err := c.reconcileInProgressRequest(ctx, &configMap, snapshotRequest)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile in-progress snapshot request %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
		if requeue {
			return ctrl.Result{
				RequeueAfter: 10 * time.Second,
			}, nil
		}
	case RequestPhaseCompleted:
		err = c.reconcileCompletedRequest(ctx, &configMap)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile completed snapshot request %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
	case RequestPhaseFailed:
		err = c.reconcileFailedRequest(ctx, &configMap)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile failed snapshot request %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
	default:
		return ctrl.Result{}, fmt.Errorf("unknown snapshot request phase %s", snapshotRequest.Status.Phase)
	}

	return ctrl.Result{}, nil
}

func (c *Reconciler) Register() error {
	isVolumeSnapshotsConfig := predicate.NewPredicateFuncs(func(obj client.Object) bool {
		if obj.GetNamespace() != c.getSnapshotRequestNamespace() {
			return false
		}

		objLabels := obj.GetLabels()
		if objLabels == nil {
			return false
		}
		_, ok := objLabels[RequestLabel]
		return ok
	})

	return ctrl.NewControllerManagedBy(c.manager).
		WithOptions(controller.Options{
			CacheSyncTimeout:        constants.DefaultCacheSyncTimeout,
			MaxConcurrentReconciles: 1,
		}).
		Named("volume-snapshots-controller").
		For(&corev1.ConfigMap{}, builder.WithPredicates(isVolumeSnapshotsConfig)).
		Complete(c)
}

// reconcileNewRequest updates the snapshot request phase to "InProgress".
func (c *Reconciler) reconcileNewRequest(ctx context.Context, configMap *corev1.ConfigMap, snapshotRequest *Request) error {
	c.eventRecorder.Eventf(configMap, corev1.EventTypeNormal, "SnapshotRequestCreated", "Snapshot request %s/%s has been created", configMap.Namespace, configMap.Name)
	_, err := c.updateRequestPhase(ctx, configMap, *snapshotRequest, RequestPhaseInProgress)
	if err != nil {
		return fmt.Errorf("failed to update snapshot request phase to %s: %w", RequestPhaseInProgress, err)
	}
	c.eventRecorder.Eventf(configMap, corev1.EventTypeNormal, "SnapshotRequestInProgress", "Snapshot request %s/%s is in progress", configMap.Namespace, configMap.Name)
	return nil
}

// reconcileInProgressRequest creates the snapshot, uploads it to the specified storage, and updates
// the snapshot request phase to "Completed".
func (c *Reconciler) reconcileInProgressRequest(ctx context.Context, configMap *corev1.ConfigMap, snapshotRequest *Request) (bool, error) {
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
	snapshotOptions, err := UnmarshalSnapshotOptions(&secret)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal vcluster snapshot request from ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}
	snapshotRequest.Spec.Options = *snapshotOptions

	// Create and save the snapshot! 💾
	c.logger.Infof("Creating vCluster snapshot in storage type %q", snapshotOptions.Type)
	snapshotClient := &Client{
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
	err = snapshotClient.Run(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to run snapshot client: %w", err)
	}
	c.logger.Infof("Created vCluster snapshot in storage type %q", snapshotOptions.Type)

	// All done, now update the snapshot request phase to "Completed"! ✅
	_, err = c.updateRequestPhase(ctx, configMap, *snapshotRequest, RequestPhaseCompleted)
	if err != nil {
		return false, fmt.Errorf("failed to update snapshot request phase to %s: %w", RequestPhaseCompleted, err)
	}
	c.eventRecorder.Eventf(configMap, corev1.EventTypeNormal, "SnapshotRequestCompleted", "Snapshot request %s/%s has been completed", configMap.Namespace, configMap.Name)
	return false, nil
}

// reconcileCompletedRequest cleans up the completed snapshot request resources.
func (c *Reconciler) reconcileCompletedRequest(ctx context.Context, configMap *corev1.ConfigMap) error {
	c.logger.Infof("Snapshot request from ConfigMap %s/%s has been completed", configMap.Namespace, configMap.Name)
	err := c.reconcileDoneRequest(ctx, configMap)
	if err != nil {
		return fmt.Errorf("failed to delete snapshot request Secret %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}
	return nil
}

// reconcileFailedRequest cleans up the failed snapshot request resources.
func (c *Reconciler) reconcileFailedRequest(ctx context.Context, configMap *corev1.ConfigMap) error {
	c.logger.Errorf("Snapshot request from ConfigMap %s/%s has failed", configMap.Namespace, configMap.Name)
	err := c.reconcileDoneRequest(ctx, configMap)
	if err != nil {
		return fmt.Errorf("failed to delete snapshot request Secret %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}
	return nil
}

// reconcileDeletedRequest deletes the snapshot request Secret and removes the finalizer from the
// snapshot request ConfigMap.
func (c *Reconciler) reconcileDeletedRequest(ctx context.Context, configMap *corev1.ConfigMap) (retErr error) {
	// snapshot request ConfigMap deleted, so delete Secret as well
	c.logger.Infof("Snapshot request ConfigMap %s/%s deleted", configMap.Namespace, configMap.Name)

	err := c.reconcileDoneRequest(ctx, configMap)
	if err != nil {
		return fmt.Errorf("failed to delete snapshot request Secret %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}
	return nil
}

// reconcileDoneRequest deletes the snapshot request Secret and removes the finalizer from the
// snapshot request ConfigMap.
func (c *Reconciler) reconcileDoneRequest(ctx context.Context, configMap *corev1.ConfigMap) (retErr error) {
	defer func() {
		if retErr != nil {
			// an error occurred, don't remove the finalizer
			return
		}
		err := c.removeFinalizer(ctx, configMap)
		if err != nil {
			retErr = fmt.Errorf("failed to remove vCluster snapshot controller finalizer from the snapshot request ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
	}()

	err := c.deleteSnapshotRequestSecret(ctx, configMap)
	if err != nil {
		return fmt.Errorf("failed to delete snapshot request Secret %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}
	return nil
}

func (c *Reconciler) addFinalizer(ctx context.Context, configMap *corev1.ConfigMap) (bool, error) {
	if controllerutil.ContainsFinalizer(configMap, ControllerFinalizer) {
		return false, nil
	}

	c.logger.Infof(
		"Adding vCluster snapshot controller finalizer %s to the snapshot request ConfigMap %s/%s",
		ControllerFinalizer,
		configMap.Namespace,
		configMap.Name)

	// before the change
	oldConfigMap := client.MergeFrom(configMap.DeepCopy())

	// add the snapshot controller finalizer to the snapshot request ConfigMap
	controllerutil.AddFinalizer(configMap, ControllerFinalizer)

	// patch the object
	err := c.client().Patch(ctx, configMap, oldConfigMap)
	if err != nil {
		return false, fmt.Errorf("failed to patch snapshot request ConfigMap %s/%s finalizers: %w", configMap.Namespace, configMap.Name, err)
	}

	c.logger.Infof(
		"Added vCluster snapshot controller finalizer %s to the snapshot request ConfigMap %s/%s",
		ControllerFinalizer,
		configMap.Namespace,
		configMap.Name)
	return true, nil
}

func (c *Reconciler) removeFinalizer(ctx context.Context, configMap *corev1.ConfigMap) error {
	if !controllerutil.ContainsFinalizer(configMap, ControllerFinalizer) {
		return nil
	}

	c.logger.Infof(
		"Removing vCluster snapshot controller finalizer %s from the snapshot request ConfigMap %s/%s",
		ControllerFinalizer,
		configMap.Namespace,
		configMap.Name)

	// before the change
	oldConfigMap := client.MergeFrom(configMap.DeepCopy())

	// add the snapshot controller finalizer to the snapshot request ConfigMap
	controllerutil.RemoveFinalizer(configMap, ControllerFinalizer)

	// patch the object
	err := c.client().Patch(ctx, configMap, oldConfigMap)
	if err != nil {
		return fmt.Errorf("failed to patch snapshot request ConfigMap %s/%s finalizers: %w", configMap.Namespace, configMap.Name, err)
	}

	c.logger.Infof(
		"Removed vCluster snapshot controller finalizer %s from the snapshot request ConfigMap %s/%s",
		ControllerFinalizer,
		configMap.Namespace,
		configMap.Name)
	return nil
}

func (c *Reconciler) deleteSnapshotRequestSecret(ctx context.Context, configMap *corev1.ConfigMap) error {
	namespace := configMap.Namespace
	name := configMap.Name
	c.logger.Debugf("Deleting snapshot request Secret %s/%s", namespace, name)

	// find snapshot request secret
	var secret corev1.Secret
	secretObjectKey := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}
	err := c.client().Get(ctx, secretObjectKey, &secret)
	if kerrors.IsNotFound(err) {
		c.logger.Debugf("Snapshot request Secret %s/%s aleady deleted", namespace, name)
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get snapshot request Secret %s/%s: %w", namespace, name, err)
	}

	// delete snapshot request secret
	err = c.client().Delete(ctx, &secret)
	if kerrors.IsNotFound(err) {
		c.logger.Debugf("Snapshot request Secret %s/%s aleady deleted", namespace, name)
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to delete snapshot request Secret %s/%s: %w", namespace, name, err)
	}

	c.logger.Debugf("Deleted snapshot request Secret %s/%s", namespace, name)
	c.eventRecorder.Eventf(configMap, corev1.EventTypeNormal, "SnapshotRequestCleanup", "Snapshot request Secret %s/%s has been deleted", configMap.Namespace, configMap.Name)
	return nil
}

func (c *Reconciler) client() client.Client {
	return c.manager.GetClient()
}

func (c *Reconciler) getSnapshotRequestNamespace() string {
	if c.isHostMode {
		return c.vConfig.HostNamespace
	}
	return "kube-system"
}

func (c *Reconciler) updateRequestPhase(ctx context.Context, configMap *corev1.ConfigMap, snapshotRequest Request, phase RequestPhase) (bool, error) {
	if snapshotRequest.Status.Phase == phase {
		return false, nil
	}

	// before the change
	oldConfigMap := client.MergeFrom(configMap.DeepCopy())

	// update phase to InProgress
	snapshotRequest.Status.Phase = phase
	snapshotRequestJSON, err := json.Marshal(snapshotRequest)
	if err != nil {
		return false, fmt.Errorf("failed to marshal snapshot request (with phase updated to %s) to JSON: %w", phase, err)
	}
	configMap.Data[RequestKey] = string(snapshotRequestJSON)

	// patch snapshot request ConfigMap
	err = c.client().Patch(ctx, configMap, oldConfigMap)
	if err != nil {
		return false, fmt.Errorf("failed to patch snapshot request ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}
	c.logger.Infof("Updated snapshot request %s/%s status, set phase to %s", configMap.Namespace, configMap.Name, phase)
	return true, nil
}

func (c *Reconciler) getOngoingSnapshotRequestsResourceNames(ctx context.Context) ([]types.NamespacedName, []types.NamespacedName, error) {
	// list options with label selector
	var configMaps corev1.ConfigMapList
	listOptions := &client.ListOptions{
		Namespace: c.getSnapshotRequestNamespace(),
		LabelSelector: labels.SelectorFromSet(map[string]string{
			RequestLabel: "",
		}),
	}
	err := c.client().List(ctx, &configMaps, listOptions)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list snapshot requests: %w", err)
	}

	var ongoingRequestConfigMaps []types.NamespacedName
	for _, configMap := range configMaps.Items {
		snapshotRequest, err := UnmarshalSnapshotRequest(&configMap)
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
