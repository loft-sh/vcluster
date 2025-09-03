package snapshot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	ControllerFinalizer = "vcluster.loft.sh/snapshot-controller"
)

type Reconciler struct {
	vConfig *config.VirtualClusterConfig
	manager ctrl.Manager
	logger  loghelper.Logger
}

func NewController(registerContext *synccontext.RegisterContext) (*Reconciler, error) {
	logger := loghelper.New("vcluster-snapshot-controller")

	if registerContext == nil {
		return nil, errors.New("register context is nil")
	}
	var manager ctrl.Manager
	if registerContext.Config.PrivateNodes.Enabled {
		logger.Infof("Registering vcluster-snapshot-controller to watch for volume snapshot requests in the virtual cluster")
		manager = registerContext.VirtualManager
	} else {
		logger.Infof("Registering vcluster-snapshot-controller to watch for volume snapshot requests in the host cluster")
		manager = registerContext.HostManager
	}

	return &Reconciler{
		vConfig: registerContext.Config,
		manager: manager,
		logger:  logger,
	}, nil
}

func (c *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, retErr error) {
	c.logger.Infof("Reconciling snapshot request ConfigMap %s", req.NamespacedName)

	var configMap corev1.ConfigMap
	err := c.client().Get(ctx, req.NamespacedName, &configMap)
	if kerrors.IsNotFound(err) {
		c.logger.Infof("snapshot request ConfigMap %s not found", req.NamespacedName)
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get snapshot request ConfigMap %s/%s: %w", req.Namespace, req.Name, err)
	}
	c.logger.Infof("Found ConfigMap %s/%s with vcluster snapshot request", configMap.Namespace, configMap.Name)

	// First reconciliation -> add finalizer 🔒
	if configMap.DeletionTimestamp.IsZero() {
		updated, err := c.addFinalizer(ctx, &configMap)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to add vCluster snapshot controller finalizer to the snapshot request ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
		if updated {
			return ctrl.Result{}, nil
		}
	}

	// Snapshot request ConfigMap deleted -> we've got some cleaning up to do 🧹
	if !configMap.DeletionTimestamp.IsZero() {
		err = c.reconcileDelete(ctx, &configMap)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile deletion of snapshot request ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
		return ctrl.Result{}, nil
	}

	// Find snapshot request secret, it contains snapshot options (with the storage credentials) 🪪
	var secret corev1.Secret
	secretObjectKey := client.ObjectKey{
		Namespace: configMap.Namespace,
		Name:      configMap.Name,
	}
	err = c.client().Get(ctx, secretObjectKey, &secret)
	if kerrors.IsNotFound(err) {
		// Too soon? Requeue if this is a recently created snapshot request.
		if time.Now().Sub(configMap.CreationTimestamp.Time) < 10*time.Second {
			return ctrl.Result{
				RequeueAfter: 10 * time.Second,
			}, nil
		}
		return ctrl.Result{}, fmt.Errorf("can't find snapshot request Secret %s/%s: %w", configMap.Namespace, configMap.Name, err)
	} else if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get snapshot request Secret %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}
	c.logger.Infof("Found snapshot request Secret %s/%s", secret.Namespace, secret.Name)

	// Extract snapshot request details from the ConfigMap and the Secret 🔎
	snapshotRequest, err := UnmarshalSnapshotRequest(&configMap, &secret)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to unmarshal vcluster snapshot request from ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}

	if snapshotRequest.Status.Phase == "" {
		_, err = c.updateRequestPhase(ctx, &configMap, *snapshotRequest, RequestPhaseInProgress)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update snapshot request ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
		return ctrl.Result{}, nil
	} else if snapshotRequest.Status.Phase == RequestPhaseCompleted {
		c.logger.Infof("snapshot request from ConfigMap %s/%s has been completed, deleting snapshot request ConfigMap", configMap.Namespace, configMap.Name)
		err = c.client().Delete(ctx, &configMap)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to delete snapshot request ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
		return ctrl.Result{}, nil
	} else if snapshotRequest.Status.Phase == RequestPhaseFailed {
		c.logger.Errorf("snapshot request from ConfigMap %s/%s has failed, deleting snapshot request ConfigMap", configMap.Namespace, configMap.Name)
		err = c.client().Delete(ctx, &configMap)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to delete snapshot request ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
		return ctrl.Result{}, nil
	}

	// Create and save the snapshot! 💾
	c.logger.Infof("Creating vCluster snapshot in storage type %q", snapshotRequest.Spec.Options.Type)
	snapshotClient := &Client{
		Options: snapshotRequest.Spec.Options,
	}
	err = snapshotClient.Run(ctx)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to run snapshot client: %w", err)
	}
	c.logger.Infof("Created vCluster snapshot in storage type %q", snapshotRequest.Spec.Options.Type)

	// All done! ✅
	_, err = c.updateRequestPhase(ctx, &configMap, *snapshotRequest, RequestPhaseCompleted)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update vcluster snapshot request %s/%s status: %w", configMap.Namespace, configMap.Name, err)
	}
	return ctrl.Result{}, nil
}

