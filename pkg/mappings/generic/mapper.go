package generic

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/store/verify"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

var shortNameMatcher = regexp.MustCompile(`v[a-z0-9]{13}`)

// PhysicalNameWithObjectFunc is a definition to translate a name that also optionally expects a vObj
type PhysicalNameWithObjectFunc func(ctx *synccontext.SyncContext, vName, vNamespace string, vObj client.Object) types.NamespacedName

// PhysicalNameFunc is a definition to translate a name
type PhysicalNameFunc func(ctx *synccontext.SyncContext, vName, vNamespace string) types.NamespacedName

// NewMapper creates a new mapper with a custom physical name func
func NewMapper(ctx *synccontext.RegisterContext, obj client.Object, translateName PhysicalNameFunc) (synccontext.Mapper, error) {
	return NewMapperWithObject(ctx, obj, func(ctx *synccontext.SyncContext, vName, vNamespace string, _ client.Object) types.NamespacedName {
		return translateName(ctx, vName, vNamespace)
	})
}

// NewMapperWithObject creates a new mapper with a custom physical name func
func NewMapperWithObject(ctx *synccontext.RegisterContext, obj client.Object, translateName PhysicalNameWithObjectFunc) (synccontext.Mapper, error) {
	return newMapper(ctx, obj, true, translateName)
}

// NewMapperWithoutRecorder creates a new mapper without a recorder to store mappings in the mappings store
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

func (n *mapper) Migrate(_ *synccontext.RegisterContext, _ synccontext.Mapper) error {
	return nil
}

func (n *mapper) VirtualToHost(ctx *synccontext.SyncContext, req types.NamespacedName, vObj client.Object) (retName types.NamespacedName) {
	return n.translateName(ctx, req.Name, req.Namespace, vObj)
}

func (n *mapper) HostToVirtual(ctx *synccontext.SyncContext, req types.NamespacedName, pObj client.Object) (retName types.NamespacedName) {
	if !verify.CheckHostObject(ctx, synccontext.Object{
		GroupVersionKind: n.gvk,
		NamespacedName:   req,
	}) {
		return types.NamespacedName{}
	}

	vName := TryToTranslateBackByAnnotations(ctx, req, pObj, n.gvk)
	if vName.Name != "" {
		return vName
	}

	return TryToTranslateBackByName(ctx, req, n.gvk)
}

func TryToTranslateBackByAnnotations(ctx *synccontext.SyncContext, req types.NamespacedName, pObj client.Object, objectGvk schema.GroupVersionKind) types.NamespacedName {
	if pObj == nil {
		return types.NamespacedName{}
	}

	// check if name annotation is there
	pAnnotations := pObj.GetAnnotations()
	if pAnnotations[translate.NameAnnotation] == "" {
		return types.NamespacedName{}
	}

	// exclude objects that are from other vClusters
	markerLabel := pObj.GetLabels()[translate.MarkerLabel]
	if markerLabel != "" {
		if pObj.GetNamespace() != "" && markerLabel != translate.VClusterName {
			return types.NamespacedName{}
		} else if pObj.GetNamespace() == "" && markerLabel != translate.Default.MarkerLabelCluster() {
			return types.NamespacedName{}
		}
	}

	// make sure kind matches
	gvk, ok := pAnnotations[translate.KindAnnotation]
	if ok && objectGvk.String() != gvk {
		return types.NamespacedName{}
	}

	// make sure host name matches
	pName, ok := pAnnotations[translate.HostNameAnnotation]
	if ok && pName != pObj.GetName() {
		return types.NamespacedName{}
	}

	// make sure host namespace matches
	pNamespace, ok := pAnnotations[translate.HostNamespaceAnnotation]
	if ok && pNamespace != pObj.GetNamespace() {
		return types.NamespacedName{}
	}

	klog.FromContext(ctx).V(1).Info("Translated back name/namespace via annotations method", "req", req.String(), "ret", types.NamespacedName{
		Namespace: pAnnotations[translate.NamespaceAnnotation],
		Name:      pAnnotations[translate.NameAnnotation],
	}.String())
	return types.NamespacedName{
		Namespace: pAnnotations[translate.NamespaceAnnotation],
		Name:      pAnnotations[translate.NameAnnotation],
	}
}

