package endpoints

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/specialservices"
	syncer "github.com/loft-sh/vcluster/pkg/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	return &endpointsSyncer{
		NamespacedTranslator: translator.NewNamespacedTranslator(ctx, "endpoints", &corev1.Endpoints{}),
	}, nil
}

type endpointsSyncer struct {
	translator.NamespacedTranslator
}

func (s *endpointsSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	return s.SyncToHostCreate(ctx, vObj, s.translate(ctx.Context, vObj))
}

func (s *endpointsSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	newEndpoints := s.translateUpdate(ctx.Context, pObj.(*corev1.Endpoints), vObj.(*corev1.Endpoints))
	if newEndpoints != nil {
		translator.PrintChanges(pObj, newEndpoints, ctx.Log)
	}

	return s.SyncToHostUpdate(ctx, vObj, newEndpoints)
}

var _ syncer.Starter = &endpointsSyncer{}

func (s *endpointsSyncer) ReconcileStart(ctx *synccontext.SyncContext, req ctrl.Request) (bool, error) {
	if req.NamespacedName == specialservices.DefaultKubernetesSvcKey {
		return true, nil
	} else if _, ok := specialservices.Default.SpecialServicesToSync()[req.NamespacedName]; ok {
		return true, nil
	}

	svc := &corev1.Service{}
	err := ctx.VirtualClient.Get(ctx.Context, types.NamespacedName{
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
		err := ctx.PhysicalClient.Get(ctx.Context, s.NamespacedTranslator.VirtualToHost(ctx.Context, req.NamespacedName, nil), endpoints)
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
			err = ctx.PhysicalClient.Delete(ctx.Context, endpoints)
			if err != nil {
				klog.Infof("Error deleting endpoints %s/%s: %v", endpoints.Namespace, endpoints.Name, err)
				return true, err
			}
		}

		return true, nil
	}

	// check if it was a Kubernetes managed endpoints object before and delete it
	endpoints := &corev1.Endpoints{}
	err = ctx.PhysicalClient.Get(ctx.Context, s.NamespacedTranslator.VirtualToHost(ctx.Context, req.NamespacedName, nil), endpoints)
	if err == nil && (endpoints.Annotations == nil || endpoints.Annotations[translate.NameAnnotation] == "") {
		klog.Infof("Refresh endpoints in physical cluster because they should be managed by vCluster now")
		err = ctx.PhysicalClient.Delete(ctx.Context, endpoints)
		if err != nil {
			klog.Infof("Error deleting endpoints %s/%s: %v", endpoints.Namespace, endpoints.Name, err)
			return true, err
		}
	}

	return false, nil
}

func (s *endpointsSyncer) ReconcileEnd() {}
