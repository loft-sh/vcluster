package syncer

import (
	"context"
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/config"
	vclusterconfig "github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/blockingcacheclient"
	"github.com/loft-sh/vcluster/pkg/util/pluginhookclient"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	toolscache "k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

// FromHostSyncer encapsulates kind specific actions that need to happen in the from host sync.
type FromHostSyncer interface {
	// CopyHostObjectToVirtual takes virtual and physical object and copies all needed fields from physical to virtual object.
	// E.g. for Secrets and ConfigMaps, it will be labels, annotations and .data
	CopyHostObjectToVirtual(vObj, pObj client.Object)
	// GetProPatches returns pro patches from config for given kind.
	GetProPatches(cfg config.Config) []config.TranslatePatch
	// GetMappings returns mappings from config for given kind.
	GetMappings(cfg config.Config) map[string]string
}

func NewFromHost(_ *synccontext.RegisterContext, fromHost FromHostSyncer, translator syncertypes.GenericTranslator, skipFuncs ...translator.ShouldSkipHostObjectFunc) (syncertypes.Object, error) {
	s := &genericFromHostSyncer{
		FromHostSyncer:    fromHost,
		GenericTranslator: translator,
		skipFuncs:         skipFuncs,
	}
	return s, nil
}

type genericFromHostSyncer struct {
	syncertypes.GenericTranslator
	FromHostSyncer
	skipFuncs []translator.ShouldSkipHostObjectFunc
}

func (s *genericFromHostSyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{
		ObjectCaching: true,
	}
}

func (s *genericFromHostSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[client.Object]) (ctrl.Result, error) {
	if event.HostOld == nil {
		return ctrl.Result{}, nil
	}
	klog.FromContext(ctx).V(1).Info("SyncToHost called")
	return ctrl.Result{}, ctx.VirtualClient.Delete(ctx, event.Virtual)
}

func (s *genericFromHostSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[client.Object]) (_ ctrl.Result, retErr error) {
	klog.FromContext(ctx).V(1).Info("Sync called")

	patchHelper, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(s.GetProPatches(ctx.Config.Config), false), patcher.SkipHostPatch())
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}

	defer func() {
		if err := patchHelper.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
		if retErr != nil {
			s.EventRecorder().Eventf(event.Virtual, "Warning", "SyncError", "Error syncing: %v", retErr)
		}
	}()

	s.FromHostSyncer.CopyHostObjectToVirtual(event.Virtual, event.Host)

	return ctrl.Result{}, nil
}

func (s *genericFromHostSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[client.Object]) (ctrl.Result, error) {
	klog.FromContext(ctx).V(1).Info("SyncToVirtual called")
	if event.VirtualOld != nil && event.Host.GetDeletionTimestamp() != nil {
		return patcher.DeleteVirtualObject(ctx, event.VirtualOld, event.Host, "host object was deleted")
	}

	vObj := translate.VirtualMetadata(event.Host, s.HostToVirtual(ctx, types.NamespacedName{Name: event.Host.GetName(), Namespace: event.Host.GetNamespace()}, event.Host))

	// make sure namespace exists
	namespace := &corev1.Namespace{}
	err := ctx.VirtualClient.Get(ctx, client.ObjectKey{Name: vObj.GetNamespace()}, namespace)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return ctrl.Result{Requeue: true},
				client.IgnoreAlreadyExists(ctx.VirtualClient.Create(
					ctx, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{Name: vObj.GetNamespace()},
					},
				))
		}
		return ctrl.Result{}, err
	} else if namespace.DeletionTimestamp != nil {
		// cannot create events in terminating namespaces, requeue to re-create namespaces later
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	err = pro.ApplyPatchesVirtualObject(ctx, nil, vObj, event.Host, s.GetProPatches(ctx.Config.Config), false)
	if err != nil {
		return ctrl.Result{}, err
	}

	return patcher.CreateVirtualObject(ctx, event.Host, vObj, s.EventRecorder(), false)
}

