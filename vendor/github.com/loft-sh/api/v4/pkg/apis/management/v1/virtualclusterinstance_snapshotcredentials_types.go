package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualClusterSnapshotCredentials holds a response used by a tenant cluster to pull its own
// instance's snapshot storage credentials on demand, so credentials never have to be pushed into
// and persisted in the tenant cluster. Credentials are per instance (per auto-snapshot storage
// configuration), not per snapshot, so the request carries no snapshot identifier.
// +subresource-request
type VirtualClusterSnapshotCredentials struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualClusterSnapshotCredentialsSpec   `json:"spec,omitempty"`
	Status VirtualClusterSnapshotCredentialsStatus `json:"status,omitempty"`
}

type VirtualClusterSnapshotCredentialsSpec struct {
}

type VirtualClusterSnapshotCredentialsStatus struct {
	// Options is the JSON-encoded snapshot storage options
	// (github.com/loft-sh/api/v4/pkg/snapshot.Options), resolved server-side from the instance's
	// auto-snapshot configuration. It carries the storage credentials (the storage location comes
	// from the snapshot request itself). The tenant cluster is expected to unmarshal it and use the
	// credentials in memory only, without persisting them.
	// +optional
	Options string `json:"options,omitempty"`
}
