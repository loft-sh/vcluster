package translator

import (
	context2 "context"
	"reflect"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewNamespacedTranslator(ctx *context.RegisterContext, name string, obj client.Object, excludedAnnotations ...string) NamespacedTranslator {
	return &namespacedTranslator{
		name: name,

		syncedLabels:        ctx.Options.SyncLabels,
		excludedAnnotations: excludedAnnotations,

		virtualClient: ctx.VirtualManager.GetClient(),
		obj:           obj,

		eventRecorder: ctx.VirtualManager.GetEventRecorderFor(name + "-syncer"),
	}
}

type namespacedTranslator struct {
	name string

	nameTranslator      translate.PhysicalNamespacedNameTranslator
	excludedAnnotations []string
	syncedLabels        []string

	virtualClient client.Client
	obj           client.Object

	eventRecorder record.EventRecorder
}

func (n *namespacedTranslator) SetNameTranslator(nameTranslator translate.PhysicalNamespacedNameTranslator) {
	n.nameTranslator = nameTranslator
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
		return []string{translate.Default.PhysicalNamespace(rawObj.GetNamespace()) + "/" + translate.Default.PhysicalName(rawObj.GetName(), rawObj.GetNamespace())}
	})
}

func (n *namespacedTranslator) SyncDownCreate(ctx *context.SyncContext, vObj, pObj client.Object) (ctrl.Result, error) {
	ctx.Log.Infof("create physical %s %s/%s", n.name, pObj.GetNamespace(), pObj.GetName())
	err := ctx.PhysicalClient.Create(ctx.Context, pObj)
	if err != nil {
		ctx.Log.Infof("error syncing %s %s/%s to physical cluster: %v", n.name, vObj.GetNamespace(), vObj.GetName(), err)
		n.eventRecorder.Eventf(vObj, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (n *namespacedTranslator) SyncDownUpdate(ctx *context.SyncContext, vObj, pObj client.Object) (ctrl.Result, error) {
	// this is needed because of interface nil check
	if !(pObj == nil || (reflect.ValueOf(pObj).Kind() == reflect.Ptr && reflect.ValueOf(pObj).IsNil())) {
		ctx.Log.Infof("updating physical %s/%s, because virtual %s have changed", pObj.GetNamespace(), pObj.GetName(), n.name)
		err := ctx.PhysicalClient.Update(ctx.Context, pObj)
		if err != nil {
			n.eventRecorder.Eventf(vObj, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (n *namespacedTranslator) IsManaged(pObj client.Object) (bool, error) {
	return translate.Default.IsManaged(pObj), nil
}

func (n *namespacedTranslator) VirtualToPhysical(req types.NamespacedName, vObj client.Object) types.NamespacedName {
	name := translate.Default.PhysicalName(req.Name, req.Namespace)
	if n.nameTranslator != nil {
		name = n.nameTranslator(req, vObj)
	}

	return types.NamespacedName{
		Namespace: translate.Default.PhysicalNamespace(req.Namespace),
		Name:      name,
	}
}

func (n *namespacedTranslator) PhysicalToVirtual(pObj client.Object) types.NamespacedName {
	pAnnotations := pObj.GetAnnotations()
	if pAnnotations != nil && pAnnotations[translate.NameAnnotation] != "" {
		return types.NamespacedName{
			Namespace: pAnnotations[translate.NamespaceAnnotation],
			Name:      pAnnotations[translate.NameAnnotation],
		}
	}

	vObj := n.obj.DeepCopyObject().(client.Object)
	err := clienthelper.GetByIndex(context2.Background(), n.virtualClient, vObj, constants.IndexByPhysicalName, pObj.GetNamespace()+"/"+pObj.GetName())
	if err != nil {
		return types.NamespacedName{}
	}

	return types.NamespacedName{
		Namespace: vObj.GetNamespace(),
		Name:      vObj.GetName(),
	}
}

func (n *namespacedTranslator) TranslateMetadata(vObj client.Object) client.Object {
	pObj := vObj.DeepCopyObject().(client.Object)
	m, err := meta.Accessor(pObj)
	if err != nil {
		return nil
	}

	// reset metadata & translate name and namespace
	translate.ResetObjectMetadata(m)
	m.SetName(n.VirtualToPhysical(types.NamespacedName{Name: vObj.GetName(), Namespace: vObj.GetNamespace()}, vObj).Name)
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

func (n *namespacedTranslator) TranslateMetadataUpdate(vObj client.Object, pObj client.Object) (bool, map[string]string, map[string]string) {
	return translate.Default.ApplyMetadataUpdate(vObj, pObj, n.syncedLabels, n.excludedAnnotations...)
}
