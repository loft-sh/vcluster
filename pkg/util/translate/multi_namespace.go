package translate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

var _ Translator = &multiNamespace{}

func NewMultiNamespaceTranslator(currentNamespace string) Translator {
	return &multiNamespace{
		currentNamespace: currentNamespace,
	}
}

type multiNamespace struct {
	currentNamespace string
}

func (s *multiNamespace) SingleNamespaceTarget() bool {
	return false
}

// HostName returns the physical name of the name / namespace resource
func (s *multiNamespace) HostName(ctx *synccontext.SyncContext, name, namespace string) types.NamespacedName {
	return types.NamespacedName{
		Name:      name,
		Namespace: s.HostNamespace(ctx, namespace),
	}
}

// HostNameShort returns the short physical name of the name / namespace resource
func (s *multiNamespace) HostNameShort(ctx *synccontext.SyncContext, name, namespace string) types.NamespacedName {
	return s.HostName(ctx, name, namespace)
}

func (s *multiNamespace) HostNameCluster(name string) string {
	if name == "" {
		return ""
	}
	return SafeConcatName("vcluster", name, "x", s.currentNamespace, "x", VClusterName)
}

func (s *multiNamespace) IsManaged(ctx *synccontext.SyncContext, pObj client.Object) bool {
	// check if cluster scoped object
	if pObj.GetNamespace() == "" {
		return pObj.GetLabels()[MarkerLabel] == s.MarkerLabelCluster()
	}

	// vcluster has not synced the object IF:
	// If obj is not in the synced namespace OR
	// If object-name annotation is not set OR
	// If object-name annotation is different from actual name
	if !s.IsTargetedNamespace(ctx, pObj.GetNamespace()) || pObj.GetAnnotations()[NameAnnotation] == "" {
		return false
	} else if pObj.GetAnnotations()[KindAnnotation] != "" {
		gvk, err := apiutil.GVKForObject(pObj, scheme.Scheme)
		if err == nil && gvk.String() != pObj.GetAnnotations()[KindAnnotation] {
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

func (s *multiNamespace) LabelsToTranslate() map[string]bool {
	return map[string]bool{
		// namespace, marker & controlled-by
		NamespaceLabel:  true,
		MarkerLabel:     true,
		ControllerLabel: true,
	}
}

func (s *multiNamespace) IsTargetedNamespace(_ *synccontext.SyncContext, pNamespace string) bool {
	return strings.HasPrefix(pNamespace, s.getNamespacePrefix()) && strings.HasSuffix(pNamespace, getNamespaceSuffix(s.currentNamespace, VClusterName))
}

func (s *multiNamespace) getNamespacePrefix() string {
	return "vcluster"
}

func (s *multiNamespace) HostNamespace(_ *synccontext.SyncContext, vNamespace string) string {
	if vNamespace == "" {
		return ""
	}

	return hostNamespace(s.currentNamespace, vNamespace, s.getNamespacePrefix(), VClusterName)
}

func hostNamespace(currentNamespace, vNamespace, prefix, suffix string) string {
	sha := sha256.Sum256([]byte(vNamespace))
	return fmt.Sprintf("%s-%s-%s", prefix, hex.EncodeToString(sha[0:])[0:8], getNamespaceSuffix(currentNamespace, suffix))
}

func getNamespaceSuffix(currentNamespace, suffix string) string {
	sha := sha256.Sum256([]byte(currentNamespace + "x" + suffix))
	return hex.EncodeToString(sha[0:])[0:8]
}

func (s *multiNamespace) MarkerLabelCluster() string {
	return SafeConcatName(s.currentNamespace, "x", VClusterName)
}
