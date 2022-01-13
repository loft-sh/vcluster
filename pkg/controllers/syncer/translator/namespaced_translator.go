package translator

import (
	context2 "context"
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"sort"
	"strings"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
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
	LabelPrefix                  = "vcluster.loft.sh/label"
)

func DefaultPhysicalName(vName string, vObj client.Object) string {
	name, namespace := vObj.GetName(), vObj.GetNamespace()
	return translate.PhysicalName(name, namespace)
}

func NewNamespacedTranslator(ctx *context.RegisterContext, name string, obj client.Object, excludedAnnotations ...string) NamespacedTranslator {
	return &namespacedTranslator{
		name: name,

		physicalNamespace:   ctx.TargetNamespace,
		excludedAnnotations: excludedAnnotations,

		virtualClient: ctx.VirtualManager.GetClient(),
		obj:           obj,

		eventRecorder: ctx.EventBroadcaster.NewRecorder(ctx.PhysicalManager.GetScheme(), corev1.EventSource{Component: name + "-syncer"}),
	}
}

type namespacedTranslator struct {
	name string

	physicalNamespace   string
	excludedAnnotations []string

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
		return []string{ObjectPhysicalName(rawObj)}
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
	return translate.IsManaged(pObj), nil
}

func (n *namespacedTranslator) VirtualToPhysical(req types.NamespacedName, vObj client.Object) types.NamespacedName {
	return types.NamespacedName{
		Namespace: n.physicalNamespace,
		Name:      translate.PhysicalName(req.Name, req.Namespace),
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
	err := clienthelper.GetByIndex(context2.Background(), n.virtualClient, vObj, constants.IndexByPhysicalName, pObj.GetName())
	if err != nil {
		return types.NamespacedName{}
	}

	return types.NamespacedName{
		Namespace: vObj.GetNamespace(),
		Name:      vObj.GetName(),
	}
}

func (n *namespacedTranslator) TranslateMetadata(vObj client.Object) client.Object {
	return TranslateMetadata(n.physicalNamespace, vObj, n.excludedAnnotations...)
}

func TranslateMetadata(phyiscalNamespace string, vObj client.Object, excludedAnnotations ...string) client.Object {
	pObj, err := setupMetadataWithName(phyiscalNamespace, vObj, DefaultPhysicalName)
	if err != nil {
		return nil
	}

	pObj.SetLabels(translateLabels(vObj))
	pObj.SetAnnotations(translateAnnotations(vObj, nil, excludedAnnotations))
	return pObj
}

func (n *namespacedTranslator) TranslateMetadataUpdate(vObj client.Object, pObj client.Object) (bool, map[string]string, map[string]string) {
	return TranslateMetadataUpdate(vObj, pObj, n.excludedAnnotations...)
}

func TranslateMetadataUpdate(vObj client.Object, pObj client.Object, excludedAnnotations ...string) (bool, map[string]string, map[string]string) {
	updatedAnnotations := translateAnnotations(vObj, pObj, excludedAnnotations)
	updatedLabels := translateLabels(vObj)
	return !equality.Semantic.DeepEqual(updatedAnnotations, pObj.GetAnnotations()) || !equality.Semantic.DeepEqual(updatedLabels, pObj.GetLabels()), updatedAnnotations, updatedLabels
}

func translateAnnotations(vObj client.Object, pObj client.Object, excluded []string) map[string]string {
	excluded = append(excluded, ManagedAnnotationsAnnotation, NameAnnotation, NamespaceAnnotation)

	retMap := map[string]string{}
	managedAnnotations := []string{}
	if vObj != nil {
		for k, v := range vObj.GetAnnotations() {
			if exists(excluded, k) {
				continue
			}

			retMap[k] = v
			managedAnnotations = append(managedAnnotations, k)
		}
	}

	if pObj != nil {
		pAnnotations := pObj.GetAnnotations()
		if pAnnotations != nil {
			oldManagedAnnotationsStr := pAnnotations[ManagedAnnotationsAnnotation]
			oldManagedAnnotations := strings.Split(oldManagedAnnotationsStr, "\n")

			for key, value := range pAnnotations {
				if exists(excluded, key) {
					if value != "" {
						retMap[key] = value
					}
					continue
				} else if exists(managedAnnotations, key) || (exists(oldManagedAnnotations, key) && !exists(managedAnnotations, key)) {
					continue
				}

				retMap[key] = value
			}
		}
	}

	sort.Strings(managedAnnotations)
	retMap[NameAnnotation] = vObj.GetName()
	if vObj.GetNamespace() == "" {
		delete(retMap, NamespaceAnnotation)
	} else {
		retMap[NamespaceAnnotation] = vObj.GetNamespace()
	}

	managedAnnotationsStr := strings.Join(managedAnnotations, "\n")
	if managedAnnotationsStr == "" {
		delete(retMap, ManagedAnnotationsAnnotation)
	} else {
		retMap[ManagedAnnotationsAnnotation] = managedAnnotationsStr
	}
	return retMap
}

func translateLabels(vObj client.Object) map[string]string {
	newLabels := map[string]string{}
	for k, v := range vObj.GetLabels() {
		if k == translate.NamespaceLabel {
			newLabels[k] = v
			continue
		}

		newLabels[ConvertLabelKey(k)] = v
	}

	newLabels[translate.MarkerLabel] = translate.Suffix
	if vObj.GetNamespace() != "" {
		newLabels[translate.NamespaceLabel] = vObj.GetNamespace()
	}
	return newLabels
}

func setupMetadataWithName(targetNamespace string, vObj client.Object, translator PhysicalNameTranslator) (client.Object, error) {
	target := vObj.DeepCopyObject().(client.Object)
	m, err := meta.Accessor(target)
	if err != nil {
		return nil, err
	}

	// reset metadata & translate name and namespace
	ResetObjectMetadata(m)
	m.SetName(translator(m.GetName(), vObj))
	if vObj.GetNamespace() != "" {
		m.SetNamespace(targetNamespace)

		// set owning stateful set if defined
		if translate.Owner != nil {
			m.SetOwnerReferences(translate.GetOwnerReference())
		}
	}

	return target, nil
}

func ConvertLabelKey(key string) string {
	return ConvertLabelKeyWithPrefix(LabelPrefix, key)
}

func ConvertLabelKeyWithPrefix(prefix, key string) string {
	digest := sha256.Sum256([]byte(key))
	return translate.SafeConcatName(prefix, translate.Suffix, "x", hex.EncodeToString(digest[0:])[0:10])
}

func exists(a []string, k string) bool {
	for _, i := range a {
		if i == k {
			return true
		}
	}

	return false
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
	obj.SetClusterName("")
	obj.SetManagedFields(nil)
}
