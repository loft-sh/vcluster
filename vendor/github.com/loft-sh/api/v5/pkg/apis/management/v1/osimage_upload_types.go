package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +subresource-request
type OSImageUpload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              OSImageUploadSpec   `json:"spec"`
	Status            OSImageUploadStatus `json:"status,omitempty"`
}

type OSImageUploadSpec struct {
	// SizeBytes is the total size of the image the caller is about to upload. It has to match
	// the size the upload session was opened for.
	// +kubebuilder:validation:Minimum=1
	SizeBytes int64 `json:"sizeBytes"`

	// FromPart is the 1-based part number this batch starts at; zero starts at the first part.
	// Pointing it back at a part already handed out signs that target again, which is how an
	// upload outliving its targets refreshes them.
	// +optional
	FromPart int32 `json:"fromPart,omitempty"`
}

type OSImageUploadStatus struct {
	// PartSizeBytes is the length of every part but the last.
	PartSizeBytes int64 `json:"partSizeBytes,omitempty"`

	// TotalParts is how many parts SizeBytes covers, across every batch.
	TotalParts int32 `json:"totalParts,omitempty"`

	// NextPart is the FromPart of the next batch, unset once this batch reached TotalParts.
	// +optional
	NextPart *int32 `json:"nextPart,omitempty"`

	// Targets is one batch of upload targets.
	Targets []OSImageUploadTarget `json:"targets,omitempty"`

	// ExpiresAt is when every target in this batch stops being accepted.
	ExpiresAt metav1.Time `json:"expiresAt,omitempty"`
}

type OSImageUploadTarget struct {
	// PartNumber is 1-based, so the part covers the bytes at offset
	// (PartNumber-1)*PartSizeBytes.
	PartNumber int32 `json:"partNumber"`

	// URL accepts a plain PUT of that part's bytes. Only Host is signed, so no
	// additional request headers are needed.
	URL string `json:"url"`
}
