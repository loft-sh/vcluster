package generic

import (
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewMirrorBackwardTranslator() Translator {
	return &mirrorBackwardTranslator{}
}

type mirrorBackwardTranslator struct {
}

func (n *mirrorBackwardTranslator) IsManaged(pObj client.Object) (bool, error) {
	return true, nil
}

func (n *mirrorBackwardTranslator) VirtualToPhysical(req types.NamespacedName, _ client.Object) types.NamespacedName {
	return req
}

func (n *mirrorBackwardTranslator) PhysicalToVirtual(pObj client.Object) types.NamespacedName {
	return types.NamespacedName{
		Namespace: pObj.GetNamespace(),
		Name:      pObj.GetName(),
	}
}
