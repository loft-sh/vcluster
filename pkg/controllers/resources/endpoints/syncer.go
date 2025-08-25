package endpoints

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/specialservices"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.Endpoints())
	if err != nil {
		return nil, err
	}

	return &endpointsSyncer{
		//nolint:staticcheck // SA1019: corev1.Endpoints is deprecated, but still required for compatibility
		GenericTranslator: translator.NewGenericTranslator(ctx, "endpoints", &corev1.Endpoints{}, mapper),

		excludedAnnotations: []string{
			"control-plane.alpha.kubernetes.io/leader",
		},
	}, nil
}

type endpointsSyncer struct {
	syncertypes.GenericTranslator

	excludedAnnotations []string
}

var _ syncertypes.OptionsProvider = &endpointsSyncer{}

func (s *endpointsSyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{
		ObjectCaching: true,
	}
}

var _ syncertypes.Syncer = &endpointsSyncer{}

func (s *endpointsSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer(s)
}

var _ syncertypes.ControllerModifier = &endpointsSyncer{}

func (s *endpointsSyncer) ModifyController(_ *synccontext.RegisterContext, bld *builder.Builder) (*builder.Builder, error) {
	klog.Info("Starting to modify the controller to watch for Service changes and reconcile Endpoints")

	// Watch for changes to Services and reconcile Endpoints
	return bld.Watches(&corev1.Service{}, handler.EnqueueRequestsFromMapFunc(func(_ context.Context, obj client.Object) []ctrl.Request {
		service, ok := obj.(*corev1.Service)
		if !ok || service == nil {
			klog.Info("Received an object that is not a Service or is nil, skipping")
			return []ctrl.Request{}
		}

		// Enqueue a request to reconcile the corresponding Endpoints
		return []ctrl.Request{{
			NamespacedName: types.NamespacedName{
				Namespace: service.Namespace,
				Name:      service.Name,
			},
		}}
	})), nil
}

//nolint:staticcheck // SA1019: corev1.Endpoints is deprecated, but still required for compatibility
func (s *endpointsSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*corev1.Endpoints]) (ctrl.Result, error) {
	if event.HostOld != nil {
		return patcher.DeleteVirtualObject(ctx, event.Virtual, event.HostOld, "host object was deleted")
	}

	pObj := s.translate(ctx, event.Virtual)
	err := pro.ApplyPatchesHostObject(ctx, nil, pObj, event.Virtual, ctx.Config.Sync.ToHost.Endpoints.Patches, false)
	if err != nil {
		return ctrl.Result{}, err
	}

	return patcher.CreateHostObject(ctx, event.Virtual, pObj, s.EventRecorder(), false)
}

//nolint:staticcheck // SA1019: corev1.Endpoints is deprecated, but still required for compatibility
func (s *endpointsSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*corev1.Endpoints]) (_ ctrl.Result, retErr error) {
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.ToHost.Endpoints.Patches, false))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = errors.Join(retErr, err)
		}

		if retErr != nil {
			s.EventRecorder().Eventf(event.Virtual, "Warning", "SyncError", "Error syncing: %v", retErr)
		}
	}()

	err = s.translateUpdate(ctx, event.Host, event.Virtual)
	if err != nil {
		return ctrl.Result{}, err
	}

	// bi-directional sync of annotations and labels
	event.Virtual.Annotations, event.Host.Annotations = translate.AnnotationsBidirectionalUpdate(event, s.excludedAnnotations...)
	event.Virtual.Labels, event.Host.Labels = translate.LabelsBidirectionalUpdate(event)

	return ctrl.Result{}, nil
}

//nolint:staticcheck // SA1019: corev1.Endpoints is deprecated, but still required for compatibility
func (s *endpointsSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*corev1.Endpoints]) (_ ctrl.Result, retErr error) {
	// virtual object is not here anymore, so we delete
	return patcher.DeleteHostObject(ctx, event.Host, event.VirtualOld, "virtual object was deleted")
}

var _ syncertypes.Starter = &endpointsSyncer{}

//nolint:staticcheck // SA1019: corev1.Endpoints is deprecated, but still required for compatibility
func (s *endpointsSyncer) ReconcileStart(ctx *synccontext.SyncContext, req ctrl.Request) (bool, error) {
	if req.NamespacedName == specialservices.DefaultKubernetesSvcKey {
		return true, nil
	}
	if specialservices.Default != nil {
		if _, ok := specialservices.Default.SpecialServicesToSync()[req.NamespacedName]; ok {
			return true, nil
		}
	}

	svc := &corev1.Service{}
	err := ctx.VirtualClient.Get(ctx, types.NamespacedName{
		Namespace: req.Namespace,
		Name:      req.Name,
	}, svc)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return true, nil
		}

		return true, err
	} else if svc.Spec.Selector != nil {
		// check if it was a managed endpoints object before and delete it
		endpoints := &corev1.Endpoints{}
		err = ctx.HostClient.Get(ctx, s.VirtualToHost(ctx, req.NamespacedName, nil), endpoints)
		if err != nil {
			if !kerrors.IsNotFound(err) {
				klog.Infof("Error retrieving endpoints: %v", err)
			}

			return true, nil
		}

		// check if endpoints were created by us
		if endpoints.Annotations != nil && endpoints.Annotations[translate.NameAnnotation] != "" {
			// Deleting the endpoints is necessary here as some clusters would not correctly maintain
			// the endpoints if they were managed by us previously and now should be managed by Kubernetes.
			// In the worst case we would end up in a state where we have multiple endpoint slices pointing
			// to the same endpoints resulting in wrong DNS and cluster networking. Hence, deleting the previously
			// managed endpoints signals the Kubernetes controller to recreate the endpoints from the selector.
			klog.Infof("Refresh endpoints in physical cluster because they shouldn't be managed by vcluster anymore")
			err = ctx.HostClient.Delete(ctx, endpoints)
			if err != nil {
				klog.Infof("Error deleting endpoints %s/%s: %v", endpoints.Namespace, endpoints.Name, err)
				return true, err
			}
		}

		return true, nil
	}

	// check if it was a Kubernetes managed endpoints object before and delete it
	endpoints := &corev1.Endpoints{}
	err = ctx.HostClient.Get(ctx, s.VirtualToHost(ctx, req.NamespacedName, nil), endpoints)
	if err == nil && (endpoints.Annotations == nil || endpoints.Annotations[translate.NameAnnotation] == "") {
		klog.Infof("Refresh endpoints in physical cluster because they should be managed by vCluster now")
		err = ctx.HostClient.Delete(ctx, endpoints)
		if err != nil {
			klog.Infof("Error deleting endpoints %s/%s: %v", endpoints.Namespace, endpoints.Name, err)
			return true, err
		}
	}

	return false, nil
}

func (s *endpointsSyncer) ReconcileEnd() {}
