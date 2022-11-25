package translator

import (
	context2 "context"
	"reflect"
	"sort"
	"strings"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ManagedAnnotationsAnnotation = "vcluster.loft.sh/managed-annotations"
	NamespaceAnnotation          = "vcluster.loft.sh/object-namespace"
	NameAnnotation               = "vcluster.loft.sh/object-name"
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

	excludedAnnotations []string
	syncedLabels        []string

	virtualClient client.Client
	obj           client.Object

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
	return types.NamespacedName{
		Namespace: translate.Default.PhysicalNamespace(req.Namespace),
		Name:      translate.Default.PhysicalName(req.Name, req.Namespace),
	}
}

func (n *namespacedTranslator) PhysicalToVirtual(pObj client.Object) types.NamespacedName {
	pAnnotations := pObj.GetAnnotations()
	if pAnnotations != nil && pAnnotations[NameAnnotation] != "" {
		return types.NamespacedName{
			Namespace: pAnnotations[NamespaceAnnotation],
			Name:      pAnnotations[NameAnnotation],
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
	return TranslateMetadata(vObj, n.syncedLabels, n.excludedAnnotations...)
}

func TranslateMetadata(vObj client.Object, syncedLabels []string, excludedAnnotations ...string) client.Object {
	pObj, err := setupMetadataWithName(vObj, func(vName string, vObj client.Object) string {
		return translate.Default.ObjectPhysicalName(vObj)
	})
	if err != nil {
		return nil
	}
	pObj.SetLabels(TranslateLabels(vObj, nil, syncedLabels))
	pObj.SetAnnotations(TranslateAnnotations(vObj, nil, excludedAnnotations))
	return pObj
}

func (n *namespacedTranslator) TranslateMetadataUpdate(vObj client.Object, pObj client.Object) (bool, map[string]string, map[string]string) {
	return TranslateMetadataUpdate(vObj, pObj, n.syncedLabels, n.excludedAnnotations...)
}

func TranslateMetadataUpdate(vObj client.Object, pObj client.Object, syncedLabels []string, excludedAnnotations ...string) (bool, map[string]string, map[string]string) {
	updatedAnnotations := TranslateAnnotations(vObj, pObj, excludedAnnotations)
	updatedLabels := TranslateLabels(vObj, pObj, syncedLabels)
	return !equality.Semantic.DeepEqual(updatedAnnotations, pObj.GetAnnotations()) || !equality.Semantic.DeepEqual(updatedLabels, pObj.GetLabels()), updatedAnnotations, updatedLabels
}

func TranslateAnnotations(src client.Object, dest client.Object, excluded []string) map[string]string {
	excluded = append(excluded, ManagedAnnotationsAnnotation, NameAnnotation, NamespaceAnnotation)

	retMap := map[string]string{}
	managedAnnotations := []string{}
	if src != nil {
		for k, v := range src.GetAnnotations() {
			if translate.Exists(excluded, k) {
				continue
			}

			retMap[k] = v
			managedAnnotations = append(managedAnnotations, k)
		}
	}

	if dest != nil {
		pAnnotations := dest.GetAnnotations()
		if pAnnotations != nil {
			oldManagedAnnotationsStr := pAnnotations[ManagedAnnotationsAnnotation]
			oldManagedAnnotations := strings.Split(oldManagedAnnotationsStr, "\n")

			for key, value := range pAnnotations {
				if translate.Exists(excluded, key) {
					if value != "" {
						retMap[key] = value
					}
					continue
				} else if translate.Exists(managedAnnotations, key) || (translate.Exists(oldManagedAnnotations, key) && !translate.Exists(managedAnnotations, key)) {
					continue
				}

				retMap[key] = value
			}
		}
	}

	sort.Strings(managedAnnotations)
	retMap[NameAnnotation] = src.GetName()
	if src.GetNamespace() == "" {
		delete(retMap, NamespaceAnnotation)
	} else {
		retMap[NamespaceAnnotation] = src.GetNamespace()
	}

	managedAnnotationsStr := strings.Join(managedAnnotations, "\n")
	if managedAnnotationsStr == "" {
		delete(retMap, ManagedAnnotationsAnnotation)
	} else {
		retMap[ManagedAnnotationsAnnotation] = managedAnnotationsStr
	}
	return retMap
}

func TranslateLabels(src client.Object, dest client.Object, syncedLabels []string) map[string]string {
	newLabels := map[string]string{}
	vObjLabels := src.GetLabels()
	for k, v := range vObjLabels {
		newLabels[translate.ConvertLabelKey(k)] = v
	}
	if vObjLabels != nil {
		for _, k := range syncedLabels {
			if value, ok := vObjLabels[k]; ok {
				newLabels[k] = value
			}
		}
	}
	if dest != nil {
		pObjLabels := dest.GetLabels()
		if pObjLabels != nil && pObjLabels[translate.ControllerLabel] != "" {
			newLabels[translate.ControllerLabel] = pObjLabels[translate.ControllerLabel]
		}
	}

	newLabels[translate.MarkerLabel] = translate.Suffix
	if src.GetNamespace() != "" {
		newLabels[translate.NamespaceLabel] = src.GetNamespace()
	}
	return newLabels
}

func setupMetadataWithName(vObj client.Object, translator PhysicalNameTranslator) (client.Object, error) {
	target := vObj.DeepCopyObject().(client.Object)
	m, err := meta.Accessor(target)
	if err != nil {
		return nil, err
	}

	// reset metadata & translate name and namespace
	ResetObjectMetadata(m)
	m.SetName(translator(m.GetName(), vObj))
	if vObj.GetNamespace() != "" {
		m.SetNamespace(translate.Default.PhysicalNamespace(vObj.GetNamespace()))

		// set owning stateful set if defined
		if translate.Owner != nil {
			m.SetOwnerReferences(translate.Default.GetOwnerReference(vObj))
		}
	}

	return target, nil
}

// ResetObjectMetadata resets the objects metadata except name, namespace and annotations
func ResetObjectMetadata(obj metav1.Object) {
	obj.SetGenerateName("")
	obj.SetSelfLink("")
	obj.SetUID("")
	obj.SetResourceVersion("")
	obj.SetGeneration(0)
	obj.SetCreationTimestamp(metav1.Time{})
	obj.SetDeletionTimestamp(nil)
	obj.SetDeletionGracePeriodSeconds(nil)
	obj.SetOwnerReferences(nil)
	obj.SetFinalizers(nil)
	obj.SetManagedFields(nil)
}
