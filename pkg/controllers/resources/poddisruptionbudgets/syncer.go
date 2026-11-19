package poddisruptionbudgets

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := generic.NewMapper(ctx, &policyv1.PodDisruptionBudget{}, translate.Default.HostName)
	if err != nil {
		return nil, err
	}

	return &pdbSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "podDisruptionBudget", &policyv1.PodDisruptionBudget{}, mapper),
	}, nil
}

type pdbSyncer struct {
	syncertypes.GenericTranslator
}

var _ syncertypes.OptionsProvider = &pdbSyncer{}

func (s *pdbSyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{
		ObjectCaching: true,
	}
}

var _ syncertypes.Syncer = &pdbSyncer{}

func (s *pdbSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer(s)
}

var _ syncertypes.ControllerModifier = &pdbSyncer{}

func (s *pdbSyncer) ModifyController(ctx *synccontext.RegisterContext, bld *builder.Builder) (*builder.Builder, error) {
	virtualClient := ctx.VirtualManager.GetClient()
	eventHandler := handler.Funcs{
		CreateFunc: func(c context.Context, e event.CreateEvent, q workqueue.TypedRateLimitingInterface[ctrl.Request]) {
			for _, req := range mapWorkloadToPDBs(c, virtualClient, e.Object) {
				q.Add(req)
			}
		},
		UpdateFunc: func(c context.Context, e event.UpdateEvent, q workqueue.TypedRateLimitingInterface[ctrl.Request]) {
			for _, req := range mapWorkloadUpdateToPDBs(c, virtualClient, e.ObjectOld, e.ObjectNew) {
				q.Add(req)
			}
		},
		DeleteFunc: func(c context.Context, e event.DeleteEvent, q workqueue.TypedRateLimitingInterface[ctrl.Request]) {
			for _, req := range mapWorkloadToPDBs(c, virtualClient, e.Object) {
				q.Add(req)
			}
		},
		GenericFunc: func(c context.Context, e event.GenericEvent, q workqueue.TypedRateLimitingInterface[ctrl.Request]) {
			for _, req := range mapWorkloadToPDBs(c, virtualClient, e.Object) {
				q.Add(req)
			}
		},
	}

	return bld.
		Watches(&appsv1.Deployment{}, eventHandler, builder.WithPredicates(workloadPredicate())).
		Watches(&appsv1.StatefulSet{}, eventHandler, builder.WithPredicates(workloadPredicate())).
		Watches(&appsv1.ReplicaSet{}, eventHandler, builder.WithPredicates(workloadPredicate())).
		Watches(&corev1.Pod{}, eventHandler, builder.WithPredicates(podPredicate())), nil
}

func workloadPredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.ObjectOld == nil || e.ObjectNew == nil {
				return false
			}
			if !equality.Semantic.DeepEqual(e.ObjectOld.GetDeletionTimestamp(), e.ObjectNew.GetDeletionTimestamp()) {
				return true
			}
			switch oldObj := e.ObjectOld.(type) {
			case *appsv1.Deployment:
				newObj, ok := e.ObjectNew.(*appsv1.Deployment)
				if !ok {
					return false
				}
				return !equality.Semantic.DeepEqual(oldObj.Spec.Replicas, newObj.Spec.Replicas) ||
					!equality.Semantic.DeepEqual(oldObj.Spec.Template.Labels, newObj.Spec.Template.Labels)
			case *appsv1.StatefulSet:
				newObj, ok := e.ObjectNew.(*appsv1.StatefulSet)
				if !ok {
					return false
				}
				return !equality.Semantic.DeepEqual(oldObj.Spec.Replicas, newObj.Spec.Replicas) ||
					!equality.Semantic.DeepEqual(oldObj.Spec.Template.Labels, newObj.Spec.Template.Labels)
			case *appsv1.ReplicaSet:
				newObj, ok := e.ObjectNew.(*appsv1.ReplicaSet)
				if !ok {
					return false
				}
				return !equality.Semantic.DeepEqual(oldObj.Spec.Replicas, newObj.Spec.Replicas) ||
					!equality.Semantic.DeepEqual(oldObj.Spec.Template.Labels, newObj.Spec.Template.Labels)
			default:
				return false
			}
		},
	}
}

func podPredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.ObjectOld == nil || e.ObjectNew == nil {
				return false
			}
			oldPod, ok1 := e.ObjectOld.(*corev1.Pod)
			newPod, ok2 := e.ObjectNew.(*corev1.Pod)
			if !ok1 || !ok2 {
				return false
			}
			if !equality.Semantic.DeepEqual(oldPod.Labels, newPod.Labels) {
				return true
			}
			if !equality.Semantic.DeepEqual(oldPod.OwnerReferences, newPod.OwnerReferences) {
				return true
			}
			if !equality.Semantic.DeepEqual(oldPod.DeletionTimestamp, newPod.DeletionTimestamp) {
				return true
			}
			if oldPod.Status.Phase != newPod.Status.Phase &&
				(oldPod.Status.Phase == corev1.PodSucceeded || oldPod.Status.Phase == corev1.PodFailed ||
					newPod.Status.Phase == corev1.PodSucceeded || newPod.Status.Phase == corev1.PodFailed) {
				return true
			}
			return false
		},
	}
}

func mapWorkloadToPDBs(reconcileCtx context.Context, virtualClient client.Client, obj client.Object) []ctrl.Request {
	if obj == nil {
		return nil
	}
	ns := obj.GetNamespace()
	if ns == "" {
		return nil
	}

	var targetLabels map[string]string
	switch o := obj.(type) {
	case *appsv1.Deployment:
		targetLabels = o.Spec.Template.Labels
	case *appsv1.StatefulSet:
		targetLabels = o.Spec.Template.Labels
	case *appsv1.ReplicaSet:
		targetLabels = o.Spec.Template.Labels
	case *corev1.Pod:
		targetLabels = o.GetLabels()
	default:
		targetLabels = obj.GetLabels()
	}

	pdbList := &policyv1.PodDisruptionBudgetList{}
	if err := virtualClient.List(reconcileCtx, pdbList, client.InNamespace(ns)); err != nil {
		return nil
	}

	var requests []ctrl.Request
	for _, pdb := range pdbList.Items {
		if !needsBudgetConversion(&pdb) {
			continue
		}
		if pdb.Spec.Selector == nil {
			continue
		}
		selector, err := metav1.LabelSelectorAsSelector(pdb.Spec.Selector)
		if err != nil {
			continue
		}
		if selector.Matches(labels.Set(targetLabels)) {
			requests = append(requests, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: pdb.Namespace,
					Name:      pdb.Name,
				},
			})
		}
	}
	return requests
}

func mapWorkloadUpdateToPDBs(reconcileCtx context.Context, virtualClient client.Client, oldObj, newObj client.Object) []ctrl.Request {
	seen := make(map[ctrl.Request]struct{})
	var requests []ctrl.Request
	for _, obj := range []client.Object{oldObj, newObj} {
		if obj == nil {
			continue
		}
		for _, req := range mapWorkloadToPDBs(reconcileCtx, virtualClient, obj) {
			if _, ok := seen[req]; !ok {
				seen[req] = struct{}{}
				requests = append(requests, req)
			}
		}
	}
	return requests
}

func (s *pdbSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*policyv1.PodDisruptionBudget]) (ctrl.Result, error) {
	if event.HostOld != nil || event.Virtual.DeletionTimestamp != nil {
		return patcher.DeleteVirtualObject(ctx, event.Virtual, event.HostOld, "host object was deleted")
	}

	newPDB, err := s.translate(ctx, event.Virtual)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("translate: %w", err)
	}

	err = pro.ApplyPatchesHostObject(ctx, nil, newPDB, event.Virtual, ctx.Config.Sync.ToHost.PodDisruptionBudgets.Patches, false)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("apply patches: %w", err)
	}

	return patcher.CreateHostObject(ctx, event.Virtual, newPDB, s.EventRecorder(), true)
}

func (s *pdbSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*policyv1.PodDisruptionBudget]) (_ ctrl.Result, retErr error) {
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.ToHost.PodDisruptionBudgets.Patches, false))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
		if retErr != nil {
			s.EventRecorder().Eventf(
				event.Virtual,
				nil,
				"Warning",
				"SyncError",
				fmt.Sprintf("Sync%s", event.Virtual.GetObjectKind().GroupVersionKind().Kind),
				"Error syncing: %v",
				retErr,
			)
		}
	}()

	err = s.translateUpdate(ctx, event.Host, event.Virtual)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("translate update: %w", err)
	}

	// bi-directional sync of annotations and labels
	event.Virtual.Annotations, event.Host.Annotations = translate.AnnotationsBidirectionalUpdate(event)
	event.Virtual.Labels, event.Host.Labels = translate.LabelsBidirectionalUpdate(event)

	return ctrl.Result{}, nil
}

func (s *pdbSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*policyv1.PodDisruptionBudget]) (_ ctrl.Result, retErr error) {
	// virtual object is not here anymore, so we delete
	return patcher.DeleteHostObject(ctx, event.Host, event.VirtualOld, "virtual object was deleted")
}
