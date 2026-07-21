package translate

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/base36"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

var _ Translator = &singleNamespace{}

func NewSingleNamespaceTranslator(targetNamespace string) Translator {
	return &singleNamespace{
		targetNamespace: targetNamespace,
	}
}

type singleNamespace struct {
	targetNamespace string
}

func (s *singleNamespace) SingleNamespaceTarget() bool {
	return true
}

func (s *singleNamespace) HostName(ctx *synccontext.SyncContext, vName, vNamespace string) types.NamespacedName {
	if vName == "" {
		return types.NamespacedName{}
	}

	return types.NamespacedName{
		Name:      SingleNamespaceHostName(vName, vNamespace, VClusterName),
		Namespace: s.HostNamespace(ctx, vNamespace),
	}
}

func (s *singleNamespace) HostNameShort(ctx *synccontext.SyncContext, vName, vNamespace string) types.NamespacedName {
	if vName == "" {
		return types.NamespacedName{}
	}

	// we use base36 to avoid as much conflicts as possible
	digest := sha256.Sum256([]byte(strings.Join([]string{vName, "x", vNamespace, "x", VClusterName}, "-")))
	return types.NamespacedName{
		Name:      "v" + base36.EncodeBytes(digest[:])[0:13], // needs to start with a character for certain objects (e.g. services)
		Namespace: s.HostNamespace(ctx, vNamespace),
	}
}

func SingleNamespaceHostName(name, namespace, suffix string) string {
	if name == "" {
		return ""
	}
	return SafeConcatName(name, "x", namespace, "x", suffix)
}

func (s *singleNamespace) HostNameCluster(name string) string {
	if name == "" {
		return ""
	}
	return SafeConcatName("vcluster", name, "x", s.targetNamespace, "x", VClusterName)
}

func (s *singleNamespace) MarkerLabelCluster() string {
	return SafeConcatName(s.targetNamespace, "x", VClusterName)
}

func (s *singleNamespace) IsManaged(ctx *synccontext.SyncContext, pObj client.Object) bool {
	// check if cluster scoped object
	if pObj.GetNamespace() == "" {
		return pObj.GetLabels()[MarkerLabel] == s.MarkerLabelCluster()
	}

	// is object not in our target namespace?
	if !s.IsTargetedNamespace(ctx, pObj.GetNamespace()) {
		return false
	}

	// if host namespace is mapped, we don't check for marker label
	if pObj.GetLabels()[MarkerLabel] != VClusterName {
		return false
	}

	// vcluster has not synced the object IF:
	// If object-name annotation is not set OR
	// If object-name annotation is different from actual name
	gvk, err := apiutil.GVKForObject(pObj, scheme.Scheme)
	if err == nil {
		// check if the name annotation is correct
		if pObj.GetAnnotations()[NameAnnotation] == "" {
			return false
		} else if ctx.Mappings != nil && ctx.Mappings.Has(gvk) && pObj.GetName() != mappings.VirtualToHostName(ctx, pObj.GetAnnotations()[NameAnnotation], pObj.GetAnnotations()[NamespaceAnnotation], gvk) {
			klog.FromContext(ctx).V(1).Info("Host object doesn't match, because name annotations is wrong",
				"object", pObj.GetName(),
				"kind", gvk.String(),
				"existingName", pObj.GetName(),
				"expectedName", mappings.VirtualToHostName(ctx, pObj.GetAnnotations()[NameAnnotation], pObj.GetAnnotations()[NamespaceAnnotation], gvk),
				"nameAnnotation", pObj.GetAnnotations()[NamespaceAnnotation]+"/"+pObj.GetAnnotations()[NameAnnotation],
			)

			return false
		}

		// if kind doesn't match vCluster has probably not synced the object
		if pObj.GetAnnotations()[KindAnnotation] != "" && gvk.String() != pObj.GetAnnotations()[KindAnnotation] {
			klog.FromContext(ctx).V(1).Info("Host object doesn't match, because kind annotations is wrong",
				"object", pObj.GetName(),
				"existingKind", gvk.String(),
				"expectedKind", pObj.GetAnnotations()[KindAnnotation],
			)
			return false
		}
	}

	// check if host name / namespace matches actual name / namespace
	if pObj.GetAnnotations()[HostNameAnnotation] != "" && pObj.GetAnnotations()[HostNameAnnotation] != pObj.GetName() {
		return false
	} else if pObj.GetAnnotations()[HostNamespaceAnnotation] != "" && pObj.GetAnnotations()[HostNamespaceAnnotation] != pObj.GetNamespace() {
		return false
	}

	return true
}

func (s *singleNamespace) IsTargetedNamespace(_ *synccontext.SyncContext, pNamespace string) bool {
	return pNamespace == s.targetNamespace
}

func (s *singleNamespace) LabelsToTranslate() map[string]bool {
	return map[string]bool{
		// rewrite release
		VClusterReleaseLabel: true,

		// namespace, marker & controlled-by
		NamespaceLabel:  true,
		MarkerLabel:     true,
		ControllerLabel: true,
	}
}

func (s *singleNamespace) HostNamespace(_ *synccontext.SyncContext, vNamespace string) string {
	if vNamespace == "" {
		return ""
	}

	return s.targetNamespace
}

func HostLabelNamespace(key string) string {
	return convertLabelKeyWithPrefix(NamespaceLabelPrefix, key)
}

func HostLabelSelectorNamespace(labelSelector *metav1.LabelSelector) *metav1.LabelSelector {
	return hostLabelSelector(labelSelector, func(key string) string {
		return HostLabelNamespace(key)
	})
}

func convertLabelKeyWithPrefix(prefix, key string) string {
	digest := sha256.Sum256([]byte(key))
	return SafeConcatName(prefix, VClusterName, "x", hex.EncodeToString(digest[0:])[0:10])
}
