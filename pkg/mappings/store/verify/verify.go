package verify

import (
	"github.com/loft-sh/vcluster/pkg/mappings/store"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
)

func NewVerifyMapping(ctx *synccontext.SyncContext) store.VerifyMapping {
	return func(mapping synccontext.NameMapping) bool {
		return CheckHostObject(ctx, mapping.Host())
	}
}

func CheckHostObject(ctx *synccontext.SyncContext, hostObject synccontext.Object) bool {
	// we don't allow mappings that are not within targeted namespaces
	if hostObject.Namespace != "" && !translate.Default.IsTargetedNamespace(ctx, hostObject.Namespace) {
		return false
	}

	// we don't allow namespace mappings that are not within targeted namespaces
	if hostObject.GroupVersionKind.String() == corev1.SchemeGroupVersion.WithKind("Namespace").String() && !translate.Default.IsTargetedNamespace(ctx, hostObject.Name) {
		return false
	}

	return true
}
