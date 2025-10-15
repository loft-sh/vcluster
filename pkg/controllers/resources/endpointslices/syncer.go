package endpointslices

import (
	"errors"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/specialservices"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.EndpointSlices())
	if err != nil {
		return nil, err
	}

	return &endpointSliceSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "endpointslice", &discoveryv1.EndpointSlice{}, mapper),

		excludedAnnotations: []string{
			"control-plane.alpha.kubernetes.io/leader",
		},
	}, nil
}

type endpointSliceSyncer struct {
	syncertypes.GenericTranslator

	excludedAnnotations []string
}

var _ syncertypes.OptionsProvider = &endpointSliceSyncer{}

func (s *endpointSliceSyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{
		ObjectCaching: true,
	}
}

var _ syncertypes.Syncer = &endpointSliceSyncer{}

func (s *endpointSliceSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer(s)
}

func (s *endpointSliceSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*discoveryv1.EndpointSlice]) (ctrl.Result, error) {
	if event.HostOld != nil {
		return patcher.DeleteVirtualObject(ctx, event.Virtual, event.HostOld, "host object was deleted")
	}

	pObj := s.translate(ctx, event.Virtual)
	err := pro.ApplyPatchesHostObject(ctx, nil, pObj, event.Virtual, ctx.Config.Sync.ToHost.EndpointSlices.Patches, false)
	if err != nil {
		return ctrl.Result{}, err
	}

	return patcher.CreateHostObject(ctx, event.Virtual, pObj, s.EventRecorder(), false)
}

func (s *endpointSliceSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*discoveryv1.EndpointSlice]) (_ ctrl.Result, retErr error) {
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.ToHost.EndpointSlices.Patches, false))
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

func (s *endpointSliceSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*discoveryv1.EndpointSlice]) (_ ctrl.Result, retErr error) {
	// virtual object is not here anymore, so we delete
	return patcher.DeleteHostObject(ctx, event.Host, event.VirtualOld, "virtual object was deleted")
}

var _ syncertypes.Starter = &endpointSliceSyncer{}

func (s *endpointSliceSyncer) ReconcileStart(ctx *synccontext.SyncContext, req ctrl.Request) (bool, error) {
	if req.NamespacedName == specialservices.DefaultKubernetesSvcKey {
		return true, nil
	}
	if specialservices.Default != nil {
		if _, ok := specialservices.Default.SpecialServicesToSync()[req.NamespacedName]; ok {
			return true, nil
		}
	}

	eps := &discoveryv1.EndpointSlice{}
	err := ctx.VirtualClient.Get(ctx, types.NamespacedName{
		Namespace: req.Namespace,
		Name:      req.Name,
	}, eps)
	if err != nil {
		// if endpointSlice is not found on virtual cluster, then remove it from host as well
		if kerrors.IsNotFound(err) {
			hostEps := &discoveryv1.EndpointSlice{}
			err = ctx.HostClient.Get(ctx, s.VirtualToHost(ctx, req.NamespacedName, nil), hostEps)
			if err != nil {
				if !kerrors.IsNotFound(err) {
					return true, fmt.Errorf("error retrieving host endpointSlice: %w", err)
				}
				return true, nil
			}
			err = ctx.HostClient.Delete(ctx, hostEps)
			if err != nil {
				return true, fmt.Errorf("error deleting endpointSlice from host %s: %w", eps.Name, err)
			}
			return true, nil
		}

		return true, fmt.Errorf("error retrieving endpointslice: %w", err)
	}

	epsLabels := eps.GetLabels()
	svcName, ok := epsLabels[translate.K8sServiceNameLabel]
	if !ok {
		return true, fmt.Errorf("unable to retrieve label 'kubernetes.io/service-name'")
	}

	svc := &corev1.Service{}
	err = ctx.VirtualClient.Get(ctx, types.NamespacedName{
		Namespace: req.Namespace,
		Name:      svcName,
	}, svc)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return true, nil
		}

		return true, err
	} else if svc.Spec.Selector != nil {
		// check if it was a managed endpointSlice object before and delete it
		endpointSlice := &discoveryv1.EndpointSlice{}
		err = ctx.HostClient.Get(ctx, s.VirtualToHost(ctx, req.NamespacedName, nil), endpointSlice)
		if err != nil {
			if !kerrors.IsNotFound(err) {
				return true, fmt.Errorf("error retrieving endpointSlice: %w", err)
			}
			return true, nil
		}

		// check if endpoints were created by us
		if endpointSlice.Annotations != nil && endpointSlice.Annotations[translate.NameAnnotation] != "" {
			// Deleting the endpointSlice is necessary here as some clusters would not correctly maintain
			// the endpointSlices if they were managed by us previously and now should be managed by Kubernetes.
			// In the worst case we would end up in a state where we have multiple endpoint slices pointing
			// to the same endpoints resulting in wrong DNS and cluster networking. Hence, deleting the previously
			// managed endpointSlices signals the Kubernetes controller to recreate the endpointSlices from the selector.
			klog.Infof("Refresh endpointSlice in physical cluster because they shouldn't be managed by vcluster anymore")
			err = ctx.HostClient.Delete(ctx, endpointSlice)
			if err != nil {
				return true, fmt.Errorf("error deleting endpointSlice %s/%s: %w", endpointSlice.Namespace, endpointSlice.Name, err)
			}
		}

		return true, nil
	}

	// check if it was a Kubernetes managed endpointSlice object before and delete it
	endpointSlice := &discoveryv1.EndpointSlice{}
	err = ctx.HostClient.Get(ctx, s.VirtualToHost(ctx, req.NamespacedName, nil), endpointSlice)
	if err == nil && (endpointSlice.Annotations == nil || endpointSlice.Annotations[translate.NameAnnotation] == "") {
		klog.Infof("Refresh endpointSlice in physical cluster because they should be managed by vCluster now")
		err = ctx.HostClient.Delete(ctx, endpointSlice)
		if err != nil {
			return true, fmt.Errorf("error deleting endpointSlice %s/%s: %w", endpointSlice.Namespace, endpointSlice.Name, err)
		}
	}

	return false, nil
}

func (s *endpointSliceSyncer) ReconcileEnd() {}
