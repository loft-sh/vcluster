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
	"k8s.io/klog/v2"
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

	retMapper := &mapper{
		translateName: translateName,
		virtualClient: ctx.VirtualManager.GetClient(),
		obj:           obj,
		gvk:           gvk,
	}

	mapperOptions := getOptions(options...)
	if !mapperOptions.SkipIndex {
		err = ctx.VirtualManager.GetFieldIndexer().IndexField(ctx, obj.DeepCopyObject().(client.Object), constants.IndexByPhysicalName, func(rawObj client.Object) []string {
			// we build a sync context here to record the mapping
			syncContext := ctx.ToSyncContext(gvk.String())
			syncContext.Context = synccontext.WithMapping(syncContext.Context, synccontext.NameMapping{
				GroupVersionKind: gvk,
				VirtualName: types.NamespacedName{
					Name:      rawObj.GetName(),
					Namespace: rawObj.GetNamespace(),
				},
			})
			defer func() {
				err = syncContext.Close()
				if err != nil {
					klog.FromContext(ctx).Error(err, "save mapping")
				}
			}()

			// record mapping here
			pName := retMapper.VirtualToHost(syncContext, types.NamespacedName{Name: rawObj.GetName(), Namespace: rawObj.GetNamespace()}, rawObj)
			if pName.Namespace != "" {
				return []string{pName.Namespace + "/" + pName.Name}
			}

			return []string{pName.Name}
		})
		if err != nil {
			return nil, fmt.Errorf("index field: %w", err)
		}
	}

	return retMapper, nil
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

func (n *mapper) VirtualToHost(ctx *synccontext.SyncContext, req types.NamespacedName, vObj client.Object) (retName types.NamespacedName) {
	defer func() {
		RecordMapping(ctx, retName, req, n.gvk)
	}()

	// check store first
	vName, ok := VirtualToHostFromStore(ctx, req, n.gvk)
	if ok {
		return vName
	}

	pNamespace := req.Namespace
	if pNamespace != "" {
		pNamespace = translate.Default.HostNamespace(pNamespace)
	}

	return types.NamespacedName{
		Namespace: pNamespace,
		Name:      n.translateName(req.Name, req.Namespace, vObj),
	}
}

func (n *mapper) HostToVirtual(ctx *synccontext.SyncContext, req types.NamespacedName, pObj client.Object) (retName types.NamespacedName) {
	defer func() {
		RecordMapping(ctx, req, retName, n.gvk)
	}()

	// check store first
	vName, ok := HostToVirtualFromStore(ctx, req, n.gvk)
	if ok {
		return vName
	}

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

func RecordMapping(ctx *synccontext.SyncContext, pName, vName types.NamespacedName, gvk schema.GroupVersionKind) {
	if pName.Name == "" || vName.Name == "" {
		return
	}

	if ctx != nil && ctx.Mappings != nil && ctx.Mappings.Store() != nil {
		// check if we have the owning object in the context
		belongsTo, ok := synccontext.MappingFrom(ctx)
		if !ok {
			return
		}

		// record the reference
		err := ctx.Mappings.Store().RecordReference(ctx, synccontext.NameMapping{
			GroupVersionKind: gvk,

			HostName:    pName,
			VirtualName: vName,
		}, belongsTo)
		if err != nil {
			klog.FromContext(ctx).Error(err, "record name mapping", "host", pName, "virtual", vName)
		}
	}
}

func HostToVirtualFromStore(ctx *synccontext.SyncContext, req types.NamespacedName, gvk schema.GroupVersionKind) (types.NamespacedName, bool) {
	if ctx != nil && ctx.Mappings != nil && ctx.Mappings.Store() != nil {
		return ctx.Mappings.Store().HostToVirtualName(ctx, synccontext.Object{
			GroupVersionKind: gvk,
			NamespacedName:   req,
		})
	}

	return types.NamespacedName{}, false
}

func VirtualToHostFromStore(ctx *synccontext.SyncContext, req types.NamespacedName, gvk schema.GroupVersionKind) (types.NamespacedName, bool) {
	if ctx != nil && ctx.Mappings != nil && ctx.Mappings.Store() != nil {
		return ctx.Mappings.Store().VirtualToHostName(ctx, synccontext.Object{
			GroupVersionKind: gvk,
			NamespacedName:   req,
		})
	}

	return types.NamespacedName{}, false
}
