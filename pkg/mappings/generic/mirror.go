package generic

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func NewMirrorMapper(obj client.Object) (synccontext.Mapper, error) {
	gvk, err := apiutil.GVKForObject(obj, scheme.Scheme)
	if err != nil {
		return nil, fmt.Errorf("retrieve GVK for object failed: %w", err)
	}

	return &mirrorMapper{
		gvk: gvk,
	}, nil
}

type mirrorMapper struct {
	gvk schema.GroupVersionKind
}

func (n *mirrorMapper) GroupVersionKind() schema.GroupVersionKind {
	return n.gvk
}

func (n *mirrorMapper) VirtualToHost(_ *synccontext.SyncContext, req types.NamespacedName, _ client.Object) types.NamespacedName {
	pNamespace := req.Namespace
	if pNamespace != "" {
		pNamespace = translate.Default.HostNamespace(pNamespace)
	}

	return types.NamespacedName{
		Namespace: pNamespace,
		Name:      req.Name,
	}
}

func (n *mirrorMapper) HostToVirtual(_ *synccontext.SyncContext, req types.NamespacedName, pObj client.Object) types.NamespacedName {
	if pObj != nil {
		pAnnotations := pObj.GetAnnotations()
		if pAnnotations != nil && pAnnotations[translate.NameAnnotation] != "" {
			return types.NamespacedName{
				Namespace: pAnnotations[translate.NamespaceAnnotation],
				Name:      pAnnotations[translate.NameAnnotation],
			}
		}
	}

	// if a namespace is requested we need to return early here
	if req.Namespace != "" {
		return types.NamespacedName{}
	}

	return types.NamespacedName{
		Name: req.Name,
	}
}

func (n *mirrorMapper) IsManaged(*synccontext.SyncContext, client.Object) (bool, error) {
	return true, nil
}
