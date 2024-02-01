package translator

import (
	"context"

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

func (n *mirrorPhysicalTranslator) TranslateMetadata(_ context.Context, pObj client.Object) client.Object {
	vObj := pObj.DeepCopyObject().(client.Object)
	vObj.SetResourceVersion("")
	vObj.SetUID("")
	vObj.SetManagedFields(nil)
	vObj.SetOwnerReferences(nil)
	return vObj
}

func (n *mirrorPhysicalTranslator) TranslateMetadataUpdate(_ context.Context, vObj client.Object, pObj client.Object) (changed bool, annotations map[string]string, labels map[string]string) {
	updatedAnnotations := pObj.GetAnnotations()
	updatedLabels := pObj.GetLabels()
	return !equality.Semantic.DeepEqual(updatedAnnotations, vObj.GetAnnotations()) || !equality.Semantic.DeepEqual(updatedLabels, vObj.GetLabels()), updatedAnnotations, updatedLabels
}

func (n *mirrorPhysicalTranslator) IsManaged(context.Context, client.Object) (bool, error) {
	return true, nil
}

func (n *mirrorPhysicalTranslator) VirtualToHost(_ context.Context, req types.NamespacedName, _ client.Object) types.NamespacedName {
	return req
}

func (n *mirrorPhysicalTranslator) HostToVirtual(_ context.Context, req types.NamespacedName, _ client.Object) types.NamespacedName {
	return req
}
