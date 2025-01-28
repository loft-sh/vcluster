package configmaps

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncer "github.com/loft-sh/vcluster/pkg/syncer/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type fromHostTranslate struct {
	gvk           schema.GroupVersionKind
	eventRecorder record.EventRecorder
	virtualToHost map[string]string
	skipFuncs     []skipHostObject
}

func NewConfigMapFromHostTranslate(ctx *synccontext.RegisterContext) (syncer.GenericTranslator, error) {
	gvk, err := apiutil.GVKForObject(&corev1.ConfigMap{}, scheme.Scheme)
	if err != nil {
		return nil, fmt.Errorf("retrieve GVK for object failed: %w", err)
	}
	hostToVirtual := ctx.Config.Sync.FromHost.ConfigMaps.Selector.Mappings
	virtualToHost := make(map[string]string, len(ctx.Config.Sync.FromHost.ConfigMaps.Selector.Mappings))
	for host, virtual := range hostToVirtual {
		virtualToHost[virtual] = host
	}

	return &fromHostTranslate{
		gvk:           gvk,
		eventRecorder: ctx.VirtualManager.GetEventRecorderFor("from-host-configmaps-syncer"),
		virtualToHost: virtualToHost,
		skipFuncs:     []skipHostObject{skipKubeRootCaConfigMap},
	}, nil
}

func (c *fromHostTranslate) Name() string {
	return "configmap-from-host-translator"
}

func (c *fromHostTranslate) Resource() client.Object {
	return &corev1.ConfigMap{}
}

func (c *fromHostTranslate) Migrate(_ *synccontext.RegisterContext, _ synccontext.Mapper) error {
	return nil
}

func (c *fromHostTranslate) GroupVersionKind() schema.GroupVersionKind {
	return c.gvk
}

func (c *fromHostTranslate) VirtualToHost(ctx *synccontext.SyncContext, req types.NamespacedName, _ client.Object) types.NamespacedName {
	vName, vNs := req.Name, req.Namespace
	nn, _ := matchesVirtualObject(vNs, vName, c.virtualToHost, ctx.Config.ControlPlaneNamespace)
	return nn
}

func (c *fromHostTranslate) HostToVirtual(ctx *synccontext.SyncContext, req types.NamespacedName, _ client.Object) types.NamespacedName {
	nn, ok := matchesHostObject(req.Name, req.Namespace, ctx.Config.Sync.FromHost.ConfigMaps.Selector.Mappings, ctx.Config.ControlPlaneNamespace, c.skipFuncs...)
	if !ok {
		return types.NamespacedName{}
	}
	return nn
}

func (c *fromHostTranslate) IsManaged(ctx *synccontext.SyncContext, pObj client.Object) (bool, error) {
	hostName, hostNs := pObj.GetName(), pObj.GetNamespace()
	_, managed := matchesHostObject(hostName, hostNs, ctx.Config.Sync.FromHost.ConfigMaps.Selector.Mappings, ctx.Config.ControlPlaneNamespace, c.skipFuncs...)
	return managed, nil
}

func (c *fromHostTranslate) EventRecorder() record.EventRecorder {
	return c.eventRecorder
}
