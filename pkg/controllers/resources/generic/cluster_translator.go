package generic

import (
	"context"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewClusterTranslator(physicalNamespace string, virtualClient client.Client, obj client.Object, nameTranslator translate.PhysicalNameTranslator) Translator {
	return &clusterTranslator{
		physicalNamespace: physicalNamespace,
		virtualClient:     virtualClient,
		obj:               obj,
		nameTranslator:    nameTranslator,
	}
}

type clusterTranslator struct {
	physicalNamespace string
	virtualClient     client.Client
	obj               client.Object
	nameTranslator    translate.PhysicalNameTranslator
}

func (n *clusterTranslator) IsManaged(pObj client.Object) (bool, error) {
	return translate.IsManagedCluster(n.physicalNamespace, pObj), nil
}

func (n *clusterTranslator) VirtualToPhysical(req types.NamespacedName, vObj client.Object) types.NamespacedName {
	return types.NamespacedName{
		Name: n.nameTranslator(req.Name, vObj),
	}
}

func (n *clusterTranslator) PhysicalToVirtual(pObj client.Object) types.NamespacedName {
	pAnnotations := pObj.GetAnnotations()
	if pAnnotations != nil && pAnnotations[translate.NameAnnotation] != "" {
		return types.NamespacedName{
			Namespace: pAnnotations[translate.NamespaceAnnotation],
			Name:      pAnnotations[translate.NameAnnotation],
		}
	}

	vObj := n.obj.DeepCopyObject().(client.Object)
	err := clienthelper.GetByIndex(context.Background(), n.virtualClient, vObj, constants.IndexByPhysicalName, pObj.GetName())
	if err != nil {
		return types.NamespacedName{}
	}

	return types.NamespacedName{
		Namespace: vObj.GetNamespace(),
		Name:      vObj.GetName(),
	}
}
