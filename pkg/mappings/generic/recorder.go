package generic

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func WithRecorder(mapper synccontext.Mapper) synccontext.Mapper {
	return &recorder{
		Mapper: mapper,
	}
}

type recorder struct {
	synccontext.Mapper
}

func (n *recorder) Migrate(ctx *synccontext.RegisterContext, mapper synccontext.Mapper) error {
	gvk := n.GroupVersionKind()
	listGvk := schema.GroupVersionKind{
		Group:   gvk.Group,
		Version: gvk.Version,
		Kind:    gvk.Kind + "List",
	}

	// migrate host objects first
	hostObjects, err := listObjects(ctx, ctx.HostManager.GetClient(), listGvk)
	if err != nil {
		return err
	}

	for _, item := range hostObjects {
		clientObject, ok := item.(client.Object)
		if !ok {
			continue
		}

		syncContext := ctx.ToSyncContext("migrate-" + listGvk.Kind)
		syncContext.Mappings = nil // this is necessary to avoid the NameAnnotation check
		isManaged, err := n.Mapper.IsManaged(syncContext, clientObject)
		if err != nil {
			klog.FromContext(ctx).Error(err, "is managed")
			continue
		} else if !isManaged {
			continue
		}

		pName := types.NamespacedName{Name: clientObject.GetName(), Namespace: clientObject.GetNamespace()}
		if ctx.Mappings.Store().HasHostObject(ctx, synccontext.Object{
			GroupVersionKind: gvk,
			NamespacedName:   pName,
		}) {
			continue
		}

		vName := n.Mapper.HostToVirtual(syncContext, pName, clientObject)
		if vName.Name != "" {
			nameMapping := synccontext.NameMapping{
				GroupVersionKind: gvk,
				VirtualName:      vName,
				HostName:         pName,
			}

			err = ctx.Mappings.Store().AddReferenceAndSave(ctx, nameMapping, nameMapping)
			if err != nil {
				klog.FromContext(ctx).Error(err, "saving reference in store", "mapping", nameMapping.String())
			}
		}
	}

	// migrate virtual objects
	virtualObjects, err := listObjects(ctx, ctx.VirtualManager.GetClient(), listGvk)
	if err != nil {
		return err
	}

	for _, item := range virtualObjects {
		clientObject, ok := item.(client.Object)
		if !ok {
			continue
		}

		vName := types.NamespacedName{Name: clientObject.GetName(), Namespace: clientObject.GetNamespace()}
		if ctx.Mappings.Store().HasVirtualObject(ctx, synccontext.Object{
			GroupVersionKind: gvk,
			NamespacedName:   vName,
		}) {
			continue
		}

		pName := n.Mapper.VirtualToHost(ctx.ToSyncContext("migrate-"+listGvk.Kind), vName, clientObject)
		if pName.Name != "" {
			nameMapping := synccontext.NameMapping{
				GroupVersionKind: gvk,
				VirtualName:      vName,
				HostName:         pName,
			}

			err = ctx.Mappings.Store().AddReferenceAndSave(ctx, nameMapping, nameMapping)
			if err != nil {
				klog.FromContext(ctx).Error(err, "saving reference in store", "mapping", nameMapping.String())
			}
		}
	}

	return n.Mapper.Migrate(ctx, mapper)
}

func listObjects(ctx *synccontext.RegisterContext, kubeClient client.Client, listGvk schema.GroupVersionKind) ([]runtime.Object, error) {
	list, err := scheme.Scheme.New(listGvk)
	if err != nil {
		if !runtime.IsNotRegisteredError(err) {
			return nil, fmt.Errorf("migrate create object list %s: %w", listGvk.String(), err)
		}

		list = &unstructured.UnstructuredList{}
	}

	uList, ok := list.(*unstructured.UnstructuredList)
	if ok {
		uList.SetKind(listGvk.Kind)
		uList.SetAPIVersion(listGvk.GroupVersion().String())
	}

	// it's safe to list here without namespace as this will just list all items in the cache
	err = kubeClient.List(ctx, list.(client.ObjectList))
	if err != nil {
		return nil, fmt.Errorf("error listing %s: %w", listGvk.String(), err)
	}

	items, err := meta.ExtractList(list)
	if err != nil {
		return nil, fmt.Errorf("extract list %s: %w", listGvk.String(), err)
	}

	return items, nil
}

