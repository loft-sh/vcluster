package generic

import (
	"context"
	"regexp"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/patches"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type importPatcher struct {
	config        *vclusterconfig.Import
	virtualClient client.Client
}

var _ ObjectPatcher = &importPatcher{}

func (s *importPatcher) ServerSideApply(ctx context.Context, _, destObj, sourceObj client.Object) error {
	return patches.ApplyPatches(destObj, sourceObj, s.config.Patches, s.config.ReversePatches, &hostToVirtualImportNameResolver{virtualClient: s.virtualClient, ctx: ctx})
}

func (s *importPatcher) ReverseUpdate(_ context.Context, destObj, sourceObj client.Object) error {
	return patches.ApplyPatches(destObj, sourceObj, s.config.ReversePatches, nil, &virtualToHostNameResolver{namespace: sourceObj.GetNamespace()})
}

type hostToVirtualImportNameResolver struct {
	virtualClient client.Client
	ctx           context.Context
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
	vNamespace := (&corev1.Namespace{}).DeepCopyObject().(client.Object)
	err := clienthelper.GetByIndex(r.ctx, r.virtualClient, vNamespace, constants.IndexByPhysicalName, namespace)
	if err != nil {
		return "", err
	}
	return vNamespace.GetName(), nil
}
