package snapshot

import (
	"context"
	"errors"
	"fmt"
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
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"
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

func (c *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, reterr error) {
	c.logger.Infof("Reconciling vcluster snapshot request ConfigMap %s", req.NamespacedName)

	var configMap corev1.ConfigMap
	err := c.client().Get(ctx, req.NamespacedName, &configMap)
	if kerrors.IsNotFound(err) {
		c.logger.Infof("Can't find vcluster snapshot request ConfigMap %s, requeuing", req.NamespacedName)
		return ctrl.Result{
			RequeueAfter: 10 * time.Second,
		}, nil
	} else if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get ConfigMap %s/%s with vcluster snapshot request: %w", req.Namespace, req.Name, err)
	}
	c.logger.Infof("Found ConfigMap %s/%s with vcluster snapshot request", configMap.Namespace, configMap.Name)

	// Find snapshot request secret
	var secret corev1.Secret
	secretObjectKey := client.ObjectKey{
		Namespace: configMap.Namespace,
		Name:      configMap.Name,
	}
	err = c.client().Get(ctx, secretObjectKey, &secret)
	if kerrors.IsNotFound(err) {
		// requeue if new snapshot request
		if time.Now().Sub(configMap.CreationTimestamp.Time) < 10*time.Second {
			return ctrl.Result{
				RequeueAfter: 10 * time.Second,
			}, nil
		}

		return ctrl.Result{}, fmt.Errorf("can't find Secret %s/%s with vcluster snapshot request options: %w", configMap.Namespace, configMap.Name, err)
	} else if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get Secret %s/%s with vcluster snapshot request: %w", configMap.Namespace, configMap.Name, err)
	}
	c.logger.Infof("Found Secret %s/%s with vcluster snapshot request", secret.Namespace, secret.Name)

	// Handle deletion
	if !configMap.DeletionTimestamp.IsZero() {
		// snapshot request ConfigMap deleted, so delete Secret as well
		c.logger.Infof(
			"snapshot request ConfigMap %s/%s deleted, deleting snapshot request Secret %s/%s",
			configMap.Namespace, configMap.Name,
			secret.Namespace, secret.Name)
		err = c.client().Delete(ctx, &secret)
		if err != nil && !kerrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("failed to delete Secret %s/%s with vcluster snapshot request: %w", secret.Namespace, secret.Name, err)
		}
		c.logger.Infof("deleted snapshot request Secret %s/%s", secret.Namespace, secret.Name)
		return ctrl.Result{}, nil
	}

	snapshotRequest, err := UnmarshalSnapshotRequest(&configMap, &secret)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to unmarshal vcluster snapshot request from ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}

	if snapshotRequest.Status.Phase == "" {
		// update phase to InProgress
		_, err = c.setRequestPhase(ctx, &configMap, *snapshotRequest, RequestPhaseInProgress)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update vcluster snapshot request %s/%s status: %w", configMap.Namespace, configMap.Name, err)
		}
		return ctrl.Result{}, nil
	} else if snapshotRequest.Status.Phase == RequestPhaseCompleted {
		c.logger.Infof("snapshot request from ConfigMap %s/%s has been completed, deleting snapshot request ConfigMap", configMap.Namespace, configMap.Name)
		err = c.client().Delete(ctx, &configMap)
		if err != nil && !kerrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("failed to delete ConfigMap %s/%s with vcluster snapshot request: %w", configMap.Namespace, configMap.Name, err)
		}
		return ctrl.Result{}, nil
	} else if snapshotRequest.Status.Phase == RequestPhaseFailed {
		c.logger.Errorf("snapshot request from ConfigMap %s/%s has failed, deleting snapshot request ConfigMap", configMap.Namespace, configMap.Name)
		err = c.client().Delete(ctx, &configMap)
		if err != nil && !kerrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("failed to delete ConfigMap %s/%s with vcluster snapshot request: %w", configMap.Namespace, configMap.Name, err)
		}
		return ctrl.Result{}, nil
	}

	c.logger.Infof("Creating vCluster snapshot in storage type %q", snapshotRequest.Spec.Options.Type)
	snapshotClient := &Client{
		Options: snapshotRequest.Spec.Options,
	}
	err = snapshotClient.Run(ctx)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to run snapshot client: %w", err)
	}
	c.logger.Infof("Created vCluster snapshot in storage type %q", snapshotRequest.Spec.Options.Type)

	// update phase to Completed
	_, err = c.setRequestPhase(ctx, &configMap, *snapshotRequest, RequestPhaseCompleted)
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

		annotations := obj.GetAnnotations()
		if annotations == nil {
			return false
		}
		_, ok := annotations[requestAnnotation]
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

func (c *Reconciler) client() client.Client {
	return c.manager.GetClient()
}

func (c *Reconciler) isHostMode() bool {
	return !c.vConfig.PrivateNodes.Enabled
}

func (c *Reconciler) setRequestPhase(ctx context.Context, configMap *corev1.ConfigMap, snapshotRequest Request, phase RequestPhase) (bool, error) {
	if snapshotRequest.Status.Phase == phase {
		return false, nil
	}
	// update phase to InProgress
	snapshotRequest.Status.Phase = phase
	updatedConfigMap, _, err := MarshalSnapshotRequest(configMap.Namespace, &snapshotRequest)
	if err != nil {
		return false, fmt.Errorf("failed to marshal vCluster snapshot request (with updated phase) to ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}
	// configMap data patch
	configMap.Data = updatedConfigMap.Data
	err = c.client().Update(ctx, configMap)
	if err != nil {
		return false, fmt.Errorf("failed to update ConfigMap %s/%s with updated vCluster snapshot request: %w", configMap.Namespace, configMap.Name, err)
	}
	c.logger.Infof("updated vCluster snapshot request %s/%s status, set phase to %s", configMap.Namespace, configMap.Name, phase)
	return true, nil
}
