package generic

import (
	context2 "context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/constants"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func NewClusterMapper(ctx *synccontext.RegisterContext, obj client.Object, nameTranslator translate.PhysicalNameClusterFunc, options ...MapperOption) (mappings.Mapper, error) {
	gvk, err := apiutil.GVKForObject(obj, scheme.Scheme)
	if err != nil {
		return nil, fmt.Errorf("retrieve GVK for object failed: %w", err)
	}

	mapperOptions := getOptions(options...)
	if !mapperOptions.SkipIndex {
		err = ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, obj.DeepCopyObject().(client.Object), constants.IndexByPhysicalName, func(rawObj client.Object) []string {
			return []string{nameTranslator(rawObj.GetName(), rawObj)}
		})
		if err != nil {
			return nil, fmt.Errorf("index field: %w", err)
		}
	}

	return &clusterMapper{
		obj:            obj,
		gvk:            gvk,
		nameTranslator: nameTranslator,
		virtualClient:  ctx.VirtualManager.GetClient(),
	}, nil
}

type clusterMapper struct {
	gvk            schema.GroupVersionKind
	obj            client.Object
	nameTranslator translate.PhysicalNameClusterFunc
	virtualClient  client.Client
}

func (n *clusterMapper) GroupVersionKind() schema.GroupVersionKind {
	return n.gvk
}

func (n *clusterMapper) VirtualToHost(_ context2.Context, req types.NamespacedName, vObj client.Object) types.NamespacedName {
	return types.NamespacedName{
		Name: n.nameTranslator(req.Name, vObj),
	}
}

func (n *clusterMapper) HostToVirtual(ctx context2.Context, req types.NamespacedName, pObj client.Object) types.NamespacedName {
	if pObj != nil {
		pAnnotations := pObj.GetAnnotations()
		if pAnnotations != nil && pAnnotations[translate.NameAnnotation] != "" {
			return types.NamespacedName{
				Namespace: pAnnotations[translate.NamespaceAnnotation],
				Name:      pAnnotations[translate.NameAnnotation],
			}
		}
	}

	vObj := n.obj.DeepCopyObject().(client.Object)
	err := clienthelper.GetByIndex(ctx, n.virtualClient, vObj, constants.IndexByPhysicalName, req.Name)
	if err != nil {
		if !kerrors.IsNotFound(err) && !kerrors.IsConflict(err) {
			panic(err.Error())
		}

		return types.NamespacedName{}
	}

	return types.NamespacedName{
		Namespace: vObj.GetNamespace(),
		Name:      vObj.GetName(),
	}
}
