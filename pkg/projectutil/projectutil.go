package projectutil

import (
	"strings"
	"sync"

	"github.com/loft-sh/vcluster/pkg/util/osutil"
	"k8s.io/klog/v2"
)

// LegacyProjectNamespacePrefix is the legacy project namespace prefix
var LegacyProjectNamespacePrefix = "loft-p-"

// having a nil value means the prefix is unset and things should panic and not fail silently
var prefix *string
var prefixMux sync.RWMutex

// SetProjectNamespacePrefix sets the global project namespace prefix
// Defaulting should be handled when reading the config via ParseProjectNamespacePrefix
func SetProjectNamespacePrefix(newPrefix string) {
	prefixMux.Lock()
	defer prefixMux.Unlock()

	prefix = &newPrefix
}

func GetProjectNamespacePrefix() string {
	prefixMux.Lock()
	defer prefixMux.Unlock()

	if prefix == nil {
		klog.Errorf("Seems like you forgot to init the project namespace prefix. This is a requirement as otherwise resolving a project namespace is not possible.")
		osutil.Exit(1)
	}

	return *prefix
}

// ProjectFromNamespace returns the project associated with the namespace
func ProjectFromNamespace(namespace string) string {
	prefixMux.RLock()
	defer prefixMux.RUnlock()

	if prefix == nil {
		klog.Errorf("Seems like you forgot to init the project namespace prefix. This is a requirement as otherwise resolving a project namespace is not possible.")
		osutil.Exit(1)
	}

	return strings.TrimPrefix(namespace, *prefix)
}

// ProjectNamespace returns the namespace associated with the project
func ProjectNamespace(projectName string) string {
	prefixMux.RLock()
	defer prefixMux.RUnlock()

	if prefix == nil {
		klog.Errorf("Seems like you forgot to init the project namespace prefix. This is a requirement as otherwise resolving a project namespace is not possible.")
		osutil.Exit(1)
	}

	return *prefix + projectName
}
