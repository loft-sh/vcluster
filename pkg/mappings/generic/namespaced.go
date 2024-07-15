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

type MapperOption func(options *MapperOptions)

func SkipIndex() MapperOption {
	return func(options *MapperOptions) {
		options.SkipIndex = true
	}
}

type MapperOptions struct {
	SkipIndex bool
}

func getOptions(options ...MapperOption) *MapperOptions {
	newOptions := &MapperOptions{}
	for _, option := range options {
		option(newOptions)
	}
	return newOptions
}

func NewNamespacedMapper(ctx *synccontext.RegisterContext, obj client.Object, translateName translate.PhysicalNameFunc, options ...MapperOption) (mappings.Mapper, error) {
	gvk, err := apiutil.GVKForObject(obj, scheme.Scheme)
	if err != nil {
		return nil, fmt.Errorf("retrieve GVK for object failed: %w", err)
	}

	mapperOptions := getOptions(options...)
	if !mapperOptions.SkipIndex {
		err = ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, obj.DeepCopyObject().(client.Object), constants.IndexByPhysicalName, func(rawObj client.Object) []string {
			return []string{translate.Default.PhysicalNamespace(rawObj.GetNamespace()) + "/" + translateName(rawObj.GetName(), rawObj.GetNamespace())}
		})
		if err != nil {
			return nil, fmt.Errorf("index field: %w", err)
		}
	}

	return &namespacedMapper{
		translateName: translateName,
		virtualClient: ctx.VirtualManager.GetClient(),
		obj:           obj,
		gvk:           gvk,
	}, nil
}

type namespacedMapper struct {
	translateName translate.PhysicalNameFunc
	virtualClient client.Client
	obj           client.Object
	gvk           schema.GroupVersionKind
}

func (n *namespacedMapper) GroupVersionKind() schema.GroupVersionKind {
	return n.gvk
}

func (n *namespacedMapper) VirtualToHost(_ context2.Context, req types.NamespacedName, _ client.Object) types.NamespacedName {
	return types.NamespacedName{
		Namespace: translate.Default.PhysicalNamespace(req.Namespace),
		Name:      n.translateName(req.Name, req.Namespace),
	}
}

func (n *namespacedMapper) HostToVirtual(ctx context2.Context, req types.NamespacedName, pObj client.Object) types.NamespacedName {
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
	err := clienthelper.GetByIndex(ctx, n.virtualClient, vObj, constants.IndexByPhysicalName, req.Namespace+"/"+req.Name)
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
