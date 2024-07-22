package translator

import (
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/types"
	"k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewMirrorPhysicalTranslator(name string, obj client.Object, mapper synccontext.Mapper) types.Translator {
	return &mirrorPhysicalTranslator{
		Mapper: mapper,

		name: name,
		obj:  obj,
	}
}

type mirrorPhysicalTranslator struct {
	synccontext.Mapper

	name string
	obj  client.Object
}

func (n *mirrorPhysicalTranslator) Name() string {
	return n.name
}

func (n *mirrorPhysicalTranslator) Resource() client.Object {
	return n.obj.DeepCopyObject().(client.Object)
}

func (n *mirrorPhysicalTranslator) TranslateMetadata(_ *synccontext.SyncContext, pObj client.Object) client.Object {
	vObj := pObj.DeepCopyObject().(client.Object)
	vObj.SetResourceVersion("")
	vObj.SetUID("")
	vObj.SetManagedFields(nil)
	vObj.SetOwnerReferences(nil)
	return vObj
}

func (n *mirrorPhysicalTranslator) TranslateMetadataUpdate(_ *synccontext.SyncContext, vObj client.Object, pObj client.Object) (changed bool, annotations map[string]string, labels map[string]string) {
	updatedAnnotations := pObj.GetAnnotations()
	updatedLabels := pObj.GetLabels()
	return !equality.Semantic.DeepEqual(updatedAnnotations, vObj.GetAnnotations()) || !equality.Semantic.DeepEqual(updatedLabels, vObj.GetLabels()), updatedAnnotations, updatedLabels
}
