package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualClusterObservabilityConnector holds the credential and write-gateway connection details a tenant
// cluster's collector needs to push metrics to the Platform.
//
// Creating it refuses in one of five ways, which differ only in what the caller should do next:
//
//   - 400: the connector or the instance's own config is missing, unusable or has to be repaired
//   - 403: the platform refuses the caller's entitlement or identity
//   - 404: the instance is not there, or is on its way out
//   - 405: the deployment builds no connector at all
//   - 503: the write gateway is still coming up
//
// The code carries the whole of that answer, so a caller decides whether to retry from it alone.
// The status message is for people: it names the specific cause, and the platform reports the same
// finding as a warning condition on the virtual cluster instance, which is where the admin who owns
// the connector is looking.
//
// +subresource-request
type VirtualClusterObservabilityConnector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualClusterObservabilityConnectorSpec   `json:"spec,omitempty"`
	Status VirtualClusterObservabilityConnectorStatus `json:"status,omitempty"`
}

type VirtualClusterObservabilityConnectorSpec struct{}

type VirtualClusterObservabilityConnectorStatus struct {
	// Endpoint is the OTLP/HTTP URL of the write gateway backing the connector named by the
	// instance's integrations.observability.connector.
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// Token is the metrics-writer access key scoped to this instance. Stable across calls.
	// +optional
	Token string `json:"token,omitempty"`

	// CaCert is the PEM trust anchor for Endpoint. Empty when the gateway serves a cert signed by a
	// publicly trusted CA, in which case the system trust store applies.
	// +optional
	CaCert string `json:"caCert,omitempty"`
}
