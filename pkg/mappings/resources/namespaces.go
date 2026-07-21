package resources

import (
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateNamespacesMapper(ctx *synccontext.RegisterContext) (synccontext.Mapper, error) {
	singleNamespaceMapper := &singleNamespaceModeMapper{
		targetNamespace: ctx.Config.HostNamespace,
	}
	if ctx.Config.Sync.ToHost.Namespaces.Enabled {
		return pro.GetNamespaceMapper(ctx, singleNamespaceMapper)
	}
	return singleNamespaceMapper, nil
}

type singleNamespaceModeMapper struct {
	targetNamespace string
}

func (s *singleNamespaceModeMapper) Migrate(_ *synccontext.RegisterContext, _ synccontext.Mapper) error {
	return nil
}

func (s *singleNamespaceModeMapper) GroupVersionKind() schema.GroupVersionKind {
	return corev1.SchemeGroupVersion.WithKind("Namespace")
}

func (s *singleNamespaceModeMapper) VirtualToHost(ctx *synccontext.SyncContext, req types.NamespacedName, _ client.Object) types.NamespacedName {
	return types.NamespacedName{Name: translate.Default.HostNamespace(ctx, req.Name)}
}

func (s *singleNamespaceModeMapper) HostToVirtual(_ *synccontext.SyncContext, _ types.NamespacedName, _ client.Object) types.NamespacedName {
	return types.NamespacedName{}
}

func (s *singleNamespaceModeMapper) IsManaged(_ *synccontext.SyncContext, _ client.Object) (bool, error) {
	return false, nil
}
