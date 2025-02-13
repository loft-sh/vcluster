package syncer

import (
	"fmt"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/scheme"

	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"sigs.k8s.io/controller-runtime/pkg/cache"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"

	toolscache "k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"

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
	if event.VirtualOld != nil || event.Host.GetDeletionTimestamp() != nil {
		return patcher.DeleteHostObject(ctx, event.Host, event.VirtualOld, "virtual object was deleted")
	}

	vObj := translate.VirtualMetadata(event.Host, s.HostToVirtual(ctx, types.NamespacedName{Name: event.Host.GetName(), Namespace: event.Host.GetNamespace()}, event.Host))

	// make sure namespace exists
	namespace := &corev1.Namespace{}
	err := ctx.VirtualClient.Get(ctx, client.ObjectKey{Name: vObj.GetNamespace()}, namespace)
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
	multiNsCacheConfig, createCustomManager := getLocalCacheOptions(mappings, ctx.Config.ControlPlaneNamespace)
	if !createCustomManager {
		return ctx, nil
	}

	logNs := make([]string, 0, len(multiNsCacheConfig.DefaultNamespaces))
	for k := range multiNsCacheConfig.DefaultNamespaces {
		logNs = append(logNs, k)
	}
	klog.FromContext(ctx).Info("Setting up custom physical multi-namespace manager for", "namespaces", logNs, "syncer", s.Name())
	localMultiNamespaceManager, err := ctrl.NewManager(ctx.Config.WorkloadConfig, ctrl.Options{
		Scheme: scheme.Scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
		PprofBindAddress: "0",
		LeaderElection:   false,
		NewClient:        pro.NewVirtualClient(ctx.Config),
		WebhookServer:    nil,
		Cache: cache.Options{
			Mapper:            ctx.PhysicalManager.GetRESTMapper(),
			DefaultNamespaces: multiNsCacheConfig.DefaultNamespaces,
			DefaultWatchErrorHandler: func(r *toolscache.Reflector, err error) {
				if kerrors.IsForbidden(err) {
					klog.FromContext(ctx).Error(err,
						"trying to watch on a namespace that does not exists / have no permissions. "+
							"Please either re-create it or remove the namespace from mappings in the vcluster.yaml and restart vCluster.")
				} else {
					toolscache.DefaultWatchErrorHandler(r, err)
				}
			},
		},
	})
	if err != nil {
		return ctx, fmt.Errorf("unable to create custom physical manager for syncer %s: %w", s.Name(), err)
	}

	go func() {
		err := localMultiNamespaceManager.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()

	if synced := localMultiNamespaceManager.GetCache().WaitForCacheSync(ctx.Context); !synced {
		klog.FromContext(ctx).Error(err, "cache was not synced")
		return ctx, fmt.Errorf("cache was not synced for custom physical manager for %s syncer", s.Name())
	}
	newCtx := *ctx
	newCtx.PhysicalManager = localMultiNamespaceManager
	return &newCtx, nil
}

func getLocalCacheOptions(mappings map[string]string, vClusterNamespace string) (cache.Options, bool) {
	defaultNamespaces := make(map[string]cache.Config)
	namespaces := parseHostNamespacesFromMappings(mappings, vClusterNamespace)
	if len(namespaces) == 1 {
		for _, k := range namespaces {
			if k == vClusterNamespace {
				// then there is no need to create custom manager
				return cache.Options{}, false
			}
		}
	}
	for _, ns := range namespaces {
		defaultNamespaces[ns] = cache.Config{}
	}
	return cache.Options{DefaultNamespaces: defaultNamespaces}, true
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
