package translator

import (
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewMirrorPhysicalTranslator(name string, obj client.Object) Translator {
	return &mirrorPhysicalTranslator{
		name: name,
		obj:  obj,
	}
}

type mirrorPhysicalTranslator struct {
	name string
	obj  client.Object
}

func (n *mirrorPhysicalTranslator) Name() string {
	return n.name
}

func (n *mirrorPhysicalTranslator) Resource() client.Object {
	return n.obj.DeepCopyObject().(client.Object)
}

func (n *mirrorPhysicalTranslator) TranslateMetadata(pObj client.Object) client.Object {
	vObj := pObj.DeepCopyObject().(client.Object)
	vObj.SetResourceVersion("")
	vObj.SetUID("")
	vObj.SetManagedFields(nil)
	vObj.SetOwnerReferences(nil)
	return vObj
}

func (n *mirrorPhysicalTranslator) TranslateMetadataUpdate(vObj client.Object, pObj client.Object) (changed bool, annotations map[string]string, labels map[string]string) {
	updatedAnnotations := pObj.GetAnnotations()
	updatedLabels := pObj.GetLabels()
	return !equality.Semantic.DeepEqual(updatedAnnotations, vObj.GetAnnotations()) || !equality.Semantic.DeepEqual(updatedLabels, vObj.GetLabels()), updatedAnnotations, updatedLabels
}

func (n *mirrorPhysicalTranslator) IsManaged(pObj client.Object) (bool, error) {
	return true, nil
}

func (n *mirrorPhysicalTranslator) VirtualToPhysical(req types.NamespacedName, _ client.Object) types.NamespacedName {
	return req
}

func (n *mirrorPhysicalTranslator) PhysicalToVirtual(pObj client.Object) types.NamespacedName {
	return types.NamespacedName{
		Namespace: pObj.GetNamespace(),
		Name:      pObj.GetName(),
	}
}
