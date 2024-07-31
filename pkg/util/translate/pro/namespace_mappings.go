package pro

import (
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"sigs.k8s.io/controller-runtime/pkg/cache"
)

var HostNamespaceMatchesMapping = func(_ *synccontext.SyncContext, _ string) (string, bool) {
	return "", false
}

var VirtualNamespaceMatchesMapping = func(_ *synccontext.SyncContext, _ string) (string, bool) {
	return "", false
}

var AddMappingsToCache = func(_ map[string]cache.Config) {

}
