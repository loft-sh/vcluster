package generic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/k0s"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
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
}

func NewFromHost(ctx *synccontext.RegisterContext, fromHost FromHostSyncer, translator syncertypes.FromConfigTranslator, skipFuncs ...translator.ShouldSkipHostObjectFunc) (syncertypes.Object, error) {
	s := &genericFromHostSyncer{
		FromHostSyncer:       fromHost,
		FromConfigTranslator: translator,
		skipFuncs:            skipFuncs,
	}
	virtualToExclude := s.checkExperimentalDeployConfig(ctx)
	s.virtualObjectsToExclude = virtualToExclude
	return s, nil
}

type genericFromHostSyncer struct {
	syncertypes.FromConfigTranslator
	FromHostSyncer
	skipFuncs               []translator.ShouldSkipHostObjectFunc
	virtualObjectsToExclude map[string]bool
}

func (s *genericFromHostSyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{
		UsesCustomPhysicalCache: true,
	}
}

func (s *genericFromHostSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[client.Object]) (ctrl.Result, error) {
	klog.FromContext(ctx).V(1).Info("SyncToHost called")
	return ctrl.Result{}, ctx.VirtualClient.Delete(ctx, event.Virtual)
}

func (s *genericFromHostSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[client.Object]) (_ ctrl.Result, retErr error) {
	klog.FromContext(ctx).V(1).Info("Sync called")

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

func (s *genericFromHostSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[client.Object]) (ctrl.Result, error) {
	klog.FromContext(ctx).V(1).Info("SyncToVirtual called")
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

func (s *genericFromHostSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer(s)
}

var _ syncertypes.Syncer = &genericFromHostSyncer{}

var _ syncertypes.OptionsProvider = &genericFromHostSyncer{}

func (s *genericFromHostSyncer) ModifyController(ctx *synccontext.RegisterContext, b *builder.Builder) (*builder.Builder, error) {
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
							"Please either re-create it or remove the namespace from mappings in the vcluster.yaml and restart vCluster.")
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

func (s *genericFromHostSyncer) ExcludeVirtual(vObj client.Object) bool {
	_, found := s.virtualObjectsToExclude[vObj.GetNamespace()+"/"+vObj.GetName()]
	if found {
		klog.Infof("excluding virtual object (%s/%s) from %s because it is part of experimental.deploy manifests", vObj.GetName(), vObj.GetNamespace(), s.Name())
	}
	return found
}

func (s *genericFromHostSyncer) ExcludePhysical(_ client.Object) bool {
	return false
}

func (s *genericFromHostSyncer) enqueuePhysical(ctx *synccontext.SyncContext, obj client.Object, q workqueue.TypedRateLimitingInterface[ctrl.Request]) {
	if obj == nil {
		return
	}
	if nn, ok := s.shouldSync(ctx, obj); ok {
		q.Add(reconcile.Request{NamespacedName: nn})
	}
}

func (s *genericFromHostSyncer) shouldSync(_ *synccontext.SyncContext, obj client.Object) (types.NamespacedName, bool) {
	hostName, hostNs := obj.GetName(), obj.GetNamespace()
	if _, ok := obj.GetLabels()[translate.MarkerLabel]; ok {
		// do not sync objects that were synced from virtual to host already
		return types.NamespacedName{}, false
	}
	return s.MatchesHostObject(hostName, hostNs)
}

func (s *genericFromHostSyncer) checkExperimentalDeployConfig(ctx *synccontext.RegisterContext) map[string]bool {
	// (Pawel): this is needed, because when user sets:
	// sync.toHost.(configMaps/secrets).enabled = true
	// sync.toHost.(configMaps/secrets).all = true
	// sync.fromHost.(configMaps/secrets).enabled = true
	// sync.fromHost.(configMaps/secrets).mappings: (mapping from vCluster host namespace to NAMESPACE_A)
	// experimental.deploy.vCluster.manifests: (manifest for (config map / secret) in NAMESPACE_A)
	// then, fromHost syncer will enqueue (config map/ secret) coming from experimental.deploy.vCluster.manifests
	// syncer will try to call SyncToHost, as physical object does not exist
	// fromHostSyncer.SyncToHost will delete virtual object, which is not wanted and expected by user.
	// Therefore, we need to exclude objects from experimental.deploy.vCluster.(manifests/manifestsTemplate)
	// in the genericFromHostSyncer.
	deploy := ctx.Config.Experimental.Deploy
	if deploy.VCluster.Manifests == "" && deploy.VCluster.ManifestsTemplate == "" {
		return nil
	}
	virtualConfigMapsToSkip := make(map[string]bool)

	for _, manifest := range strings.Split(deploy.VCluster.Manifests, "---") {
		configMapKey, found := s.processManifest(manifest)
		if found {
			virtualConfigMapsToSkip[configMapKey] = true
		}
	}

	for _, manifest := range strings.Split(deploy.VCluster.ManifestsTemplate, "---") {
		configMapKey, found := s.processTemplate(manifest, &ctx.Config.Config, ctx.Config.Name, ctx.Config.WorkloadTargetNamespace)
		if found {
			virtualConfigMapsToSkip[configMapKey] = true
		}
	}

	return virtualConfigMapsToSkip
}

func (s *genericFromHostSyncer) processManifest(manifest string) (string, bool) {
	manifest = strings.TrimSpace(manifest)
	if manifest == "" {
		return "", false
	}
	obj := s.Resource()
	err := yaml.Unmarshal([]byte(manifest), obj)
	if err != nil {
		return "", false
	}
	name, ns := obj.GetName(), obj.GetNamespace()
	if ns == "" {
		ns = "default"
	}
	return ns + "/" + name, true
}

func (s *genericFromHostSyncer) processTemplate(manifest string, vConfig *config.Config, name, targetNs string) (string, bool) {
	templatedManifests, err := k0s.ExecTemplate(manifest, name, targetNs, vConfig)
	if err != nil {
		return "", false
	}
	return s.processManifest(string(templatedManifests))
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
