package translator

import (
	context2 "context"
	"reflect"
	"time"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewNamespacedTranslator(ctx *context.RegisterContext, name string, obj client.Object, excludedAnnotations ...string) NamespacedTranslator {
	return newNamespacedTranslator(ctx, name, obj, translate.Default.PhysicalName, excludedAnnotations...)
}

func NewShortNamespacedTranslator(ctx *context.RegisterContext, name string, obj client.Object, excludedAnnotations ...string) NamespacedTranslator {
	return newNamespacedTranslator(ctx, name, obj, translate.Default.PhysicalNameShort, excludedAnnotations...)
}

func newNamespacedTranslator(ctx *context.RegisterContext, name string, obj client.Object, translateName translate.PhysicalNameFunc, excludedAnnotations ...string) NamespacedTranslator {
	return &namespacedTranslator{
		name:          name,
		translateName: translateName,

		syncedLabels:        ctx.Config.Experimental.SyncSettings.SyncLabels,
		excludedAnnotations: excludedAnnotations,

		virtualClient: ctx.VirtualManager.GetClient(),
		obj:           obj,

		eventRecorder: ctx.VirtualManager.GetEventRecorderFor(name + "-syncer"),
	}
}

type namespacedTranslator struct {
	name string

	excludedAnnotations []string
	syncedLabels        []string

	virtualClient client.Client
	obj           client.Object

	translateName translate.PhysicalNameFunc

	eventRecorder record.EventRecorder
}

func (n *namespacedTranslator) EventRecorder() record.EventRecorder {
	return n.eventRecorder
}

func (n *namespacedTranslator) Name() string {
	return n.name
}

func (n *namespacedTranslator) Resource() client.Object {
	return n.obj.DeepCopyObject().(client.Object)
}

func (n *namespacedTranslator) RegisterIndices(ctx *context.RegisterContext) error {
	return ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, n.obj.DeepCopyObject().(client.Object), constants.IndexByPhysicalName, func(rawObj client.Object) []string {
		return []string{translate.Default.PhysicalNamespace(rawObj.GetNamespace()) + "/" + n.translateName(rawObj.GetName(), rawObj.GetNamespace())}
	})
}

func (n *namespacedTranslator) SyncToHostCreate(ctx *context.SyncContext, vObj, pObj client.Object) (ctrl.Result, error) {
	ctx.Log.Infof("create physical %s %s/%s", n.name, pObj.GetNamespace(), pObj.GetName())
	err := ctx.PhysicalClient.Create(ctx.Context, pObj)
	if err != nil {
		if kerrors.IsNotFound(err) {
			ctx.Log.Debugf("error syncing %s %s/%s to physical cluster: %v", n.name, vObj.GetNamespace(), vObj.GetName(), err)
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
		ctx.Log.Infof("error syncing %s %s/%s to physical cluster: %v", n.name, vObj.GetNamespace(), vObj.GetName(), err)
		n.eventRecorder.Eventf(vObj, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (n *namespacedTranslator) SyncToHostUpdate(ctx *context.SyncContext, vObj, pObj client.Object) (ctrl.Result, error) {
	// this is needed because of interface nil check
	if !(pObj == nil || (reflect.ValueOf(pObj).Kind() == reflect.Ptr && reflect.ValueOf(pObj).IsNil())) {
		ctx.Log.Infof("updating physical %s/%s, because virtual %s have changed", pObj.GetNamespace(), pObj.GetName(), n.name)
		err := ctx.PhysicalClient.Update(ctx.Context, pObj)
		if kerrors.IsConflict(err) {
			ctx.Log.Debugf("conflict syncing physical %s %s/%s", n.name, pObj.GetNamespace(), pObj.GetName())
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil {
			n.eventRecorder.Eventf(vObj, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (n *namespacedTranslator) IsManaged(_ context2.Context, pObj client.Object) (bool, error) {
	return translate.Default.IsManaged(pObj, n.translateName), nil
}

func (n *namespacedTranslator) VirtualToHost(_ context2.Context, req types.NamespacedName, _ client.Object) types.NamespacedName {
	return types.NamespacedName{
		Namespace: translate.Default.PhysicalNamespace(req.Namespace),
		Name:      n.translateName(req.Name, req.Namespace),
	}
}

func (n *namespacedTranslator) HostToVirtual(_ context2.Context, req types.NamespacedName, pObj client.Object) types.NamespacedName {
	if pObj != nil {
		pAnnotations := pObj.GetAnnotations()
		if pAnnotations != nil && pAnnotations[translate.NameAnnotation] != "" {
			return types.NamespacedName{
				Namespace: pAnnotations[translate.NamespaceAnnotation],
				Name:      pAnnotations[translate.NameAnnotation],
			}
		}
	}

	vObj := n.obj.DeepCopyObject().(client.Object)
	err := clienthelper.GetByIndex(context2.Background(), n.virtualClient, vObj, constants.IndexByPhysicalName, req.Namespace+"/"+req.Name)
	if err != nil {
		return types.NamespacedName{}
	}

	return types.NamespacedName{
		Namespace: vObj.GetNamespace(),
		Name:      vObj.GetName(),
	}
}

func (n *namespacedTranslator) TranslateMetadata(ctx context2.Context, vObj client.Object) client.Object {
	pObj := vObj.DeepCopyObject().(client.Object)
	m, err := meta.Accessor(pObj)
	if err != nil {
		return nil
	}

	// reset metadata & translate name and namespace
	translate.ResetObjectMetadata(m)
	m.SetName(n.VirtualToHost(ctx, types.NamespacedName{Name: vObj.GetName(), Namespace: vObj.GetNamespace()}, vObj).Name)
	if vObj.GetNamespace() != "" {
		m.SetNamespace(translate.Default.PhysicalNamespace(vObj.GetNamespace()))

		// set owning stateful set if defined
		if translate.Owner != nil {
			m.SetOwnerReferences(translate.GetOwnerReference(vObj))
		}
	}

	pObj.SetAnnotations(translate.Default.ApplyAnnotations(vObj, nil, n.excludedAnnotations))
	pObj.SetLabels(translate.Default.ApplyLabels(vObj, nil, n.syncedLabels))
	return pObj
}

func (n *namespacedTranslator) TranslateMetadataUpdate(_ context2.Context, vObj client.Object, pObj client.Object) (bool, map[string]string, map[string]string) {
	return translate.Default.ApplyMetadataUpdate(vObj, pObj, n.syncedLabels, n.excludedAnnotations...)
}
