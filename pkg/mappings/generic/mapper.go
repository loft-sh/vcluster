package generic

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// PhysicalNameWithObjectFunc is a definition to translate a name that also optionally expects a vObj
type PhysicalNameWithObjectFunc func(vName, vNamespace string, vObj client.Object) string

// PhysicalNameFunc is a definition to translate a name
type PhysicalNameFunc func(vName, vNamespace string) string

// NewMapper creates a new mapper with a custom physical name func
func NewMapper(ctx *synccontext.RegisterContext, obj client.Object, translateName PhysicalNameFunc, options ...MapperOption) (synccontext.Mapper, error) {
	return NewMapperWithObject(ctx, obj, func(vName, vNamespace string, _ client.Object) string {
		return translateName(vName, vNamespace)
	}, options...)
}

// NewMapperWithObject creates a new mapper with a custom physical name func
func NewMapperWithObject(ctx *synccontext.RegisterContext, obj client.Object, translateName PhysicalNameWithObjectFunc, options ...MapperOption) (synccontext.Mapper, error) {
	gvk, err := apiutil.GVKForObject(obj, scheme.Scheme)
	if err != nil {
		return nil, fmt.Errorf("retrieve GVK for object failed: %w", err)
	}

	mapperOptions := getOptions(options...)
	if !mapperOptions.SkipIndex {
		err = ctx.VirtualManager.GetFieldIndexer().IndexField(ctx, obj.DeepCopyObject().(client.Object), constants.IndexByPhysicalName, func(rawObj client.Object) []string {
			if rawObj.GetNamespace() != "" {
				return []string{translate.Default.HostNamespace(rawObj.GetNamespace()) + "/" + translateName(rawObj.GetName(), rawObj.GetNamespace(), rawObj)}
			}

			return []string{translateName(rawObj.GetName(), rawObj.GetNamespace(), rawObj)}
		})
		if err != nil {
			return nil, fmt.Errorf("index field: %w", err)
		}
	}

	return &mapper{
		translateName: translateName,
		virtualClient: ctx.VirtualManager.GetClient(),
		obj:           obj,
		gvk:           gvk,
	}, nil
}

type mapper struct {
	translateName PhysicalNameWithObjectFunc
	virtualClient client.Client

	obj client.Object
	gvk schema.GroupVersionKind
}

func (n *mapper) GroupVersionKind() schema.GroupVersionKind {
	return n.gvk
}

func (n *mapper) VirtualToHost(_ *synccontext.SyncContext, req types.NamespacedName, vObj client.Object) types.NamespacedName {
	return types.NamespacedName{
		Namespace: translate.Default.HostNamespace(req.Namespace),
		Name:      n.translateName(req.Name, req.Namespace, vObj),
	}
}

func (n *mapper) HostToVirtual(ctx *synccontext.SyncContext, req types.NamespacedName, pObj client.Object) types.NamespacedName {
	if pObj != nil {
		pAnnotations := pObj.GetAnnotations()
		if pAnnotations != nil && pAnnotations[translate.NameAnnotation] != "" {
			return types.NamespacedName{
				Namespace: pAnnotations[translate.NamespaceAnnotation],
				Name:      pAnnotations[translate.NameAnnotation],
			}
		}
	}

	key := req.Name
	if req.Namespace != "" {
		key = req.Namespace + "/" + req.Name
	}

	vObj := n.obj.DeepCopyObject().(client.Object)
	err := clienthelper.GetByIndex(ctx, n.virtualClient, vObj, constants.IndexByPhysicalName, key)
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

func (n *mapper) IsManaged(ctx *synccontext.SyncContext, pObj client.Object) (bool, error) {
	return translate.Default.IsManaged(ctx, pObj), nil
}
