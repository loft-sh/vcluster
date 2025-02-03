package translator

import (
	"strings"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncer "github.com/loft-sh/vcluster/pkg/syncer/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type fromHostTranslate struct {
	gvk            schema.GroupVersionKind
	eventRecorder  record.EventRecorder
	virtualToHost  map[string]string
	hostToVirtual  map[string]string
	namespace      string
	translatorName string
	skipFuncs      []ShouldSkipHostObjectFunc
}

type ShouldSkipHostObjectFunc func(hostName, hostNamespace string) bool

func NewFromHostTranslatorForGVK(ctx *synccontext.RegisterContext, gvk schema.GroupVersionKind, mappings map[string]string, skipFuncs ...ShouldSkipHostObjectFunc) (syncer.FromConfigTranslator, error) {
	hostToVirtual := mappings
	virtualToHost := make(map[string]string, len(mappings))
	for host, virtual := range hostToVirtual {
		virtualToHost[virtual] = host
	}

	return &fromHostTranslate{
		gvk:            gvk,
		eventRecorder:  ctx.VirtualManager.GetEventRecorderFor("from-host-" + strings.ToLower(gvk.Kind) + "-syncer"),
		virtualToHost:  virtualToHost,
		hostToVirtual:  hostToVirtual,
		namespace:      ctx.Config.ControlPlaneNamespace,
		translatorName: strings.ToLower(gvk.Kind) + "from-host-translator",
		skipFuncs:      skipFuncs,
	}, nil
}

func (c *fromHostTranslate) Name() string {
	return c.translatorName
}

func (c *fromHostTranslate) Resource() client.Object {
	switch c.gvk.Kind {
	case "ConfigMap":
		return &corev1.ConfigMap{}
	case "Secret":
		return &corev1.Secret{}
	default:
		return nil
	}
}

func (c *fromHostTranslate) Migrate(_ *synccontext.RegisterContext, _ synccontext.Mapper) error {
	return nil
}

func (c *fromHostTranslate) GroupVersionKind() schema.GroupVersionKind {
	return c.gvk
}

func (c *fromHostTranslate) VirtualToHost(ctx *synccontext.SyncContext, req types.NamespacedName, _ client.Object) types.NamespacedName {
	vName, vNs := req.Name, req.Namespace
	nn, _ := matchesVirtualObject(vNs, vName, c.virtualToHost, c.namespace)
	return nn
}

func (c *fromHostTranslate) HostToVirtual(ctx *synccontext.SyncContext, req types.NamespacedName, _ client.Object) types.NamespacedName {
	nn, ok := matchesHostObject(req.Name, req.Namespace, c.hostToVirtual, c.namespace, c.skipFuncs...)
	if !ok {
		return types.NamespacedName{}
	}
	return nn
}

func (c *fromHostTranslate) IsManaged(ctx *synccontext.SyncContext, pObj client.Object) (bool, error) {
	hostName, hostNs := pObj.GetName(), pObj.GetNamespace()
	_, managed := matchesHostObject(hostName, hostNs, c.hostToVirtual, c.namespace, c.skipFuncs...)
	return managed, nil
}

func (c *fromHostTranslate) EventRecorder() record.EventRecorder {
	return c.eventRecorder
}

func (c *fromHostTranslate) MatchesHostObject(hostName, hostNamespace string) (types.NamespacedName, bool) {
	return matchesHostObject(hostName, hostNamespace, c.hostToVirtual, c.namespace, c.skipFuncs...)
}
func (c *fromHostTranslate) MatchesVirtualObject(virtualName, virtualNamespace string) (types.NamespacedName, bool) {
	return matchesVirtualObject(virtualNamespace, virtualName, c.virtualToHost, c.namespace)
}

func matchesHostObject(hostName, hostNamespace string, resourceMappings map[string]string, vClusterHostNamespace string, skippers ...ShouldSkipHostObjectFunc) (types.NamespacedName, bool) {
	for _, skipFunc := range skippers {
		if skipFunc(hostName, hostNamespace) {
			return types.NamespacedName{}, false
		}
	}

	key := hostNamespace + "/" + hostName
	matchesAllKeyInNamespaceKey := hostNamespace + "/*"
	matchesAllKey := "*"

	// first, let's try matching by namespace/name
	if virtual, ok := resourceMappings[key]; ok {
		virtualParts := strings.Split(virtual, "/")
		if len(virtualParts) == 2 {
			ns := virtualParts[0]
			name := virtualParts[1]
			if name == "*" {
				name = hostName
			}
			return types.NamespacedName{Namespace: ns, Name: name}, true
		}
	}

	// second, by namespace/*
	if virtual, ok := resourceMappings[matchesAllKeyInNamespaceKey]; ok {
		virtualParts := strings.Split(virtual, "/")
		if len(virtualParts) == 2 {
			ns := virtualParts[0]
			return types.NamespacedName{Namespace: ns, Name: hostName}, true
		}
	}

	// last chance, if user specified "*": <namespace>/*
	if virtual, ok := resourceMappings[matchesAllKey]; ok {
		if vClusterHostNamespace == hostNamespace {
			virtualParts := strings.Split(virtual, "/")
			if len(virtualParts) == 2 {
				return types.NamespacedName{Namespace: virtualParts[0], Name: hostName}, true
			}
		}
	}
	return types.NamespacedName{}, false
}

func matchesVirtualObject(virtualNs, virtualName string, virtualToHost map[string]string, vClusterHostNamespace string) (types.NamespacedName, bool) {
	virtualKey := virtualNs + "/" + virtualName
	virtualAllInNamespaceKey := virtualNs + "/*"

	// let's check if object is listed explicitly
	if host, ok := virtualToHost[virtualKey]; ok {
		if host == "*" {
			return types.NamespacedName{Namespace: vClusterHostNamespace, Name: virtualName}, false
		}
		hostParts := strings.Split(host, "/")
		if len(hostParts) == 2 {
			return types.NamespacedName{Namespace: hostParts[0], Name: hostParts[1]}, true
		}
	}

	// check if object's namespace is listed
	if host, ok := virtualToHost[virtualAllInNamespaceKey]; ok {
		hostParts := strings.Split(host, "/")
		if len(hostParts) == 2 {
			return types.NamespacedName{Namespace: hostParts[0], Name: virtualName}, true
		}
	}
	return types.NamespacedName{}, false
}