// TryToTranslateBackByName is used to find out the name mapping automatically in certain scenarios, this doesn't always
// work, but for some cases this is pretty useful.
func TryToTranslateBackByName(ctx *synccontext.SyncContext, req types.NamespacedName, gvk schema.GroupVersionKind) types.NamespacedName {
	if ctx == nil || ctx.Config == nil || ctx.Mappings == nil {
		return types.NamespacedName{}
	}

	// if multi-namespace mode we try to translate back
	if ctx.Config.Sync.ToHost.Namespaces.Enabled {
		if gvk == mappings.Namespaces() || !ctx.Mappings.Has(mappings.Namespaces()) {
			return types.NamespacedName{}
		}

		// get namespace mapper
		namespaceMapper, err := ctx.Mappings.ByGVK(mappings.Namespaces())
		if err != nil {
			return types.NamespacedName{}
		}

		vNamespace := namespaceMapper.HostToVirtual(ctx, types.NamespacedName{Name: req.Namespace}, nil)
		if vNamespace.Name == "" {
			return types.NamespacedName{}
		}

		klog.FromContext(ctx).V(1).Info("Translated back name/namespace via multi-namespace mode method", "req", req.String(), "ret", types.NamespacedName{
			Namespace: vNamespace.Name,
			Name:      req.Name,
		}.String())
		return types.NamespacedName{
			Namespace: vNamespace.Name,
			Name:      req.Name,
		}
	}

	// try single-namespace mode assumptions
	vName := tryToMatchHostNameShort(ctx, req)
	if vName.Name != "" {
		return vName
	}

	// try searching mapping store for host name short
	vName = tryToFindHostNameShortInStore(ctx, req)
	if vName.Name != "" {
		return vName
	}

	// try searching mapping store for host name
	vName = tryToFindHostNameInStore(ctx, req)
	if vName.Name != "" {
		return vName
	}

	return types.NamespacedName{}
}

func tryToMatchHostNameShort(ctx *synccontext.SyncContext, req types.NamespacedName) types.NamespacedName {
	// if single namespace mode and the owner object was translated via NameShort, we can try to find that name
	// within the host name and assume it's the same namespace / name
	nameMapping, ok := synccontext.MappingFrom(ctx)
	if !ok || nameMapping.VirtualName.Namespace == "" {
		return types.NamespacedName{}
	} else if translate.Default.HostNameShort(ctx, nameMapping.VirtualName.Name, nameMapping.VirtualName.Namespace).String() != nameMapping.HostName.String() {
		return types.NamespacedName{}
	}

	// test if the name is part of the host name
	if !strings.Contains(req.Name, nameMapping.HostName.Name) {
		return types.NamespacedName{}
	}

	vNamespace := nameMapping.VirtualName.Namespace
	vName := strings.ReplaceAll(req.Name, nameMapping.HostName.Name, nameMapping.VirtualName.Name)
	klog.FromContext(ctx).V(1).Info("Translated back name/namespace via single-namespace mode method", "req", req.String(), "ret", types.NamespacedName{
		Namespace: vNamespace,
		Name:      vName,
	}.String())
	return types.NamespacedName{
		Name:      vName,
		Namespace: vNamespace,
	}
}

func tryToFindHostNameInStore(ctx *synccontext.SyncContext, pName types.NamespacedName) types.NamespacedName {
	for gvk := range ctx.Mappings.List() {
		// try to find match in store
		vName, ok := ctx.Mappings.Store().HostToVirtualName(ctx, synccontext.Object{
			GroupVersionKind: gvk,
			NamespacedName:   pName,
		})
		if !ok || vName.Name == "" || vName.Namespace == "" {
			continue
		}

		// check if this is actually a HostName
		if pName.String() != translate.Default.HostName(ctx, vName.Name, vName.Namespace).String() {
			continue
		}

		// replace pName.Name with vName.Name
		return types.NamespacedName{
			Name:      vName.Name,
			Namespace: vName.Namespace,
		}
	}

	return types.NamespacedName{}
}

func tryToFindHostNameShortInStore(ctx *synccontext.SyncContext, pName types.NamespacedName) types.NamespacedName {
	// we search for strings that are exact length 10 and have no - or . in it
	matches := shortNameMatcher.FindAllString(pName.Name, -1)

	// loop over all mapping gvk's
	for _, match := range matches {
		for gvk := range ctx.Mappings.List() {
			// try to find match in store
			vName, ok := ctx.Mappings.Store().HostToVirtualName(ctx, synccontext.Object{
				GroupVersionKind: gvk,
				NamespacedName: types.NamespacedName{
					Namespace: pName.Namespace,
					Name:      match,
				},
			})
			if !ok || vName.Name == "" || vName.Namespace == "" {
				continue
			}

			// check if this is actually a HostNameShort
			if match != translate.Default.HostNameShort(ctx, vName.Name, vName.Namespace).Name {
				continue
			}

			// replace pName.Name with vName.Name
			return types.NamespacedName{
				Name:      strings.ReplaceAll(pName.Name, match, vName.Name),
				Namespace: vName.Namespace,
			}
		}
	}

	return types.NamespacedName{}
}

func (n *mapper) IsManaged(ctx *synccontext.SyncContext, pObj client.Object) (bool, error) {
	if !verify.CheckHostObject(ctx, synccontext.Object{
		GroupVersionKind: n.gvk,
		NamespacedName:   client.ObjectKeyFromObject(pObj),
	}) {
		return false, nil
	}

	return translate.Default.IsManaged(ctx, pObj), nil
}
