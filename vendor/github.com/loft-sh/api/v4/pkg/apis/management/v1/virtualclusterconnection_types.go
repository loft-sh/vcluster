package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +genclient:onlyVerbs=create,deletecollection
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualClusterConnection provides connection information for a target virtual cluster
// This allows vClusters to discover and connect to each other via Tailscale
// +k8s:openapi-gen=true
// +resource:path=virtualclusterconnections,rest=VirtualClusterConnectionREST
type VirtualClusterConnection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualClusterConnectionSpec   `json:"spec,omitempty"`
	Status VirtualClusterConnectionStatus `json:"status,omitempty"`
}

// VirtualClusterConnectionSpec defines the target virtual cluster to connect to
type VirtualClusterConnectionSpec struct {
	// TargetVirtualCluster specifies the vCluster to connect to
	// +optional
	TargetVirtualCluster TargetVirtualClusterConfig `json:"targetVirtualCluster,omitempty"`

	// Client is the client vCluster requesting the connection
	// +optional
	Client ConnectionClientConfig `json:"client,omitempty"`

	// Resources is the list of resources to allow the client to access on the target vCluster
	// +optional
	Resources []string `json:"resources,omitempty"`
}

// TargetVirtualClusterConfig specifies the target virtual cluster to connect to
type TargetVirtualClusterConfig struct {
	// Name is the name of the target vCluster
	Name string `json:"name"`

	// Project is the project of the target vCluster
	// +optional
	Project string `json:"project,omitempty"`

	// ServiceAccountRef is the service account to use for the proxy in target vCluster
	// +optional
	ServiceAccountRef NamespacedNameArgs `json:"serviceAccountRef,omitempty"`
}

// ConnectionClientConfig specifies the client vCluster to connect to
type ConnectionClientConfig struct {
	// Name is the name of the client vCluster
	Name string `json:"name"`

	// Project is the project of the client vCluster
	// +optional
	Project string `json:"project,omitempty"`
}

// VirtualClusterConnectionStatus contains the connection information for the target vCluster
type VirtualClusterConnectionStatus struct {
	// Hostname is the hostname of the target vCluster
	// +optional
	Hostname string `json:"hostname,omitempty"`

	// Online indicates whether the target vCluster is online and reachable
	// +optional
	Online bool `json:"online,omitempty"`

	// Token is a ServiceAccount token that should be used to authenticate to the target vCluster
	// +optional
	Token string `json:"token,omitempty"`

	// Message contains any error or informational message
	// +optional
	Message string `json:"message,omitempty"`
}
