package translate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
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
func (s *multiNamespace) HostName(name, _ string) string {
	return name
}

// HostNameShort returns the short physical name of the name / namespace resource
func (s *multiNamespace) HostNameShort(name, _ string) string {
	return name
}

func (s *multiNamespace) HostNameCluster(name string) string {
	if name == "" {
		return ""
	}
	return SafeConcatName("vcluster", name, "x", s.currentNamespace, "x", VClusterName)
}

func (s *multiNamespace) IsManaged(_ *synccontext.SyncContext, pObj client.Object) bool {
	// check if cluster scoped object
	if pObj.GetNamespace() == "" {
		return pObj.GetLabels()[MarkerLabel] == s.MarkerLabelCluster()
	}

	// vcluster has not synced the object IF:
	// If obj is not in the synced namespace OR
	// If object-name annotation is not set OR
	// If object-name annotation is different from actual name
	if !s.IsTargetedNamespace(pObj.GetNamespace()) || pObj.GetAnnotations()[NameAnnotation] == "" {
		return false
	} else if pObj.GetAnnotations()[KindAnnotation] != "" {
		gvk, err := apiutil.GVKForObject(pObj, scheme.Scheme)
		if err == nil && gvk.String() != pObj.GetAnnotations()[KindAnnotation] {
			return false
		}
	}

	return true
}

func (s *multiNamespace) IsTargetedNamespace(ns string) bool {
	return strings.HasPrefix(ns, s.getNamespacePrefix()) && strings.HasSuffix(ns, getNamespaceSuffix(s.currentNamespace, VClusterName))
}

func (s *multiNamespace) getNamespacePrefix() string {
	return "vcluster"
}

func (s *multiNamespace) HostNamespace(vNamespace string) string {
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

func (s *multiNamespace) HostLabelCluster(ctx *synccontext.SyncContext, key string) (retLabel string) {
	defer func() {
		recordLabelCluster(ctx, key, retLabel)
	}()

	// check if the label is within the store
	if ctx != nil && ctx.Mappings != nil && ctx.Mappings.Store() != nil {
		vLabel, ok := ctx.Mappings.Store().HostToVirtualLabelCluster(ctx, key)
		if ok {
			return vLabel
		}
	}

	if keyMatchesSyncedLabels(ctx, key) {
		return key
	}

	return hostLabelCluster(key, s.currentNamespace)
}

func (s *multiNamespace) HostLabel(_ *synccontext.SyncContext, key string) string {
	return key
}

func hostLabelCluster(key, vClusterNamespace string) string {
	digest := sha256.Sum256([]byte(key))
	return SafeConcatName(LabelPrefix, vClusterNamespace, "x", VClusterName, "x", hex.EncodeToString(digest[0:])[0:10])
}
