package translate

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	NamespaceLabel = "vcluster.loft.sh/namespace"
	MarkerLabel    = "vcluster.loft.sh/managed-by"
	Suffix         = "suffix"
)

var Owner client.Object

func DefaultImageRegistry() string {
	return os.Getenv("DEFAULT_IMAGE_REGISTRY")
}

func SafeConcatGenerateName(name ...string) string {
	fullPath := strings.Join(name, "-")
	if len(fullPath) > 53 {
		digest := sha256.Sum256([]byte(fullPath))
		return strings.Replace(fullPath[0:42]+"-"+hex.EncodeToString(digest[0:])[0:10], ".-", "-", -1)
	}
	return fullPath
}

func SafeConcatName(name ...string) string {
	fullPath := strings.Join(name, "-")
	if len(fullPath) > 63 {
		digest := sha256.Sum256([]byte(fullPath))
		return strings.Replace(fullPath[0:52]+"-"+hex.EncodeToString(digest[0:])[0:10], ".-", "-", -1)
	}
	return fullPath
}

func IsManaged(obj runtime.Object) bool {
	metaAccessor, err := meta.Accessor(obj)
	if err != nil {
		return false
	} else if metaAccessor.GetLabels() == nil {
		return false
	}

	return metaAccessor.GetLabels()[MarkerLabel] == Suffix
}

func IsManagedCluster(physicalNamespace string, obj runtime.Object) bool {
	metaAccessor, err := meta.Accessor(obj)
	if err != nil {
		return false
	} else if metaAccessor.GetLabels() == nil {
		return false
	}

	return metaAccessor.GetLabels()[MarkerLabel] == SafeConcatName(physicalNamespace, "x", Suffix)
}

// PhysicalName returns the physical name of the name / namespace resource
func PhysicalName(name, namespace string) string {
	if name == "" {
		return ""
	}
	return SafeConcatName(name, "x", namespace, "x", Suffix)
}

// PhysicalNameClusterScoped returns the physical name of a cluster scoped object in the host cluster
func PhysicalNameClusterScoped(name, physicalNamespace string) string {
	if name == "" {
		return ""
	}
	return SafeConcatName("vcluster", name, "x", physicalNamespace, "x", Suffix)
}
