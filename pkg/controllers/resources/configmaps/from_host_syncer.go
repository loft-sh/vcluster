package configmaps

import (
	"context"
	"fmt"
	"time"

	"k8s.io/klog/v2"

	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	toolscache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func NewFromHost(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	configMapTranslator, err := NewConfigMapFromHostTranslate(ctx)
	if err != nil {
		return nil, err
	}

	return &configMapFromHostSyncer{
		GenericTranslator: configMapTranslator,
		skipFuncs:         []skipHostObject{skipKubeRootCaConfigMap},
	}, nil
}

type configMapFromHostSyncer struct {
	syncertypes.GenericTranslator
	skipFuncs []skipHostObject
}

func (s *configMapFromHostSyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{
		UsesCustomPhysicalCache: true,
	}
}

func (s *configMapFromHostSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*corev1.ConfigMap]) (ctrl.Result, error) {
	klog.FromContext(ctx).Info("SyncToHost called")
	return ctrl.Result{}, ctx.VirtualClient.Delete(ctx, event.Virtual)
}

func (s *configMapFromHostSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*corev1.ConfigMap]) (_ ctrl.Result, retErr error) {
	klog.FromContext(ctx).Info("Sync called")

	patchHelper, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.FromHost.ConfigMaps.Patches, false))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}

	defer func() {
		patchHelper.SkipHostPatch()
		if err := patchHelper.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
		if retErr != nil {
			s.EventRecorder().Eventf(event.Virtual, "Warning", "SyncError", "Error syncing: %v", retErr)
		}
	}()

	hostCopy := event.Host.DeepCopy()
	event.Virtual.Annotations = event.Host.Annotations
	event.Virtual.Labels = event.Host.Labels
	event.Virtual.Data = hostCopy.Data

	return ctrl.Result{}, nil
}

func (s *configMapFromHostSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*corev1.ConfigMap]) (ctrl.Result, error) {
	klog.FromContext(ctx).Info("SyncToVirtual called")
	if event.VirtualOld != nil || event.Host.DeletionTimestamp != nil {
		return patcher.DeleteHostObject(ctx, event.Host, event.VirtualOld, "virtual object was deleted")
	}

	vObj := translate.VirtualMetadata(event.Host, s.HostToVirtual(ctx, types.NamespacedName{Name: event.Host.Name, Namespace: event.Host.Namespace}, event.Host))

	err := pro.ApplyPatchesVirtualObject(ctx, nil, vObj, event.Host, ctx.Config.Sync.FromHost.ConfigMaps.Patches, false)
	if err != nil {
		return ctrl.Result{}, err
	}

	// make sure namespace exists
	namespace := &corev1.Namespace{}
	err = ctx.VirtualClient.Get(ctx, client.ObjectKey{Name: vObj.Namespace}, namespace)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return ctrl.Result{Requeue: true},
				ctx.VirtualClient.Create(
					ctx, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{Name: vObj.Namespace},
					},
				)
		}

		return ctrl.Result{}, err
	} else if namespace.DeletionTimestamp != nil {
		// cannot create events in terminating namespaces, requeue to re-create namespaces later
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	return patcher.CreateVirtualObject(ctx, event.Host, vObj, s.EventRecorder(), false)
}

func (s *configMapFromHostSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer(s)
}

var _ syncertypes.Syncer = &configMapFromHostSyncer{}

var _ syncertypes.OptionsProvider = &configMapFromHostSyncer{}

func (s *configMapFromHostSyncer) ModifyController(ctx *synccontext.RegisterContext, b *builder.Builder) (*builder.Builder, error) {
	// the default cache is configured to look at only the target namespaces, create an event source from
	// a cache that watches all namespaces
	hostNamespacesToWatch := getHostNamespacesAndConfig(ctx.Config.Sync.FromHost.ConfigMaps.Selector.Mappings, ctx.Config.ControlPlaneNamespace)

	nsCache, err := cache.New(
		ctx.PhysicalManager.GetConfig(),
		cache.Options{
			Mapper:            ctx.PhysicalManager.GetRESTMapper(),
			DefaultNamespaces: hostNamespacesToWatch,
			DefaultWatchErrorHandler: func(r *toolscache.Reflector, err error) {
				if kerrors.IsForbidden(err) {
					klog.FromContext(ctx).Error(err,
						"trying to watch on a namespace that does not exists / have no permissions. "+
							"This may likely happen in vCluster Role & RoleBinding got deleted from this namespace. "+
							"Please either re-create it or remove the namespace from mappings in the vcluster.yaml")
				} else {
					toolscache.DefaultWatchErrorHandler(r, err)
				}
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create nsCache: %w", err)
	}

	err = ctx.PhysicalManager.Add(nsCache)
	if err != nil {
		return nil, fmt.Errorf("failed to add nsCache to physical manager: %w", err)
	}
	syncContext := ctx.ToSyncContext("configmap from host syncer")

	return b.WatchesRawSource(source.Kind(nsCache, s.Resource(), &handler.Funcs{
		CreateFunc: func(_ context.Context, ce event.TypedCreateEvent[client.Object], rli workqueue.TypedRateLimitingInterface[ctrl.Request]) {
			obj := ce.Object
			s.enqueuePhysical(syncContext, obj, rli)
		},
		UpdateFunc: func(_ context.Context, ue event.TypedUpdateEvent[client.Object], rli workqueue.TypedRateLimitingInterface[ctrl.Request]) {
			obj := ue.ObjectNew
			s.enqueuePhysical(syncContext, obj, rli)
		},
		DeleteFunc: func(_ context.Context, de event.TypedDeleteEvent[client.Object], rli workqueue.TypedRateLimitingInterface[ctrl.Request]) {
			obj := de.Object
			s.enqueuePhysical(syncContext, obj, rli)
		},
		GenericFunc: func(_ context.Context, ge event.TypedGenericEvent[client.Object], rli workqueue.TypedRateLimitingInterface[ctrl.Request]) {
			obj := ge.Object
			s.enqueuePhysical(syncContext, obj, rli)
		},
	})), nil
}

func (s *configMapFromHostSyncer) enqueuePhysical(ctx *synccontext.SyncContext, obj client.Object, q workqueue.TypedRateLimitingInterface[ctrl.Request]) {
	if obj == nil {
		return
	}
	if nn, ok := s.shouldSync(ctx, obj); ok {
		q.Add(reconcile.Request{NamespacedName: nn})
	}
}

func (s *configMapFromHostSyncer) shouldSync(ctx *synccontext.SyncContext, obj client.Object) (types.NamespacedName, bool) {
	hostName, hostNs := obj.GetName(), obj.GetNamespace()
	return matchesHostObject(hostName, hostNs, ctx.Config.Sync.FromHost.ConfigMaps.Selector.Mappings, ctx.Config.ControlPlaneNamespace, s.skipFuncs...)
}
