package generic

import (
	"context"
	"fmt"
	"regexp"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/patches"
	patchesregex "github.com/loft-sh/vcluster/pkg/patches/regex"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type exportPatcher struct {
	config *vclusterconfig.Export
	gvk    schema.GroupVersionKind
}

var _ ObjectPatcher = &exportPatcher{}

func (e *exportPatcher) ServerSideApply(_ context.Context, fromObj, destObj, sourceObj client.Object) error {
	return patches.ApplyPatches(destObj, sourceObj, e.config.Patches, e.config.ReversePatches, &virtualToHostNameResolver{
		namespace:       fromObj.GetNamespace(),
		targetNamespace: translate.Default.PhysicalNamespace(fromObj.GetNamespace()),
	})
}

func (e *exportPatcher) ReverseUpdate(_ context.Context, destObj, sourceObj client.Object) error {
	return patches.ApplyPatches(destObj, sourceObj, e.config.ReversePatches, nil, &hostToVirtualNameResolver{
		gvk:  e.gvk,
		pObj: sourceObj,
	})
}

type virtualToHostNameResolver struct {
	namespace       string
	targetNamespace string
}

func (r *virtualToHostNameResolver) TranslateName(name string, regex *regexp.Regexp, _ string) (string, error) {
	return r.TranslateNameWithNamespace(name, r.namespace, regex, "")
}

func (r *virtualToHostNameResolver) TranslateNameWithNamespace(name string, namespace string, regex *regexp.Regexp, _ string) (string, error) {
	if regex != nil {
		return patchesregex.ProcessRegex(regex, name, func(name, ns string) types.NamespacedName {
			// if the regex match doesn't contain namespace - use the namespace set in this resolver
			if ns == "" {
				ns = namespace
			}

			return types.NamespacedName{
				Namespace: translate.Default.PhysicalNamespace(namespace),
				Name:      translate.Default.PhysicalName(name, ns),
			}
		}), nil
	}

	return translate.Default.PhysicalName(name, namespace), nil
}

func (r *virtualToHostNameResolver) TranslateLabelExpressionsSelector(selector *metav1.LabelSelector) (*metav1.LabelSelector, error) {
	return translate.Default.TranslateLabelSelectorCluster(selector), nil
}

func (r *virtualToHostNameResolver) TranslateLabelKey(key string) (string, error) {
	return translate.Default.ConvertLabelKey(key), nil
}

func (r *virtualToHostNameResolver) TranslateLabelSelector(selector map[string]string) (map[string]string, error) {
	labelSelector := &metav1.LabelSelector{
		MatchLabels: selector,
	}

	return metav1.LabelSelectorAsMap(
		translate.Default.TranslateLabelSelector(labelSelector))
}

func (r *virtualToHostNameResolver) TranslateNamespaceRef(namespace string) (string, error) {
	return translate.Default.PhysicalNamespace(namespace), nil
}

func validateExportConfig(config *vclusterconfig.Export) error {
	for _, p := range append(config.Patches, config.ReversePatches...) {
		if p.Regex != "" {
			parsed, err := patchesregex.PrepareRegex(p.Regex)
			if err != nil {
				return fmt.Errorf("invalid Regex: %w", err)
			}
			p.ParsedRegex = parsed
		}
	}
	return nil
}

type hostToVirtualNameResolver struct {
	pObj client.Object
	gvk  schema.GroupVersionKind
}

func (r *hostToVirtualNameResolver) TranslateName(string, *regexp.Regexp, string) (string, error) {
	return "", fmt.Errorf("translation not supported from host to virtual object")
}

func (r *hostToVirtualNameResolver) TranslateNameWithNamespace(string, string, *regexp.Regexp, string) (string, error) {
	return "", fmt.Errorf("translation not supported from host to virtual object")
}

func (r *hostToVirtualNameResolver) TranslateLabelKey(string) (string, error) {
	return "", fmt.Errorf("translation not supported from host to virtual object")
}

func (r *hostToVirtualNameResolver) TranslateLabelExpressionsSelector(*metav1.LabelSelector) (*metav1.LabelSelector, error) {
	return nil, fmt.Errorf("translation not supported from host to virtual object")
}

func (r *hostToVirtualNameResolver) TranslateLabelSelector(map[string]string) (map[string]string, error) {
	return nil, fmt.Errorf("translation not supported from host to virtual object")
}

func (r *hostToVirtualNameResolver) TranslateNamespaceRef(string) (string, error) {
	return "", fmt.Errorf("translation not supported from host to virtual object")
}
