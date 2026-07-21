package framework

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// IsDefaultStorageClassAnnotation represents a StorageClass annotation that
// marks a class as the default StorageClass
const IsDefaultStorageClassAnnotation = "storageclass.kubernetes.io/is-default-class"

// BetaIsDefaultStorageClassAnnotation is the beta version of IsDefaultStorageClassAnnotation.
// TODO: remove Beta when no longer used
const BetaIsDefaultStorageClassAnnotation = "storageclass.beta.kubernetes.io/is-default-class"

// IsDefaultAnnotation returns a boolean if
// the annotation is set
// TODO: remove Beta when no longer needed
func IsDefaultAnnotation(obj metav1.ObjectMeta) bool {
	if obj.Annotations[IsDefaultStorageClassAnnotation] == "true" {
		return true
	}
	if obj.Annotations[BetaIsDefaultStorageClassAnnotation] == "true" {
		return true
	}

	return false
}
