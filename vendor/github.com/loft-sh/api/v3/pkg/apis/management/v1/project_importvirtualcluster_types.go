package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProjectImportVirtualCluster holds project vcluster import information
// +subresource-request
type ProjectImportVirtualCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// SourceVirtualCluster is the virtual cluster to import into this project
	SourceVirtualCluster ProjectImportVirtualClusterSource `json:"sourceVirtualCluster"`

	// UpgradeToPro indicates whether we should upgrade to Pro on import
	UpgradeToPro bool `json:"upgradeToPro,omitempty"`
}

type ProjectImportVirtualClusterSource struct {
	// Name of the virtual cluster to import
	Name string `json:"name,omitempty"`
	// Namespace of the virtual cluster to import
	Namespace string `json:"namespace,omitempty"`
	// Cluster name of the cluster the virtual cluster is running on
	Cluster string `json:"cluster,omitempty"`
	// ImportName is an optional name to use as the virtualclusterinstance name, if not provided
	// the vcluster name will be used
	// +optional
	ImportName string `json:"importName,omitempty"`
}
