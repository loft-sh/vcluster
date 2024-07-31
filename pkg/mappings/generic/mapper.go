package generic

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// PhysicalNameWithObjectFunc is a definition to translate a name that also optionally expects a vObj
type PhysicalNameWithObjectFunc func(ctx *synccontext.SyncContext, vName, vNamespace string, vObj client.Object) string

// PhysicalNameFunc is a definition to translate a name
type PhysicalNameFunc func(ctx *synccontext.SyncContext, vName, vNamespace string) string

// NewMapper creates a new mapper with a custom physical name func
func NewMapper(ctx *synccontext.RegisterContext, obj client.Object, translateName PhysicalNameFunc) (synccontext.Mapper, error) {
	return NewMapperWithObject(ctx, obj, func(ctx *synccontext.SyncContext, vName, vNamespace string, _ client.Object) string {
		return translateName(ctx, vName, vNamespace)
	})
}

// NewMapperWithObject creates a new mapper with a custom physical name func
func NewMapperWithObject(ctx *synccontext.RegisterContext, obj client.Object, translateName PhysicalNameWithObjectFunc) (synccontext.Mapper, error) {
	return newMapper(ctx, obj, true, translateName)
}

// NewMapperWithoutRecorder creates a new mapper with a recorder to store mappings in the mappings store
func NewMapperWithoutRecorder(ctx *synccontext.RegisterContext, obj client.Object, translateName PhysicalNameWithObjectFunc) (synccontext.Mapper, error) {
	return newMapper(ctx, obj, false, translateName)
}

// newMapper creates a new mapper with a recorder to store mappings in the mappings store
func newMapper(ctx *synccontext.RegisterContext, obj client.Object, recorder bool, translateName PhysicalNameWithObjectFunc) (synccontext.Mapper, error) {
	gvk, err := apiutil.GVKForObject(obj, scheme.Scheme)
	if err != nil {
		return nil, fmt.Errorf("retrieve GVK for object failed: %w", err)
	}

	var retMapper synccontext.Mapper = &mapper{
		translateName: translateName,
		virtualClient: ctx.VirtualManager.GetClient(),
		obj:           obj,
		gvk:           gvk,
	}
	if recorder {
		retMapper = WithRecorder(retMapper)
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

func (n *mapper) Migrate(ctx *synccontext.RegisterContext, mapper synccontext.Mapper) error {
	gvk := mapper.GroupVersionKind()
	listGvk := schema.GroupVersionKind{
		Group:   gvk.Group,
		Version: gvk.Version,
		Kind:    gvk.Kind + "List",
	}

	list, err := scheme.Scheme.New(listGvk)
	if err != nil {
		if !runtime.IsNotRegisteredError(err) {
			return fmt.Errorf("migrate create object list %s: %w", listGvk.String(), err)
		}

		list = &unstructured.UnstructuredList{}
	}

	uList, ok := list.(*unstructured.UnstructuredList)
	if ok {
		uList.SetKind(listGvk.Kind)
		uList.SetAPIVersion(listGvk.GroupVersion().String())
	}

	// it's safe to list here without namespace as this will just list all items in the cache
	err = ctx.VirtualManager.GetClient().List(ctx, list.(client.ObjectList))
	if err != nil {
		return fmt.Errorf("error listing %s: %w", listGvk.String(), err)
	}

	items, err := meta.ExtractList(list)
	if err != nil {
		return fmt.Errorf("extract list %s: %w", listGvk.String(), err)
	}

	for _, item := range items {
		clientObject, ok := item.(client.Object)
		if !ok {
			continue
		}

		vName := types.NamespacedName{Name: clientObject.GetName(), Namespace: clientObject.GetNamespace()}
		pName := mapper.VirtualToHost(ctx.ToSyncContext("migrate-"+listGvk.Kind), vName, clientObject)
		if pName.Name != "" {
			nameMapping := synccontext.NameMapping{
				GroupVersionKind: n.gvk,
				VirtualName:      vName,
				HostName:         pName,
			}

			err = ctx.Mappings.Store().RecordAndSaveReference(ctx, nameMapping, nameMapping)
			if err != nil {
				return fmt.Errorf("error saving reference in store: %w", err)
			}
		}
	}

	return nil
}

func (n *mapper) VirtualToHost(ctx *synccontext.SyncContext, req types.NamespacedName, vObj client.Object) (retName types.NamespacedName) {
	pNamespace := req.Namespace
	if pNamespace != "" {
		pNamespace = translate.Default.HostNamespace(ctx, pNamespace)
	}

	return types.NamespacedName{
		Namespace: pNamespace,
		Name:      n.translateName(ctx, req.Name, req.Namespace, vObj),
	}
}

func (n *mapper) HostToVirtual(_ *synccontext.SyncContext, _ types.NamespacedName, pObj client.Object) (retName types.NamespacedName) {
	if pObj != nil {
		pAnnotations := pObj.GetAnnotations()
		if pAnnotations[translate.NameAnnotation] != "" {
			// check if kind matches
			gvk, ok := pAnnotations[translate.KindAnnotation]
			if !ok || n.gvk.String() == gvk {
				return types.NamespacedName{
					Namespace: pAnnotations[translate.NamespaceAnnotation],
					Name:      pAnnotations[translate.NameAnnotation],
				}
			}
		}
	}

	return types.NamespacedName{}
}

func (n *mapper) IsManaged(ctx *synccontext.SyncContext, pObj client.Object) (bool, error) {
	return translate.Default.IsManaged(ctx, pObj), nil
}
