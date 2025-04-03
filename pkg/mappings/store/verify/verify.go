package verify

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/mappings/store"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
)

const (
	SkipHostNamespaceCheck = "mappings.store.verify.SkipHostNamespaceCheck"
)

func NewVerifyMapping() store.VerifyMapping {
	return func(ctx context.Context, mapping synccontext.NameMapping) bool {
		return CheckHostObject(ctx, mapping.Host())
	}
}

func CheckHostObject(ctx context.Context, hostObject synccontext.Object) bool {
	skipHostNamespaceCheck, ok := ctx.Value(SkipHostNamespaceCheck).(bool)
	if ok && skipHostNamespaceCheck {
		return true
	}

	// we don't allow mappings that are not within targeted namespaces
	if hostObject.Namespace != "" && !translate.Default.IsTargetedNamespace(hostObject.Namespace) {
		return false
	}

	// we don't allow namespace mappings that are not within targeted namespaces
	if hostObject.GroupVersionKind.String() == corev1.SchemeGroupVersion.WithKind("Namespace").String() && !translate.Default.IsTargetedNamespace(hostObject.Name) {
		return false
	}

	return true
}