func (n *recorder) VirtualToHost(ctx *synccontext.SyncContext, req types.NamespacedName, vObj client.Object) (retName types.NamespacedName) {
	defer func() {
		err := RecordMapping(ctx, retName, req, n.GroupVersionKind())
		if err != nil {
			klog.FromContext(ctx).Error(err, "record name mapping", "host", retName, "virtual", req)
			retName = types.NamespacedName{}
		}
	}()

	// check store first
	pName, ok := VirtualToHostFromStore(ctx, req, n.GroupVersionKind())
	if ok {
		return pName
	}

	return n.Mapper.VirtualToHost(ctx, req, vObj)
}

func (n *recorder) HostToVirtual(ctx *synccontext.SyncContext, req types.NamespacedName, pObj client.Object) (retName types.NamespacedName) {
	defer func() {
		err := RecordMapping(ctx, req, retName, n.GroupVersionKind())
		if err != nil {
			klog.FromContext(ctx).Error(err, "record name mapping", "host", req, "virtual", retName)
			retName = types.NamespacedName{}
		}
	}()

	// check store first
	vName, ok := HostToVirtualFromStore(ctx, req, n.GroupVersionKind())
	if ok {
		return vName
	}

	return n.Mapper.HostToVirtual(ctx, req, pObj)
}

func (n *recorder) IsManaged(ctx *synccontext.SyncContext, pObj client.Object) (bool, error) {
	if ctx != nil && ctx.Mappings != nil && ctx.Mappings.Store() != nil {
		_, ok := ctx.Mappings.Store().HostToVirtualName(ctx, synccontext.Object{
			GroupVersionKind: n.GroupVersionKind(),
			NamespacedName: types.NamespacedName{
				Name:      pObj.GetName(),
				Namespace: pObj.GetNamespace(),
			},
		})
		if ok {
			return true, nil
		}
	}

	return n.Mapper.IsManaged(ctx, pObj)
}

func RecordMapping(ctx *synccontext.SyncContext, pName, vName types.NamespacedName, gvk schema.GroupVersionKind) error {
	if pName.Name == "" || vName.Name == "" {
		return nil
	}

	if ctx != nil && ctx.Mappings != nil && ctx.Mappings.Store() != nil {
		// check if we have the owning object in the context
		belongsTo, ok := synccontext.MappingFrom(ctx)
		if !ok {
			return nil
		}

		// record the reference
		err := ctx.Mappings.Store().AddReferenceAndSave(ctx, synccontext.NameMapping{
			GroupVersionKind: gvk,

			HostName:    pName,
			VirtualName: vName,
		}, belongsTo)
		if err != nil {
			return err
		}
	}

	return nil
}

func HostToVirtualFromStore(ctx *synccontext.SyncContext, req types.NamespacedName, gvk schema.GroupVersionKind) (types.NamespacedName, bool) {
	if ctx == nil || ctx.Mappings == nil || ctx.Mappings.Store() == nil {
		return types.NamespacedName{}, false
	}

	return ctx.Mappings.Store().HostToVirtualName(ctx, synccontext.Object{
		GroupVersionKind: gvk,
		NamespacedName:   req,
	})
}

func VirtualToHostFromStore(ctx *synccontext.SyncContext, req types.NamespacedName, gvk schema.GroupVersionKind) (types.NamespacedName, bool) {
	if ctx == nil || ctx.Mappings == nil || ctx.Mappings.Store() == nil {
		return types.NamespacedName{}, false
	}

	return ctx.Mappings.Store().VirtualToHostName(ctx, synccontext.Object{
		GroupVersionKind: gvk,
		NamespacedName:   req,
	})
}
