package generic

import (
	"regexp"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patches"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type importPatcher struct {
	config        *vclusterconfig.Import
	virtualClient client.Client
}

var _ ObjectPatcher = &importPatcher{}

func (s *importPatcher) ServerSideApply(ctx *synccontext.SyncContext, _, destObj, sourceObj client.Object) error {
	return patches.ApplyPatches(destObj, sourceObj, s.config.Patches, s.config.ReversePatches, &hostToVirtualImportNameResolver{
		syncContext: ctx,
	})
}

func (s *importPatcher) ReverseUpdate(ctx *synccontext.SyncContext, destObj, sourceObj client.Object) error {
	return patches.ApplyPatches(destObj, sourceObj, s.config.ReversePatches, nil, &virtualToHostNameResolver{
		syncContext: ctx,
		namespace:   sourceObj.GetNamespace(),
	})
}

type hostToVirtualImportNameResolver struct {
	syncContext *synccontext.SyncContext
}

func (r *hostToVirtualImportNameResolver) TranslateName(name string, _ *regexp.Regexp, _ string) (string, error) {
	return name, nil
}

func (r *hostToVirtualImportNameResolver) TranslateNameWithNamespace(name string, _ string, _ *regexp.Regexp, _ string) (string, error) {
	return name, nil
}

func (r *hostToVirtualImportNameResolver) TranslateLabelKey(key string) (string, error) {
	return key, nil
}

func (r *hostToVirtualImportNameResolver) TranslateLabelExpressionsSelector(selector *metav1.LabelSelector) (*metav1.LabelSelector, error) {
	return selector, nil
}

func (r *hostToVirtualImportNameResolver) TranslateLabelSelector(selector map[string]string) (map[string]string, error) {
	return selector, nil
}

func (r *hostToVirtualImportNameResolver) TranslateNamespaceRef(namespace string) (string, error) {
	vNamespace := mappings.HostToVirtual(r.syncContext, namespace, "", nil, mappings.Namespaces())
	return vNamespace.Name, nil
}
