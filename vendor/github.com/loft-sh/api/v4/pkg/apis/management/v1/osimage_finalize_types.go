package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +subresource-request
type OSImageFinalize struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              OSImageFinalizeSpec   `json:"spec"`
	Status            OSImageFinalizeStatus `json:"status,omitempty"`
}

type OSImageFinalizeSpec struct {
	// Checksum is the sha256 of the whole file, lowercase hex. It is recorded for consumers
	// rather than verified: S3 has no whole-file hash for a multipart object.
	// +kubebuilder:validation:Pattern=`^[a-f0-9]{64}$`
	Checksum string `json:"checksum"`
}

type OSImageFinalizeStatus struct {
	// CompletedAt is when the object was assembled. A repeated call reports the first one,
	// since completing again would change an object consumers may already have pulled.
	CompletedAt metav1.Time `json:"completedAt,omitempty"`
}
