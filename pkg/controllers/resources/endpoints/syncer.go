package endpoints

import (
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
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

func (s *endpointsSyncer) SyncDown(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	return s.SyncDownCreate(ctx, vObj, s.translate(vObj))
}

func (s *endpointsSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	newEndpoints := s.translateUpdate(pObj.(*corev1.Endpoints), vObj.(*corev1.Endpoints))
	if newEndpoints != nil {
		translator.PrintChanges(pObj, newEndpoints, ctx.Log)
	}

	return s.SyncDownUpdate(ctx, vObj, newEndpoints)
}

var _ syncer.Starter = &endpointsSyncer{}

func (s *endpointsSyncer) ReconcileStart(ctx *synccontext.SyncContext, req ctrl.Request) (bool, error) {
	if req.Namespace == "default" && req.Name == "kubernetes" {
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
		err := ctx.PhysicalClient.Get(ctx.Context, s.NamespacedTranslator.VirtualToPhysical(req.NamespacedName, nil), endpoints)
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
			// to the same endpoints resulting in wrong DNS and cluster networking. Hence deleting the previously
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

	return false, nil
}

func (s *endpointsSyncer) ReconcileEnd() {}
