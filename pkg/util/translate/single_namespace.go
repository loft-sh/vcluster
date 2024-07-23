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

func (s *singleNamespace) HostName(name, namespace string) string {
	return SingleNamespaceHostName(name, namespace, VClusterName)
}

func (s *singleNamespace) HostNameShort(name, namespace string) string {
	if name == "" {
		return ""
	}

	// we use base36 to avoid as much conflicts as possible
	digest := sha256.Sum256([]byte(strings.Join([]string{name, "x", namespace, "x", VClusterName}, "-")))
	return base36.EncodeBytes(digest[:])[0:10]
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
	if !s.IsTargetedNamespace(pObj.GetNamespace()) {
		return false
	} else if pObj.GetLabels()[MarkerLabel] != VClusterName {
		return false
	}

	// vcluster has not synced the object IF:
	// If object-name annotation is not set OR
	// If object-name annotation is different from actual name
	gvk, err := apiutil.GVKForObject(pObj, scheme.Scheme)
	if err == nil {
		// check if the name annotation is correct
		if pObj.GetAnnotations()[NameAnnotation] == "" ||
			(ctx.Mappings.Has(gvk) && pObj.GetName() != mappings.VirtualToHostName(ctx, pObj.GetAnnotations()[NameAnnotation], pObj.GetAnnotations()[NamespaceAnnotation], gvk)) {
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

	return true
}

func (s *singleNamespace) IsTargetedNamespace(ns string) bool {
	return ns == s.targetNamespace
}

func (s *singleNamespace) HostNamespace(_ string) string {
	return s.targetNamespace
}

func (s *singleNamespace) HostLabelCluster(ctx *synccontext.SyncContext, key string) string {
	if keyMatchesSyncedLabels(ctx, key) {
		return key
	}

	return hostLabelCluster(key, s.targetNamespace)
}

func (s *singleNamespace) HostLabel(ctx *synccontext.SyncContext, key string) string {
	if keyMatchesSyncedLabels(ctx, key) {
		return key
	}

	return convertLabelKeyWithPrefix(LabelPrefix, key)
}

func keyMatchesSyncedLabels(ctx *synccontext.SyncContext, key string) bool {
	if ctx != nil && ctx.Config != nil {
		for _, k := range ctx.Config.Experimental.SyncSettings.SyncLabels {
			if strings.HasSuffix(k, "*") && strings.HasPrefix(key, strings.TrimSuffix(k, "*")) {
				return true
			} else if k == key {
				return true
			}
		}
	}

	return false
}

func HostLabelNamespace(key string) string {
	return convertLabelKeyWithPrefix(NamespaceLabelPrefix, key)
}

func HostLabelSelectorNamespace(ctx *synccontext.SyncContext, labelSelector *metav1.LabelSelector) *metav1.LabelSelector {
	return hostLabelSelector(ctx, labelSelector, func(_ *synccontext.SyncContext, key string) string {
		return HostLabelNamespace(key)
	})
}

func convertLabelKeyWithPrefix(prefix, key string) string {
	digest := sha256.Sum256([]byte(key))
	return SafeConcatName(prefix, VClusterName, "x", hex.EncodeToString(digest[0:])[0:10])
}
