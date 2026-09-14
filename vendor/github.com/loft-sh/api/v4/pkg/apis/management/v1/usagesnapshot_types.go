package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UsageDownload returns a zip of CSV files containing table data from the usage
// postgres database
// +k8s:openapi-gen=true
// +resource:path=usagedownload,rest=UsageDownloadREST
type UsageDownload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UsageDownloadSpec   `json:"spec,omitempty"`
	Status UsageDownloadStatus `json:"status,omitempty"`
}

type UsageDownloadSpec struct {
}

type UsageDownloadStatus struct {
}