func (s *genericFromHostSyncer) Syncer() syncertypes.Sync[client.Object] {
	return ToGenericSyncer(s)
}

var _ syncertypes.Syncer = &genericFromHostSyncer{}

var _ syncertypes.OptionsProvider = &genericFromHostSyncer{}

var _ syncertypes.ManagerProvider = &genericFromHostSyncer{}

func (s *genericFromHostSyncer) ExcludeVirtual(_ client.Object) bool {
	return false
}

func (s *genericFromHostSyncer) ExcludePhysical(obj client.Object) bool {
	_, ok := obj.GetLabels()[translate.MarkerLabel]
	return ok
}

func (s *genericFromHostSyncer) ConfigureAndStartManager(ctx *synccontext.RegisterContext) (*synccontext.RegisterContext, error) {
	mappings := s.GetMappings(ctx.Config.Config)
	localMultiNamespaceManager, skipCustomManagerCreation, err := ConfigureNewLocalManager(ctx, mappings, s.Name())
	if err != nil {
		return nil, err
	}
	if skipCustomManagerCreation {
		return ctx, nil
	}
	newCtx := *ctx

	go func() {
		err := localMultiNamespaceManager.Start(newCtx)
		if err != nil {
			panic(err)
		}
	}()

	if synced := localMultiNamespaceManager.GetCache().WaitForCacheSync(newCtx); !synced {
		return nil, fmt.Errorf("cache was not synced for custom physical manager for %s syncer", s.Name())
	}

	newCtx.HostManager = localMultiNamespaceManager
	return &newCtx, nil
}

func ConfigureNewLocalManager(ctx *synccontext.RegisterContext, mappings map[string]string, syncerName string) (ctrl.Manager, bool, error) {
	multiNsCacheConfig, customManagerNeeded := vclusterconfig.GetLocalCacheOptionsFromConfigMappings(mappings, ctx.Config.HostNamespace)
	if !customManagerNeeded {
		return nil, true, nil
	}
	logNs := make([]string, 0, len(multiNsCacheConfig.DefaultNamespaces))
	for k := range multiNsCacheConfig.DefaultNamespaces {
		logNs = append(logNs, k)
	}
	klog.FromContext(ctx).Info("Setting up custom physical multi-namespace manager for", "namespaces", logNs, "syncer", syncerName)
	localMultiNamespaceManager, err := ctrl.NewManager(ctx.Config.HostConfig, GetOptionsForMultiNamespaceManager(ctx, multiNsCacheConfig))
	if err != nil {
		return nil, false, fmt.Errorf("unable to create custom physical manager for syncer %s: %w", syncerName, err)
	}
	return localMultiNamespaceManager, false, nil
}

func GetOptionsForMultiNamespaceManager(ctx *synccontext.RegisterContext, options cache.Options) ctrl.Options {
	return ctrl.Options{
		Scheme: scheme.Scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
		PprofBindAddress: "0",
		LeaderElection:   false,
		NewClient:        pluginhookclient.NewVirtualPluginClientFactory(blockingcacheclient.NewCacheClient),
		WebhookServer:    nil,
		Cache: cache.Options{
			Mapper:                   ctx.HostManager.GetRESTMapper(),
			DefaultNamespaces:        options.DefaultNamespaces,
			DefaultWatchErrorHandler: additionalPermissionMissingHandler(),
		},
	}
}

func additionalPermissionMissingHandler() toolscache.WatchErrorHandlerWithContext {
	return func(ctx context.Context, r *toolscache.Reflector, err error) {
		if kerrors.IsForbidden(err) {
			klog.FromContext(ctx).Error(err,
				"trying to watch on a namespace that does not exists / have no permissions. "+
					"Please either re-create it or remove the namespace from mappings in the vcluster.yaml and restart vCluster.")
		} else {
			toolscache.DefaultWatchErrorHandler(ctx, r, err)
		}
	}
}
