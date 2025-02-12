package translator

import (
	"strings"

	"github.com/loft-sh/vcluster/pkg/constants"

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

// ShouldSkipHostObjectFunc takes object's host name and namespace
// and returns true if object should be skipped in the from host sync.
type ShouldSkipHostObjectFunc func(hostName, hostNamespace string) bool

func NewFromHostTranslatorForGVK(ctx *synccontext.RegisterContext, gvk schema.GroupVersionKind, hostToVirtual map[string]string, skipFuncs ...ShouldSkipHostObjectFunc) (syncer.FromConfigTranslator, error) {
	virtualToHost := make(map[string]string, len(hostToVirtual))
	for host, virtual := range hostToVirtual {
		virtualToHost[virtual] = host
	}

	return &fromHostTranslate{
		gvk:            gvk,
		eventRecorder:  ctx.VirtualManager.GetEventRecorderFor("from-host-" + strings.ToLower(gvk.Kind) + "-syncer"),
		virtualToHost:  virtualToHost,
		hostToVirtual:  hostToVirtual,
		namespace:      ctx.Config.ControlPlaneNamespace,
		translatorName: "from-host-" + strings.ToLower(gvk.Kind),
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

func (c *fromHostTranslate) VirtualToHost(_ *synccontext.SyncContext, req types.NamespacedName, _ client.Object) types.NamespacedName {
	vName, vNs := req.Name, req.Namespace
	nn, _ := matchesVirtualObject(vNs, vName, c.virtualToHost, c.namespace)
	return nn
}

func (c *fromHostTranslate) HostToVirtual(_ *synccontext.SyncContext, req types.NamespacedName, _ client.Object) types.NamespacedName {
	nn, ok := matchesHostObject(req.Name, req.Namespace, c.hostToVirtual, c.namespace, c.skipFuncs...)
	if !ok {
		return types.NamespacedName{}
	}
	return nn
}

func (c *fromHostTranslate) IsManaged(_ *synccontext.SyncContext, pObj client.Object) (bool, error) {
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

	// first, let's try matching by namespace/name
	if virtual, ok := resourceMappings[key]; ok {
		ns, name, found := strings.Cut(virtual, "/")
		if found && name != "" {
			if name == "*" {
				name = hostName
			}
			return types.NamespacedName{Namespace: ns, Name: name}, true
		}
	}

	// second, by namespace/*
	if virtual, ok := resourceMappings[matchesAllKeyInNamespaceKey]; ok {
		ns, _, found := strings.Cut(virtual, "/")
		if found {
			return types.NamespacedName{Namespace: ns, Name: hostName}, true
		}
	}

	// last chance, if user specified "": <namespace>/*
	if virtual, ok := resourceMappings[constants.VClusterNamespaceInHostMappingSpecialCharacter]; ok {
		if vClusterHostNamespace == hostNamespace {
			ns, name, found := strings.Cut(virtual, "/")
			if found && name != "" {
				return types.NamespacedName{Namespace: ns, Name: hostName}, true
			} else if !strings.Contains(virtual, "/") {
				// then the mapping is "": "virtual-namespace" where "" means vCluster host namespace
				// in this case, we want to return virtual-namespace/hostName
				return types.NamespacedName{Namespace: virtual, Name: hostName}, true
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
		ns, name, found := strings.Cut(host, "/")
		if found && name != "" {
			return types.NamespacedName{Namespace: ns, Name: name}, true
		}
	}

	// check if object's namespace is listed
	if host, ok := virtualToHost[virtualAllInNamespaceKey]; ok {
		ns, _, found := strings.Cut(host, "/")
		if found {
			return types.NamespacedName{Namespace: ns, Name: virtualName}, true
		}
	}

	// check if object's namespace is a target namespace for vCluster host namespace,
	// then return vCluster host namespace + object name
	if host, ok := virtualToHost[virtualNs]; ok && host == constants.VClusterNamespaceInHostMappingSpecialCharacter {
		return types.NamespacedName{Namespace: vClusterHostNamespace, Name: virtualName}, true
	}

	return types.NamespacedName{}, false
}
