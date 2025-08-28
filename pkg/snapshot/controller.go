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

func (c *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	c.logger.Infof("Reconciling vcluster snapshot request %s", req.NamespacedName)
	var configMap corev1.ConfigMap
	err := c.client().Get(ctx, req.NamespacedName, &configMap)
	if kerrors.IsNotFound(err) {
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get ConfigMap %s/%s with vcluster snapshot request: %w", req.Namespace, req.Name, err)
	}
	c.logger.Debugf("Found ConfigMap %s/%s with vcluster snapshot request", configMap.Namespace, configMap.Name)

	snapshotRequest, err := UnmarshalSnapshotRequest(&configMap)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to unmarshal vcluster snapshot request from ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}

	c.logger.Debugf("Snapshot request %v", snapshotRequest)
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
		_, ok := annotations[snapshotRequestAnnotation]
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
