package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +subresource-request
type ProjectChartInfo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectChartInfoSpec   `json:"spec,omitempty"`
	Status ProjectChartInfoStatus `json:"status,omitempty"`
}

type ProjectChartInfoSpec struct {
	// Chart holds information about the chart to retrieve info for
	// +optional
	Chart storagev1.Chart `json:"chart,omitempty"`
}

type ProjectChartInfoStatus struct {
	// Metadata provides information about a chart
	// +optional
	Metadata *storagev1.Metadata `json:"metadata,omitempty"`

	// Readme is the readme of the chart
	// +optional
	Readme string `json:"readme,omitempty"`

	// Values are the default values of the chart
	// +optional
	Values string `json:"values,omitempty"`
}
