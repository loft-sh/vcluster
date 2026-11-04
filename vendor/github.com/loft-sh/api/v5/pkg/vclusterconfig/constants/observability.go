package constants

// Default namespace and name of the delivered metrics-writer Secret.
const (
	DefaultObservabilityNamespace  string = "observability"
	DefaultMetricsWriterSecretName string = "metrics-writer"
)

// Keys of the delivered metrics-writer Secret. Whatever collector reads the Secret is configured
// against these names, so renaming one breaks every deployed collector.
//
// Not the JSON tags of managementv1.VirtualClusterObservabilityConnectorStatus, which is the
// platform side of the same contract: ca.crt is the Kubernetes convention for a mounted trust
// anchor and does not follow that type's caCert.
const (
	MetricsWriterEndpointKey string = "endpoint"
	MetricsWriterTokenKey    string = "token"
	MetricsWriterCACertKey   string = "ca.crt"
)

// Ownership marker on a delivered metrics-writer Secret. Nothing else sets it, so a Secret at the
// delivery target without it belongs to the tenant and must not be overwritten or deleted.
const (
	MetricsWriterManagedByLabel string = "observability.platform.vcluster.com/managed-by"
	MetricsWriterManagedByValue string = "vcluster"
)
