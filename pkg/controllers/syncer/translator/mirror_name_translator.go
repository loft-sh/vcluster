package translator

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewMirrorPhysicalTranslator(name string, obj client.Object, mapper mappings.Mapper) Translator {
	return &mirrorPhysicalTranslator{
		Mapper: mapper,

		name: name,
		obj:  obj,
	}
}

type mirrorPhysicalTranslator struct {
	mappings.Mapper

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
