package configmaps

import (
	"strings"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/cache"
)

func parseHostNamespacesFromMappings(mappings map[string]string, vClusterNs string) []string {
	ret := make([]string, 0)
	for host := range mappings {
		if host == "*" {
			ret = append(ret, vClusterNs)
		}
		parts := strings.Split(host, "/")
		if len(parts) != 2 {
			continue
		}
		hostNs := parts[0]
		ret = append(ret, hostNs)
	}
	return ret
}

func getHostNamespacesAndConfig(mappings map[string]string, vClusterNs string) map[string]cache.Config {
	namespaces := parseHostNamespacesFromMappings(mappings, vClusterNs)
	ret := make(map[string]cache.Config, len(namespaces))
	for _, ns := range namespaces {
		ret[ns] = cache.Config{}
	}
	return ret
}

type skipHostObject func(hostName, hostNamespace string) bool

func skipKubeRootCaConfigMap(hostName, _ string) bool {
	return hostName == "kube-root-ca.crt"
}

func matchesHostObject(hostName, hostNamespace string, resourceMappings map[string]string, vClusterHostNamespace string, skippers ...skipHostObject) (types.NamespacedName, bool) {
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
