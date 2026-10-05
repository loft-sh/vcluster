package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualClusterExternalDatabase holds kube config request and response data for tenant clusters
// +subresource-request
type VirtualClusterExternalDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualClusterExternalDatabaseSpec   `json:"spec,omitempty"`
	Status VirtualClusterExternalDatabaseStatus `json:"status,omitempty"`
}

type VirtualClusterExternalDatabaseSpec struct {
	// Connector specifies the secret that should be used to connect to an external database server. The connection is
	// used to manage a user and database for the vCluster. A data source endpoint constructed from the created user and
	// database is returned on status. The secret specified by connector should contain the following fields:
	// endpoint - the endpoint where the database server can be accessed
	// user - the database username
	// password - the password for the database username
	// port - the port to be used in conjunction with the endpoint to connect to the databse server. This is commonly
	// 3306
	// The following field is optional:
	// caCert - PEM-encoded CA bundle used by the tenant cluster to verify the database server's TLS
	// certificate. When set, the value is returned on status so the tenant cluster can write it to
	// disk and enable sslmode=verify-full.
	// +optional
	Connector string `json:"connector,omitempty"`
}

type VirtualClusterExternalDatabaseStatus struct {
	// DataSource holds a datasource endpoint constructed from the vCluster's designated user and database. The user and
	// database are created from the given connector.
	DataSource string `json:"dataSource,omitempty"`

	// IdentityProvider is the kine identity provider to use when generating temporary authentication tokens for
	// enhanced security.
	IdentityProvider string `json:"identityProvider,omitempty"`

	// CaCert is the PEM-encoded CA bundle the tenant cluster should use to verify the database
	// server's TLS certificate, sourced from the connector secret's caCert field. When non-empty,
	// the tenant cluster should write this to disk and configure Kine to verify the server against
	// it (sslmode=verify-full).
	// +optional
	CaCert string `json:"caCert,omitempty"`

	// SslMode is an explicit Postgres sslmode value sourced from the connector secret's sslMode
	// field. When non-empty (e.g. "disable", "require", "verify-full"), the tenant cluster should
	// pass this to Kine to override the default policy. Empty means the tenant cluster should
	// derive the mode from CaCert (verify-full when CaCert is set, require otherwise).
	// +optional
	SslMode string `json:"sslMode,omitempty"`
}
