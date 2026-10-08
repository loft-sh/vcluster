package v1

import (
	"slices"

	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OSImage holds the information of machine networks
// +k8s:openapi-gen=true
type OSImage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OSImageSpec   `json:"spec,omitempty"`
	Status OSImageStatus `json:"status,omitempty"`
}

func (a *OSImage) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *OSImage) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *OSImage) GetAccess() []Access {
	return a.Spec.Access
}

func (a *OSImage) SetAccess(access []Access) {
	a.Spec.Access = access
}

func (a *OSImage) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

func (a *OSImage) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}

type OSImageSpec struct {
	// DisplayName is the name that should be displayed in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes an OS image
	// +optional
	Description string `json:"description,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// Access holds the access rights for users and teams
	// +optional
	Access []Access `json:"access,omitempty"`

	// Properties is the configuration for the OS image
	// +optional
	Properties map[string]string `json:"properties,omitempty"`

	// ConnectorRef names the image store connector holding this image's blob.
	// Empty means a properties-only image. Immutable once set.
	// +optional
	ConnectorRef string `json:"connectorRef,omitempty"`

	// Format is the on-disk format of the image blob and part of the object key.
	// Immutable once set.
	// +optional
	Format OSImageFormat `json:"format,omitempty"`
}

// OSImageFormat is the on-disk format of an OS image blob.
// +kubebuilder:validation:Enum=qcow2
type OSImageFormat string

const OSImageFormatQCOW2 OSImageFormat = "qcow2"

const (
	// OSImageUploadSessionIDAnnotation holds the multipart upload ID of the open session.
	OSImageUploadSessionIDAnnotation = "machines.vcluster.com/os-image-upload-session-id"

	// OSImageUploadDeclaredSizeAnnotation holds the byte size the session was opened for.
	OSImageUploadDeclaredSizeAnnotation = "machines.vcluster.com/os-image-upload-declared-size"

	// OSImageUploadLastInitiatedAtAnnotation holds the RFC3339 time of the last upload call.
	OSImageUploadLastInitiatedAtAnnotation = "machines.vcluster.com/os-image-upload-last-initiated-at"

	// OSImageUploadCompletedAtAnnotation holds the RFC3339 time the object was completed.
	OSImageUploadCompletedAtAnnotation = "machines.vcluster.com/os-image-upload-completed-at"

	// OSImageUploadChecksumAnnotation holds the sha256 the client declared at finalize.
	OSImageUploadChecksumAnnotation = "machines.vcluster.com/os-image-upload-checksum"
)

// OSImageUploadSessionIdentityAnnotations is what names the open session, as opposed to what
// it is uploading. A settled image drops these and keeps the rest: the size is what a later
// pass re-checks the object against.
var OSImageUploadSessionIdentityAnnotations = []string{
	OSImageUploadSessionIDAnnotation,
	OSImageUploadLastInitiatedAtAnnotation,
}

// OSImageUploadInitiateAnnotations is what opening a session writes.
var OSImageUploadInitiateAnnotations = slices.Concat(
	OSImageUploadSessionIdentityAnnotations,
	[]string{OSImageUploadDeclaredSizeAnnotation},
)

// OSImageUploadFinalizeAnnotations is what completing the object writes.
var OSImageUploadFinalizeAnnotations = []string{
	OSImageUploadCompletedAtAnnotation,
	OSImageUploadChecksumAnnotation,
}

// OSImageUploadSessionAnnotations is every annotation one upload session writes, so cleanup
// clears this set rather than a list of its own.
var OSImageUploadSessionAnnotations = slices.Concat(
	OSImageUploadInitiateAnnotations,
	OSImageUploadFinalizeAnnotations,
)

// OSImagePhase is where an image is in its upload lifecycle. An image with no ConnectorRef
// holds no bytes and therefore carries no phase.
type OSImagePhase string

const (
	// OSImagePhasePending means the image exists but no upload session has been opened.
	OSImagePhasePending OSImagePhase = "Pending"

	// OSImagePhaseUploading means a session is open and parts are expected.
	OSImagePhaseUploading OSImagePhase = "Uploading"

	// OSImagePhaseReady means the object is assembled and its size matches what the client
	// declared. Terminal: the bytes of a ready image are not replaced.
	OSImagePhaseReady OSImagePhase = "Ready"

	// OSImagePhaseFailed means the upload did not finish. Terminal for reads only: a new
	// upload call starts the image over without losing its name, owner or access rules.
	OSImagePhaseFailed OSImagePhase = "Failed"
)

// OSImageConditionTypeUploaded is True once the object is assembled and verified, and False
// with the reason while it is not. It carries the why behind Failed.
const OSImageConditionTypeUploaded agentstoragev1.ConditionType = "Uploaded"

type OSImageStatus struct {
	// Phase is where the image is in its upload lifecycle. Empty for an image that holds no
	// bytes. The controller owns every transition; the upload endpoints only annotate.
	// +optional
	Phase OSImagePhase `json:"phase,omitempty"`

	// Conditions holds the upload conditions of the image.
	// +optional
	Conditions agentstoragev1.Conditions `json:"conditions,omitempty"`

	// ExpiresAt is the deadline of the current non-terminal phase, which the phase names.
	// Cleared on Ready and Failed, and set again when a retry opens a new session.
	// +optional
	ExpiresAt *metav1.Time `json:"expiresAt,omitempty"`

	// Store describes the object holding the image bytes. Set once the upload is verified.
	// +optional
	Store *OSImageStoreStatus `json:"store,omitempty"`
}

// OSImageStoreStatus is the verified object behind a Ready image.
type OSImageStoreStatus struct {
	// Location is the object key in the bucket.
	// +optional
	Location string `json:"location,omitempty"`

	// SizeBytes is the size the object store reports for the object, not the one the client
	// declared.
	// +optional
	SizeBytes int64 `json:"sizeBytes,omitempty"`

	// Checksum is the sha256 the client declared at finalize. It is recorded for consumers
	// such as Ironic rather than verified, since S3 has no whole-file hash for a multipart
	// object.
	// +optional
	Checksum string `json:"checksum,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OSImageList contains a list of OSImages
type OSImageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OSImage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OSImage{}, &OSImageList{})
}
