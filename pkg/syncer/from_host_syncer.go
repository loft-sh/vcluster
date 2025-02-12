package syncer

import (
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"

	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FromHostSyncer interface {
	SyncToHost(vOjb, pObj client.Object)
	GetProPatches(ctx *synccontext.SyncContext) []config.TranslatePatch
	GetMappings(ctx *synccontext.SyncContext) map[string]string
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
		UsesCustomPhysicalCache: true,
		ObjectCaching:           true,
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

	patchHelper, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(s.GetProPatches(ctx), false), patcher.SkipHostPatch())
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

	s.FromHostSyncer.SyncToHost(event.Virtual, event.Host)

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

	err = pro.ApplyPatchesVirtualObject(ctx, nil, vObj, event.Host, s.GetProPatches(ctx), false)
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

func (s *genericFromHostSyncer) ExcludeVirtual(_ client.Object) bool {
	return false
}

func (s *genericFromHostSyncer) ExcludePhysical(obj client.Object) bool {
	_, ok := obj.GetLabels()[translate.MarkerLabel]
	return ok
}
