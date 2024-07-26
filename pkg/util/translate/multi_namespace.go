package translate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

var _ Translator = &multiNamespace{}

func NewMultiNamespaceTranslator(currentNamespace string, nameFormat config.ExperimentalMultiNamespaceNameFormat) Translator {
	return &multiNamespace{
		currentNamespace: currentNamespace,
		nameFormat:       nameFormat,
	}
}

type multiNamespace struct {
	currentNamespace string
	nameFormat       config.ExperimentalMultiNamespaceNameFormat
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
	return strings.HasPrefix(ns, s.getNamespacePrefix()) && strings.HasSuffix(ns, s.getNamespaceSuffix())
}

func (s *multiNamespace) getNamespacePrefix() string {
	return s.nameFormat.Prefix
}

func (s *multiNamespace) HostNamespace(vNamespace string) string {
	base := vNamespace
	if !s.nameFormat.RawBase {
		sha := sha256.Sum256([]byte(base))
		base = hex.EncodeToString(sha[0:])[0:8]
	}
	if s.nameFormat.AvoidRedundantFormatting && s.IsTargetedNamespace(base) {
		return base
	}
	prefix := s.getNamespacePrefix()
	suffix := s.getNamespaceSuffix()
	return fmt.Sprintf("%s-%s-%s", prefix, base, suffix)
}

func (s *multiNamespace) getNamespaceSuffix() string {
	suffix := VClusterName
	if !s.nameFormat.RawSuffix {
		sha := sha256.Sum256([]byte(s.currentNamespace + "x" + suffix))
		suffix = hex.EncodeToString(sha[0:])[0:8]
	}
	return suffix
}

func (s *multiNamespace) MarkerLabelCluster() string {
	return SafeConcatName(s.currentNamespace, "x", VClusterName)
}

func (s *multiNamespace) HostLabelCluster(ctx *synccontext.SyncContext, key string) string {
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
