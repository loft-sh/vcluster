package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +genclient:noStatus
// +genclient:onlyVerbs=list
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DatabaseConnector represents a connector that can be used to provision and manage a backingstore
// for a vCluster
//
// Deprecated: the Connector API (connectors.management.loft.sh) supersedes this list-only kind.
// It serves the same shared-database connector Secrets as Connector objects with
// spec.sharedDatabase, carrying the display name and the connection settings this kind never had.
// Read connectors instead.
// +k8s:openapi-gen=true
// +resource:path=databaseconnectors,rest=DatabaseConnectorREST
type DatabaseConnector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatabaseConnectorSpec   `json:"spec,omitempty"`
	Status DatabaseConnectorStatus `json:"status,omitempty"`
}

// DatabaseConnectorSpec holds the specification
type DatabaseConnectorSpec struct {
	// The client id of the client
	Type        string `json:"type,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}

// DatabaseConnectorStatus holds the status
type DatabaseConnectorStatus struct {
}
