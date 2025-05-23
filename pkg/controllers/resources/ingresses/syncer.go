package ingresses

import (
	"context"
	"fmt"
	"reflect"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/services"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/selector"
	"github.com/loft-sh/vcluster/pkg/util/translate"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	return NewSyncer(ctx)
}

func NewSyncer(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.Ingresses())
	if err != nil {
		return nil, err
	}

	return &ingressSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "ingress", &networkingv1.Ingress{}, mapper),
		Importer:          pro.NewImporter(mapper),

		labelSelector:         ctx.Config.Sync.FromHost.IngressClasses.Selector,
		physicalClusterClient: ctx.PhysicalManager.GetClient(),

		// exclude "field.cattle.io/publicEndpoints" annotation used by Rancher, similar to service syncer
		excludedAnnotations: []string{services.RancherPublicEndpointsAnnotation},
	}, nil
}

type ingressSyncer struct {
	syncertypes.GenericTranslator
	syncertypes.Importer

	labelSelector         config.StandardLabelSelector
	physicalClusterClient client.Client
	excludedAnnotations   []string
}

var _ syncertypes.OptionsProvider = &ingressSyncer{}

func (s *ingressSyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{
		ObjectCaching: true,
	}
}

var _ syncertypes.Syncer = &ingressSyncer{}

func (s *ingressSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer[*networkingv1.Ingress](s)
}

func (s *ingressSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*networkingv1.Ingress]) (ctrl.Result, error) {
	if event.HostOld != nil || event.Virtual.DeletionTimestamp != nil {
		return patcher.DeleteVirtualObject(ctx, event.Virtual, event.HostOld, "host object was deleted")
	}

	pObj, err := s.translate(ctx, event.Virtual)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = pro.ApplyPatchesHostObject(ctx, nil, pObj, event.Virtual, ctx.Config.Sync.ToHost.Ingresses.Patches, false)
	if err != nil {
		return ctrl.Result{}, err
	}

	return patcher.CreateHostObject(ctx, event.Virtual, pObj, s.EventRecorder(), true)
}

func (s *ingressSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*networkingv1.Ingress]) (_ ctrl.Result, retErr error) {
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.ToHost.Ingresses.Patches, false))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}

	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
		if retErr != nil {
			s.EventRecorder().Eventf(event.Virtual, "Warning", "SyncError", "Error syncing: %v", retErr)
		}
	}()

	event.Virtual.Spec.IngressClassName, event.Host.Spec.IngressClassName = patcher.CopyBidirectional(
		event.VirtualOld.Spec.IngressClassName,
		event.Virtual.Spec.IngressClassName,
		event.HostOld.Spec.IngressClassName,
		event.Host.Spec.IngressClassName,
	)
	event.Virtual.Status = event.Host.Status
	s.translateUpdate(ctx, event)
	return ctrl.Result{}, nil
}

func (s *ingressSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*networkingv1.Ingress]) (_ ctrl.Result, retErr error) {
	// virtual object is not here anymore, so we delete
	if event.VirtualOld != nil || translate.ShouldDeleteHostObject(event.Host) {
		return patcher.DeleteHostObject(ctx, event.Host, event.VirtualOld, "virtual object was deleted")
	}

	vIngress := translate.VirtualMetadata(event.Host, s.HostToVirtual(ctx, types.NamespacedName{Name: event.Host.Name, Namespace: event.Host.Namespace}, event.Host), s.excludedAnnotations...)
	err := pro.ApplyPatchesVirtualObject(ctx, nil, vIngress, event.Host, ctx.Config.Sync.ToHost.Ingresses.Patches, false)
	if err != nil {
		return ctrl.Result{}, err
	}

	return patcher.CreateVirtualObject(ctx, event.Host, vIngress, s.EventRecorder(), true)
}