func (c *Reconciler) Register() error {
	isVolumeSnapshotsConfig := predicate.NewPredicateFuncs(func(obj client.Object) bool {
		if c.isHostMode() {
			// Host mode with shared nodes - snapshot request configMap must be in the vCluster namespace!
			if obj.GetNamespace() != c.vConfig.HostNamespace {
				return false
			}
		}

		labels := obj.GetLabels()
		if labels == nil {
			return false
		}
		_, ok := labels[requestLabel]
		return ok
	})

	return ctrl.NewControllerManagedBy(c.manager).
		WithOptions(controller.Options{
			CacheSyncTimeout: constants.DefaultCacheSyncTimeout,
		}).
		Named("volume-snapshots-controller").
		For(&corev1.ConfigMap{}, builder.WithPredicates(isVolumeSnapshotsConfig)).
		Complete(c)
}

func (c *Reconciler) reconcileDelete(ctx context.Context, configMap *corev1.ConfigMap) (retErr error) {
	// snapshot request ConfigMap deleted, so delete Secret as well
	c.logger.Infof(
		"snapshot request ConfigMap %s/%s deleted, deleting snapshot request Secret %s/%s",
		configMap.Namespace, configMap.Name,
		configMap.Namespace, configMap.Name)

	defer func() {
		if retErr != nil {
			// an error occurred, don't remove finalizer
			return
		}
		// deletion successfully reconciled, remove the finalizer
		err := c.removeFinalizer(ctx, configMap)
		if err != nil {
			retErr = fmt.Errorf("failed to remove vCluster snapshot controller finalizer from the snapshot request ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
	}()

	// Find snapshot request secret
	var secret corev1.Secret
	secretObjectKey := client.ObjectKey{
		Namespace: configMap.Namespace,
		Name:      configMap.Name,
	}
	err := c.client().Get(ctx, secretObjectKey, &secret)
	if kerrors.IsNotFound(err) {
		c.logger.Infof("snapshot request Secret %s/%s aleady deleted", configMap.Namespace, configMap.Name)
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get snapshot request Secret %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}

	// Delete snapshot request secret
	err = c.client().Delete(ctx, &secret)
	if kerrors.IsNotFound(err) {
		c.logger.Infof("snapshot request Secret %s/%s aleady deleted", secret.Namespace, secret.Name)
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to delete snapshot request Secret %s/%s: %w", secret.Namespace, secret.Name, err)
	}

	c.logger.Infof("deleted snapshot request Secret %s/%s", secret.Namespace, secret.Name)
	return nil
}

func (c *Reconciler) addFinalizer(ctx context.Context, configMap *corev1.ConfigMap) (bool, error) {
	if controllerutil.ContainsFinalizer(configMap, ControllerFinalizer) {
		return false, nil
	}

	c.logger.Infof(
		"adding vCluster snapshot controller finalizer %s to the snapshot request ConfigMap %s/%s",
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
		"added vCluster snapshot controller finalizer %s to the snapshot request ConfigMap %s/%s",
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
		"removing vCluster snapshot controller finalizer %s from the snapshot request ConfigMap %s/%s",
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
		"removed vCluster snapshot controller finalizer %s from the snapshot request ConfigMap %s/%s",
		ControllerFinalizer,
		configMap.Namespace,
		configMap.Name)
	return nil
}

func (c *Reconciler) client() client.Client {
	return c.manager.GetClient()
}

func (c *Reconciler) isHostMode() bool {
	return !c.vConfig.PrivateNodes.Enabled
}

func (c *Reconciler) updateRequestPhase(ctx context.Context, configMap *corev1.ConfigMap, snapshotRequest Request, phase RequestPhase) (bool, error) {
	if snapshotRequest.Status.Phase == phase {
		return false, nil
	}
	// update phase to InProgress
	snapshotRequest.Status.Phase = phase
	updatedConfigMap, _, err := MarshalSnapshotRequest(configMap.Namespace, &snapshotRequest)
	if err != nil {
		return false, fmt.Errorf("failed to marshal snapshot request (with phase updated to %s) to ConfigMap %s/%s: %w", phase, configMap.Namespace, configMap.Name, err)
	}
	// configMap data patch
	configMap.Data = updatedConfigMap.Data
	err = c.client().Update(ctx, configMap)
	if err != nil {
		return false, fmt.Errorf("failed to update snapshot request ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}
	c.logger.Infof("updated snapshot request %s/%s status, set phase to %s", configMap.Namespace, configMap.Name, phase)
	return true, nil
}
