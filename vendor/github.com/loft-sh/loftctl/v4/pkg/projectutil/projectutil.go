package projectutil

import (
	"strings"
	"sync"
)

var DefaultProjectNamespacePrefix = "loft-p-"

// here a nil value means the prefix is unset and things should panic
var prefix *string
var prefixMux sync.RWMutex

// SetProjectNamespacePrefix sets the global project namespace prefix
// Defaulting should be handled when reading the config via ParseProjectNamespacePrefix
func SetProjectNamespacePrefix(newPrefix string) {
	prefixMux.Lock()
	defer prefixMux.Unlock()
	prefix = &newPrefix
}

// ParseConfiguredProjectNSPrefix handles the defaulting for a configured prefix and returns the prefix to be used
func ParseConfiguredProjectNSPrefix(configuredPrefix *string) string {
	if configuredPrefix == nil {
		return DefaultProjectNamespacePrefix
	}
	return *configuredPrefix
}

// ProjectFromNamespace returns the project associated with the namespace
func ProjectFromNamespace(namespace string) string {
	prefixMux.RLock()
	defer prefixMux.RUnlock()
	return strings.TrimPrefix(namespace, *prefix)
}

// ProjectNamespace returns the namespace associated with the project
func ProjectNamespace(projectName string) string {
	prefixMux.RLock()
	defer prefixMux.RUnlock()
	return *prefix + projectName
}