func (s *ingressSyncer) ExcludeVirtual(obj client.Object) bool {
	ingress, ok := obj.(*networkingv1.Ingress)
	if !ok ||
		((ingress.Spec.IngressClassName == nil || *ingress.Spec.IngressClassName == "") && selector.IsLabelSelectorEmpty(s.labelSelector)) {
		return false
	}

	ingressClass := &networkingv1.IngressClass{}
	err := s.physicalClusterClient.Get(context.Background(), types.NamespacedName{Name: *ingress.Spec.IngressClassName}, ingressClass)
	if err != nil {
		klog.FromContext(context.Background()).Info(
			fmt.Sprintf("Warning: Ingress %q will not be synced to host cluster, because IngressClass %q couldn't be found: %v", ingress.Name, *ingress.Spec.IngressClassName, err))
		return true
	}

	exclude := !selector.StandardLabelSelectorMatches(ingressClass, s.labelSelector)
	if exclude {
		klog.FromContext(context.Background()).Info(
			fmt.Sprintf("Warning: Ingress %q will not be synced to host cluster, because IngressClass %q does NOT match the label selector in the 'sync.fromHost.ingressClasses' configuration", ingress.Name, *ingress.Spec.IngressClassName))
	}
	return exclude
}

func (s *ingressSyncer) ExcludePhysical(_ client.Object) bool {
	return false
}

func (s *ingressSyncer) ModifyController(registerCxt *synccontext.RegisterContext, builder *builder.Builder) (*builder.Builder, error) {
	loggerDebug := func(verb, objectName string) {
		klog.FromContext(registerCxt.Context).V(1).Info(
			fmt.Sprintf("%s triggers requeue of ingresses related with ingressClass %q", verb, objectName))
	}
	eventHandler := handler.Funcs{
		CreateFunc: func(_ context.Context, e event.CreateEvent, q workqueue.TypedRateLimitingInterface[ctrl.Request]) {
			loggerDebug("creation", e.Object.GetName())
			requeueRelatedIngresses(registerCxt, nil, e.Object, q)
		},
		UpdateFunc: func(_ context.Context, e event.UpdateEvent, q workqueue.TypedRateLimitingInterface[ctrl.Request]) {
			loggerDebug("update", e.ObjectNew.GetName())
			requeueRelatedIngresses(registerCxt, e.ObjectOld, e.ObjectNew, q)
		},
		DeleteFunc: func(_ context.Context, e event.DeleteEvent, q workqueue.TypedRateLimitingInterface[ctrl.Request]) {
			loggerDebug("delete", e.Object.GetName())
			requeueRelatedIngresses(registerCxt, e.Object, nil, q)
		},
	}

	return builder.Watches(&networkingv1.IngressClass{}, eventHandler), nil
}

func requeueRelatedIngresses(registerCxt *synccontext.RegisterContext, oldObj, newObj client.Object, q workqueue.TypedRateLimitingInterface[ctrl.Request]) {
	if newObj != nil && oldObj != nil && reflect.DeepEqual(newObj.GetLabels(), oldObj.GetLabels()) { // Update with no change in labels
		return
	}
	var ingressClassName string
	if newObj != nil { // Create || Update
		ingressClassName = newObj.GetName()
	}
	if oldObj != nil && newObj == nil { // Delete
		ingressClassName = oldObj.GetName()
	}

	ingresses := &networkingv1.IngressList{}
	if err := registerCxt.VirtualManager.GetClient().List(registerCxt.Context, ingresses); err != nil {
		return
	}

	for _, ingress := range ingresses.Items {
		if ingress.Spec.IngressClassName == nil || *ingress.Spec.IngressClassName != ingressClassName {
			continue
		}
		klog.FromContext(registerCxt.Context).V(1).Info("ingressClass watcher requeue Ingress", "ingressClassName", ingressClassName, "ingress", ingress.Name, "namespace", ingress.Namespace)
		q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
			Name:      ingress.GetName(),
			Namespace: ingress.GetNamespace(),
		}})
	}
}
