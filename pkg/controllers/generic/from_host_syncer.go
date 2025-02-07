package generic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/pkg/constants"

	"github.com/loft-sh/vcluster/pkg/syncer/translator"

	"github.com/loft-sh/vcluster/config"
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
	"k8s.io/klog/v2"

	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type FromHostSyncer interface {
	SyncToHost(vOjb, pObj client.Object)
	GetProPatches(ctx *synccontext.SyncContext) []config.TranslatePatch
	GetMappings(ctx *synccontext.SyncContext) map[string]string
	syncertypes.ObjectExcluder
}

func NewFromHost(_ *synccontext.RegisterContext, fromHost FromHostSyncer, translator syncertypes.FromConfigTranslator, skipFuncs ...translator.ShouldSkipHostObjectFunc) (syncertypes.Object, error) {
	return &configMapFromHostSyncer{
		FromHostSyncer:       fromHost,
		FromConfigTranslator: translator,
		skipFuncs:            skipFuncs,
	}, nil
}

type configMapFromHostSyncer struct {
	syncertypes.FromConfigTranslator
	FromHostSyncer
	skipFuncs []translator.ShouldSkipHostObjectFunc
}

func (s *configMapFromHostSyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{
		UsesCustomPhysicalCache: true,
	}
}

func (s *configMapFromHostSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[client.Object]) (ctrl.Result, error) {
	klog.FromContext(ctx).Info("SyncToHost called")
	return ctrl.Result{}, ctx.VirtualClient.Delete(ctx, event.Virtual)
}

func (s *configMapFromHostSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[client.Object]) (_ ctrl.Result, retErr error) {
	klog.FromContext(ctx).Info("Sync called")

	patchHelper, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(s.GetProPatches(ctx), false))
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

	s.FromHostSyncer.SyncToHost(event.Virtual, event.Host)

	return ctrl.Result{}, nil
}

func (s *configMapFromHostSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[client.Object]) (ctrl.Result, error) {
	klog.FromContext(ctx).Info("SyncToVirtual called")
	if event.VirtualOld != nil || event.Host.GetDeletionTimestamp() != nil {
		return patcher.DeleteHostObject(ctx, event.Host, event.VirtualOld, "virtual object was deleted")
	}

	vObj := translate.VirtualMetadata(event.Host, s.HostToVirtual(ctx, types.NamespacedName{Name: event.Host.GetName(), Namespace: event.Host.GetNamespace()}, event.Host))

	err := pro.ApplyPatchesVirtualObject(ctx, nil, vObj, event.Host, s.GetProPatches(ctx), false)
	if err != nil {
		return ctrl.Result{}, err
	}

	// make sure namespace exists
	namespace := &corev1.Namespace{}
	err = ctx.VirtualClient.Get(ctx, client.ObjectKey{Name: vObj.GetNamespace()}, namespace)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return ctrl.Result{Requeue: true},
				ctx.VirtualClient.Create(
					ctx, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{Name: vObj.GetNamespace()},
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
	hostNamespacesToWatch := getHostNamespacesAndConfig(s.GetMappings(ctx.ToSyncContext("configmap-from-host-syncer")), ctx.Config.ControlPlaneNamespace)

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
	syncContext := ctx.ToSyncContext("from host syncer")

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

func (s *configMapFromHostSyncer) shouldSync(_ *synccontext.SyncContext, obj client.Object) (types.NamespacedName, bool) {
	hostName, hostNs := obj.GetName(), obj.GetNamespace()
	if _, ok := obj.GetLabels()[translate.MarkerLabel]; ok {
		// do not sync objects that were synced from virtual to host already
		return types.NamespacedName{}, false
	}
	return s.MatchesHostObject(hostName, hostNs)
}

func parseHostNamespacesFromMappings(mappings map[string]string, vClusterNs string) []string {
	ret := make([]string, 0)
	for host := range mappings {
		if host == constants.VClusterNamespaceInHostMappingSpecialCharacter {
			ret = append(ret, vClusterNs)
		}
		parts := strings.Split(host, "/")
		if len(parts) != 2 {
			continue
		}
		hostNs := parts[0]
		ret = append(ret, hostNs)
	}
	return ret
}

func getHostNamespacesAndConfig(mappings map[string]string, vClusterNs string) map[string]cache.Config {
	namespaces := parseHostNamespacesFromMappings(mappings, vClusterNs)
	ret := make(map[string]cache.Config, len(namespaces))
	for _, ns := range namespaces {
		ret[ns] = cache.Config{}
	}
	return ret
}
