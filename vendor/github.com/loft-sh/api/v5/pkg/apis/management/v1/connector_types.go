package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v5/pkg/apis/loft/storage/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +genclient:noStatus
// +genclient:skipVerbs=watch,deleteCollection
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Connector represents an integration connector of the platform (shared database,
// Argo CD, observability). It is a cluster-scoped virtual resource: it has no CRD of
// its own and is served straight from the connector Secret (labeled with
// loft.sh/connector-type) in the platform namespace, through the same translating
// client every other Secret-backed management kind uses. Watch is not served in v1: a
// watch request fails instead of hanging, so list and poll instead.
//
// Rejection happens at write time only: create and update through this API validate
// the request, while a read never rejects, so pre-existing Secrets with missing
// optional fields still project. A read does re-run that validation, but only to
// report the outcome in status, so a malformed Secret is described rather than hidden.
// Writers that bypass this API and write the backing Secrets directly (the UI and
// platform controllers today) are trusted to keep the Secrets well-formed; this API
// does not defend against out-of-band malformed Secrets.
//
// Status is read-only on the wire. Whatever a request carries is ignored: a create stores
// no status and an update keeps the stored object's. Status is assembled on every read,
// part of it computed from the stored payload and part of it unpacked from an annotation
// controllers publish on the backing Secret. The redacted view serves the conditions without
// their messages, which can quote payload values and endpoints.
//
// Metadata is the backing Secret's. Labels and annotations are served and written as they are
// stored, as on every other management kind, with the exceptions each connector type declares.
// Projected keys become fields and are not served as metadata: the loft.sh/connector-type
// label is derived from the payload section, the loft.sh/display-name annotation is spec.displayName. Server-owned keys
// are the platform's, never served, and a write that names one is refused: the status
// annotation controllers publish, unpacked into status. Withheld keys are never served: kubectl's
// last-applied-configuration annotation, because on a Secret written by hand it carries the
// whole manifest, credentials included. A write that sends a projected key disagreeing with the
// field it mirrors is refused. Labels the
// tenancy layer owns (tenant.vcluster.com/*) are that layer's, not this type's: an operator
// sees them and assigns a connector to a tenant by writing one, while a tenant never sees them
// and a tenant's write naming one is refused. Finalizers and ownerReferences round-trip.
// managedFields are served with this API's apiVersion. A label selector on
// loft.sh/connector-type still narrows a list to that type although the served object does not
// carry the label. Because the last-applied annotation is withheld, a client-side kubectl apply
// never finds it and reports a permanent diff: use server-side apply, imperative writes or
// full-replace updates. This describes the full view. The redacted view keeps the identity
// fields and labels and drops annotations, managedFields, ownerReferences and finalizers.
//
// Credentials and connection details live in the type-specific payload sections of spec and
// are served only to a caller who could update this connector, such as a holder of
// loft-connectors-admin. Every other caller, holders of loft-connectors-viewer and tenants
// included, reads a redacted view built from an allowlist: the name, the display name, the
// payload section as a type marker, and the fields the type opts in (the shared-database
// dialect, the image-store protocol). A field the type adds later is not served in it. A
// platform component that needs the connection details reads the backing Secret or
// authenticates as an identity holding loft-connectors-admin; with a lesser identity it reads
// the redacted view, and the empty fields there are not an error.
//
// Tenancy in this release: a platform admin creates connectors and assigns one to a
// tenant through the tenancy layer's exclusive assignment. A tenant reads the connectors
// assigned to it in the redacted view, and cannot create, update or delete a connector; every other
// connector is not found. Tenant-authored connectors come later, without a change to this
// type.
//
// An update through this API replaces the backing Secret from the request; nothing is read
// from the stored Secret to merge with. Passthrough labels and annotations, finalizers and
// ownerReferences round-trip because they are served and sent back. A credential the
// request leaves empty keeps its stored value, and the conditions controllers published
// are carried over from the stored object. What this API never serves does not survive an
// update: Data keys the connector type does not model, and the withheld annotations. The
// keys this API writes are the ones the platform's consumers of a connector read.
//
// The verbs behave as the generic path serves every Secret-backed kind. A delete is by
// name, with no precondition on the object's UID, and is reported as completed even when
// a finalizer on the backing Secret holds the Secret back. NotFound, AlreadyExists and
// Conflict speak of connectors; an Invalid or Forbidden raised by the store itself is
// passed through as the store phrased it.
// +k8s:openapi-gen=true
// +resource:path=connectors,rest=ConnectorREST
type Connector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConnectorSpec   `json:"spec,omitempty"`
	Status ConnectorStatus `json:"status,omitempty"`
}

// ConnectorSpec holds the specification. Exactly one payload section must be set.
// The section determines the connector type, stamped to the loft.sh/connector-type
// label on create. The connector type is immutable after create.
type ConnectorSpec struct {
	// DisplayName is the human-readable name shown in the UI. It is projected from
	// and stamped to the loft.sh/display-name annotation on the backing Secret.
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description is the free-text description shown in the UI. It is projected from
	// and stamped to the loft.sh/description annotation on the backing Secret.
	// +optional
	Description string `json:"description,omitempty"`

	// SharedDatabase is the shared-database connector payload. Exactly one payload
	// section must be set. Its fields
	// mirror the Data keys of the backing Secret one to one, so a connector written
	// through this API is byte-compatible with one written by the UI directly.
	// +optional
	SharedDatabase *ConnectorSharedDatabaseSpec `json:"sharedDatabase,omitempty"`

	// ArgoCD is the argocd connector payload. Exactly one payload section
	// must be set. Its fields mirror the Data keys of the
	// backing Secret one to one, so a connector written through this API is
	// byte-compatible with one written by the UI directly.
	// +optional
	ArgoCD *ConnectorArgoCDSpec `json:"argoCd,omitempty"`

	// NetBox is the netbox connector payload. Exactly one payload section
	// must be set. Its fields mirror the Data keys of the
	// backing Secret one to one, so a connector written through this API is
	// byte-compatible with one written by the UI directly.
	// +optional
	NetBox *ConnectorNetBoxSpec `json:"netBox,omitempty"`

	// ImageStore is the os-image-store connector payload. Exactly one payload section
	// must be set. The fields of its protocol
	// block mirror the Data keys of the backing Secret one to one, so a connector
	// written through this API is exactly what the OSImage store loader reads.
	// +optional
	ImageStore *ConnectorImageStoreSpec `json:"imageStore,omitempty"`
	// Observability is the observability connector payload. It is required when type
	// is observability and must be unset for every other type. Its fields mirror the
	// Data keys of the backing Secret one to one (fleetDefault excepted, which is
	// backed by a server-owned label), so a connector written through this API is
	// byte-compatible with one written by the UI directly.
	// +optional
	Observability *ConnectorObservabilitySpec `json:"observability,omitempty"`
}

// SharedDatabaseDialect is the database server dialect of a shared-database
// connector. It is stored in the "type" Data key of the backing Secret; the field is
// named dialect on the wire to distinguish it from the connector type label.
type SharedDatabaseDialect string

const (
	// SharedDatabaseDialectMySQL is the MySQL dialect. It is the platform's
	// historical default: consumers treat a connector Secret without an explicit
	// dialect as MySQL.
	SharedDatabaseDialectMySQL SharedDatabaseDialect = "mysql"

	// SharedDatabaseDialectPostgres is the PostgreSQL dialect.
	SharedDatabaseDialectPostgres SharedDatabaseDialect = "postgres"
)

// ConnectorSharedDatabaseSpec configures a connection to a shared database server
// used to provision databases as tenant cluster backing stores. Every field maps to
// one Data key of the backing Secret (given in each field comment), so direct Secret
// writers and this API stay byte-compatible in both directions.
type ConnectorSharedDatabaseSpec struct {
	// Dialect is the database server dialect, mysql or postgres. Data key "type"
	// (distinct from the connector type label). Required on every write: there
	// is no server-side default and unknown values are rejected. Reads of a
	// pre-existing Secret without the key still project mysql, matching how the read
	// consumers treat such Secrets.
	// +required
	Dialect SharedDatabaseDialect `json:"dialect"`

	// Endpoint is the host of the database server. Data key "endpoint". Required.
	// +required
	Endpoint string `json:"endpoint"`

	// Port is the port of the database server, kept as a string exactly as stored in
	// the Data key "port". An empty port defaults by dialect on CREATE ONLY: 5432
	// for postgres, 3306 for mysql (the same defaults the UI applies). Updates never
	// default it. For postgres an empty port on update removes the key and the driver
	// falls back to 5432; for mysql an empty port is rejected on update, because the
	// mysql driver has no fallback and would dial port 0. A stored mysql connector
	// without a port is therefore rejected on every update, whatever else the update
	// changes, until the port is set; such a connector cannot connect until then
	// anyway, and its Configured condition says so.
	// +optional
	Port string `json:"port,omitempty"`

	// User is the database user the platform connects as. Data key "user". Required.
	// +required
	User string `json:"user"`

	// Password is the password of the database user. Data key "password". Required
	// on create unless identityProvider is set. It is a credential: served only to a
	// caller who could update this connector and absent for every other caller (see
	// the Connector type). On update an empty password keeps the stored one, so a
	// read without the credential written back never clears it; switching to
	// identityProvider removes it. Switching back from identityProvider to password
	// authentication requires a non-empty password in the request: the stored one
	// was removed by the earlier switch, so there is nothing to keep.
	// +optional
	Password string `json:"password,omitempty"`

	// IdentityProvider selects password-less IAM authentication. Data key
	// "identityProvider". The only supported value is "aws"; other values are
	// rejected. When set, password must be empty and the stored password key is
	// removed, matching the UI behavior.
	// +optional
	IdentityProvider string `json:"identityProvider,omitempty"`

	// AWSPermissionBoundaryPolicyARN is the ARN of the IAM policy attached as the
	// permissions boundary of every IAM role the platform provisions for a tenant
	// cluster through this connector. It is only read when identityProvider is set.
	// Unlike the other fields it maps to an annotation on the backing Secret
	// (connector.platform.vcluster.com/aws-permission-boundary-policy-arn), not to a
	// Data key, and this field is the one way to write it: the annotation is
	// projected, so a metadata write carrying a different value is refused. Empty
	// means no boundary, which is a supported configuration, and an empty value on
	// update removes the annotation, so a read-modify-write that drops the field
	// clears a boundary that was set.
	// +optional
	AWSPermissionBoundaryPolicyARN string `json:"awsPermissionBoundaryPolicyArn,omitempty"`

	// CACert is the PEM-encoded CA bundle used to verify the database server's TLS
	// certificate. Data key "caCert".
	// +optional
	CACert string `json:"caCert,omitempty"`

	// SSLMode is an explicit Postgres sslmode value (disable, allow, prefer, require,
	// verify-ca or verify-full) overriding the default policy derived from caCert.
	// Data key "sslMode".
	// +optional
	SSLMode string `json:"sslMode,omitempty"`
}

// ConnectorArgoCDSpec configures a connection to an Argo CD instance used by the Argo CD
// integration. Exactly one of selfHosted or akuity must be set. The flavor is stored in
// the "connectorType" Data key of the backing Secret: "akuity" for akuity, and absent
// (or any other value, as on legacy Secrets) for selfHosted, exactly like the UI.
//
// Every field maps to one Data key of the backing Secret (given in each field comment),
// so direct Secret writers and this API stay byte-compatible in both directions.
//
// Three fields are credentials under the Connector type's view rule: token, password
// and the Akuity apiKeySecret are served to a caller who could update this connector
// and absent for every other caller, and on update an empty value keeps the stored one,
// so a read without the credential written back never clears it.
type ConnectorArgoCDSpec struct {
	// SelfHosted is a self-hosted Argo CD instance reached directly at the server URL.
	// Must be unset when akuity is set.
	// +optional
	SelfHosted *ConnectorArgoCDSelfHostedSpec `json:"selfHosted,omitempty"`

	// Akuity is an Argo CD instance managed by the Akuity Platform: cluster
	// registration goes through Akuity's control plane API. Must be unset when
	// selfHosted is set.
	// +optional
	Akuity *ConnectorArgoCDAkuitySpec `json:"akuity,omitempty"`
}

// ConnectorArgoCDSelfHostedSpec configures a self-hosted Argo CD instance.
type ConnectorArgoCDSelfHostedSpec struct {
	ConnectorArgoCDServer `json:",inline"`
}

// ConnectorArgoCDAkuitySpec configures an Akuity-managed Argo CD instance. The Argo CD
// API server fields apply as for a self-hosted instance, with the Akuity Platform
// fields on top.
type ConnectorArgoCDAkuitySpec struct {
	ConnectorArgoCDServer `json:",inline"`

	// OrgID is the Akuity Platform organization ID. Data key "akuityOrgId". Required.
	// +required
	OrgID string `json:"orgId"`

	// InstanceID is the Akuity-hosted Argo CD instance ID. Data key "akuityInstanceId".
	// Required.
	// +required
	InstanceID string `json:"instanceId"`

	// APIKeyID is the Akuity API key ID. Data key "akuityApiKeyId". Required.
	// +required
	APIKeyID string `json:"apiKeyId"`

	// APIKeySecret is the Akuity API key secret. Data key "akuityApiKeySecret".
	// Required whenever no stored value exists: on create, and on an update that
	// switches from selfHosted to akuity, since the live Secret of a self-hosted
	// connector carries no Akuity keys. A credential (see ConnectorArgoCDSpec): an
	// empty value on an update that stays akuity keeps the stored one.
	// +optional
	APIKeySecret string `json:"apiKeySecret,omitempty"`

	// AgentSize is the resource allocation of the Akuity agent, an Akuity size name of
	// the form CLUSTER_SIZE_<NAME> such as CLUSTER_SIZE_SMALL, CLUSTER_SIZE_MEDIUM or
	// CLUSTER_SIZE_LARGE; the value is passed to Akuity as given, so any size Akuity
	// accepts is valid here. Data key "akuityAgentSize". Empty leaves the sizing to
	// the Akuity side (medium), like the UI.
	// +optional
	AgentSize string `json:"agentSize,omitempty"`

	// RepoServerReplicas is an optional replica count override for the
	// argocd-repo-server of the Akuity agent, a positive integer kept as a string
	// exactly as stored in the Data key "akuityRepoServerReplicas".
	// +optional
	RepoServerReplicas string `json:"repoServerReplicas,omitempty"`

	// RepoServerMemory is an optional memory limit/request override for the
	// argocd-repo-server of the Akuity agent, written as a number with an optional
	// decimal part and an optional SI or binary suffix (k, M, G, T, P, E, Ki, Mi, Gi,
	// Ti, Pi, Ei), for example "512Mi" or "1Gi". This is the shape the UI accepts; it
	// is narrower than a full Kubernetes quantity, which also allows exponents such as
	// 1e9 and the milli suffix. Data key "akuityRepoServerMemory".
	// +optional
	RepoServerMemory string `json:"repoServerMemory,omitempty"`
}

// ConnectorArgoCDServer is how the platform reaches and authenticates against the
// Argo CD API server. Both flavors talk to it, so both carry these fields.
type ConnectorArgoCDServer struct {
	// Server is the URL the Argo CD API server is reachable at; for akuity this is the
	// Akuity-hosted Argo CD instance URL. Data key "server". Required, must be an
	// http or https URL.
	// +required
	Server string `json:"server"`

	// Namespace is the namespace Argo CD runs in on the destination cluster. Data key
	// "namespace". An empty namespace is defaulted to "argocd" on create only (the
	// same default the UI pre-fills and consumers assume for Secrets without the
	// key); on update an empty namespace removes the key, and reads fall back to the
	// same default, so the effective namespace consumers use never changes.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Token is the Argo CD API token, the default authentication method. Data key
	// "token". Required on create unless username and password are set. A credential
	// (see ConnectorArgoCDSpec): an empty token on update keeps the stored one, and
	// setting username/password removes it (switching the authentication method, like
	// the UI does). Keeping is only valid while the stored connector holds a usable
	// credential: a token, or a full username and password pair (half a pair does not
	// count). A pre-existing Secret holding neither cannot be updated through this
	// API, even on unrelated fields, until the request supplies a credential; the
	// escape hatch is editing the backing Secret directly.
	// +optional
	Token string `json:"token,omitempty"`

	// Username is the Argo CD username for basic authentication, used together with
	// password as the alternative to token. Data key "username". Projected in the full view
	// only when no token is stored, because the consumer authenticates with the token whenever one
	// is present and a username beside it describes a method the connector does not use.
	// +optional
	Username string `json:"username,omitempty"`

	// Password is the Argo CD password for basic authentication. Data key "password".
	// Required whenever username is set and no stored password exists: on create, and
	// on an update that switches from token to basic authentication, since setting the
	// token had removed the stored password. A credential (see ConnectorArgoCDSpec): an
	// empty password on an update that keeps basic authentication keeps the stored one,
	// and setting token removes it (switching the authentication method, like the UI
	// does).
	// +optional
	Password string `json:"password,omitempty"`

	// CAData is the PEM-encoded CA bundle used to verify the Argo CD server's TLS
	// certificate. Data key "caData".
	// +optional
	CAData string `json:"caData,omitempty"`

	// Insecure skips TLS verification when talking to the Argo CD server. Data key
	// "insecure". Defaults to false and is always stamped explicitly ("true" or
	// "false"), exactly like the UI writes it.
	// +optional
	Insecure bool `json:"insecure,omitempty"`
}

// ImageStoreProtocol discriminates the object store implementations an os-image-store
// connector can point at. Each value has a block of its own in ConnectorImageStoreSpec,
// so a second implementation becomes a new block, not a new connector type, the same
// way a NodeProvider discriminates between spec.metal3 and spec.kubeVirt.
// +enum
type ImageStoreProtocol string

const (
	// ImageStoreProtocolS3 is an S3-compatible object store reached with SigV4 pre-signed
	// URLs: AWS S3 itself, MinIO, Ceph RGW, SeaweedFS and the like.
	ImageStoreProtocolS3 ImageStoreProtocol = "s3"
)

// ConnectorImageStoreSpec configures the object store the platform uploads OSImage bytes
// to and serves them from. An OSImage names the connector through spec.connectorRef and
// the platform mints pre-signed URLs from these credentials on the image's behalf: the
// client uploading the bytes and the node provider reading them only ever see a URL,
// never the keys. The Secret behind this payload is read by pkg/osimage/store, whose
// loader is the arbiter of the Data keys named in the field comments below.
//
// Two fields are credentials under the Connector type's view rule: s3.secretKey and
// s3.sessionToken are served to a caller who could update this connector and absent for
// every other caller. On update an empty secretKey keeps the stored one, so a read without
// the credential written back never clears the key. The access key is not a credential:
// SigV4 puts it into every pre-signed URL as X-Amz-Credential, so a caller who could update
// this connector is served it, and an empty value on update is not "keep". Like every other
// payload field except protocol, it is absent from the redacted view.
type ConnectorImageStoreSpec struct {
	// Protocol selects the object store implementation and which block below applies.
	// Required; only s3 is supported. It is not stored on the backing Secret because
	// every os-image-store Secret currently describes an S3-compatible store.
	// +required
	Protocol ImageStoreProtocol `json:"protocol"`

	// S3 carries the settings of an S3-compatible store. Required when protocol is s3.
	// +optional
	S3 *ConnectorImageStoreS3Spec `json:"s3,omitempty"`
}

// ConnectorImageStoreS3Spec is the S3 block of an os-image-store connector. Every field maps
// to one Data key of the backing Secret (given in each field comment), spelled exactly as
// pkg/osimage/store reads it.
type ConnectorImageStoreS3Spec struct {
	// Endpoint is the object store address, for example https://minio.example.com. Data
	// key "endpoint". Optional: empty means AWS S3 itself, resolved from the region. When
	// set it must be an http or https URL.
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// Bucket is the bucket image objects live in. Data key "bucket". Required.
	// +required
	Bucket string `json:"bucket"`

	// Region is the region SigV4 signs for. Data key "region". Required also for stores
	// that ignore it, because the signature covers it and there is nothing to fall back
	// to.
	// +required
	Region string `json:"region"`

	// ForcePathStyle addresses the bucket as a path (https://endpoint/bucket/key) instead
	// of a virtual host (https://bucket.endpoint/key), which most self-hosted stores need.
	// Data key "forcePathStyle", always stamped as "true" or "false"; a missing stored
	// value projects as false, which is how the store loader reads it too.
	// +optional
	ForcePathStyle bool `json:"forcePathStyle,omitempty"`

	// AccessKey is the access key ID of the static credentials. Data key "accessKey".
	// Required. The credentials must be long-lived: a pre-signed URL is only valid as
	// long as the credentials that signed it, and upload URLs live 24 hours.
	// +required
	AccessKey string `json:"accessKey"`

	// SecretKey is the secret access key of the static credentials. Data key "secretKey".
	// Required on create. A credential (see the type comment): an empty value on update
	// keeps the stored key, and a pre-existing Secret without a stored key cannot be
	// updated through this API until the request supplies one.
	// +optional
	SecretKey string `json:"secretKey,omitempty"`

	// SessionToken is set only for temporary credentials. Data key "sessionToken".
	// Optional; an empty value on update removes it. A credential (see the type comment).
	// Pre-signed URLs expire with the session, so this is meant for short-lived use.
	// +optional
	SessionToken string `json:"sessionToken,omitempty"`
}

// ConnectorObservabilitySpec configures the connection to an observability stack: the
// Grafana instance the platform proxies under /grafana/ and the metrics backend the
// observability gateway writes to and queries from. Every field except fleetDefault
// maps to one Data key of the backing Secret (given in each field comment), so direct
// Secret writers and this API stay byte-compatible in both directions.
//
// The four Grafana and metrics backend fields default on CREATE ONLY to the same
// in-cluster endpoints the UI pre-fills on create (the Prometheus and Grafana the
// platform ships), so a minimal create through this API produces the same Secret as
// an untouched UI create. Updates never default them. Only grafanaUrl is required
// afterwards: the metrics backend endpoints and the dashboard UID may be emptied, which
// leaves the gateway idle until they are set again, a state the platform tolerates and
// the UI form does not offer.
//
// observabilityGatewayCertSecretName appears both here and in
// status.observability.gatewayCertSecretName. That is one stored value reported twice,
// not two: a single Data key backs both, so they can never disagree. It stays settable
// here because leaving it empty and bringing your own certificate are different
// requests, and it is reported in status so a consumer finds the stored name next to
// the other gateway observations instead of having to read spec for one of them.
//
// This type has no credential fields: every Secret the connector refers to is named,
// not embedded, so the whole payload is served to every caller who may read it.
type ConnectorObservabilitySpec struct {
	// FleetDefault marks this connector as the one backing the platform-wide fleet
	// metrics Grafana. It is projected from and stamped to the
	// platform.vcluster.com/fleet-observability-connector label on the backing Secret,
	// which is the API for it: a metadata write carrying the label with a value that
	// disagrees with this field is refused. At most one connector may be the fleet
	// default: a write claiming it while another connector (including a terminating
	// one) holds it is rejected with a 409 Conflict naming the holder.
	//
	// The field is tri-state. Unset carries no opinion: on create the connector
	// claims the label automatically only when no holder exists anywhere (exactly
	// like the UI's first create), on update it keeps the connector's current holder
	// state and can never demote it. True claims the label explicitly (or is
	// rejected naming the holder), false never claims it and releases it when this
	// connector is the holder. Reads carry the value true when the label is present
	// and leave the field unset when it is absent, never an explicit false, so a
	// read-modify-write round trip of a non-holder stays a no-op instead of
	// becoming a demotion request. An explicit false is accepted and demotes, but is
	// never echoed back, so declarative tooling that keeps fleetDefault: false in its
	// manifest sees a permanent diff on that field.
	//
	// Moving the fleet default to another connector takes two writes: demote the
	// holder with an explicit false first, then claim with true on the new
	// connector. A claim while a holder exists is never an implicit steal.
	//
	// A pre-existing sparse Secret missing the grafanaUrl key cannot be demoted
	// through this API: write-time validation rejects the whole update until the
	// field is filled in. The escape hatch is removing the fleet label on the backing
	// Secret directly.
	// +optional
	FleetDefault *bool `json:"fleetDefault,omitempty"`

	// GrafanaURL is the URL of the Grafana instance the platform proxies under
	// /grafana/. Data key "grafanaUrl". Must be an http or https URL; an empty value
	// defaults on CREATE ONLY to the in-cluster Grafana the platform ships
	// (https://grafana.<platform namespace>.svc.cluster.local). An emptied value on
	// update is rejected, matching the UI edit form.
	// +optional
	GrafanaURL string `json:"grafanaUrl,omitempty"`

	// GrafanaInsecureSkipVerify skips TLS verification when the platform talks to
	// Grafana. Data key "grafanaInsecureSkipVerify": true stores "true", false stores
	// no key. A write replaces the Secret from the request, so a "false" the UI wrote
	// earlier does not survive an update; every consumer reads an absent key and
	// "false" the same way.
	// +optional
	GrafanaInsecureSkipVerify bool `json:"grafanaInsecureSkipVerify,omitempty"`

	// FleetObservabilityDashboardUID is the UID of the Grafana dashboard the fleet
	// observability view embeds. Data key "fleetObservabilityDashboardUid". An empty
	// value defaults on CREATE ONLY to "vcluster-fleet-observability", the dashboard
	// the platform provisions. It may be emptied on update: the fleet view then shows
	// an empty state until a dashboard is named again.
	// +optional
	FleetObservabilityDashboardUID string `json:"fleetObservabilityDashboardUid,omitempty"`

	// MetricsBackendOtlpEndpoint is the metrics backend's OTLP write endpoint the
	// observability gateway forwards tenant metrics to. Data key
	// "metricsBackendOtlpEndpoint". Must be an http or https URL; an empty value
	// defaults on CREATE ONLY to the in-cluster Prometheus the platform ships
	// (https://prom-prometheus-server.<platform namespace>.svc.cluster.local/api/v1/otlp).
	// It may be emptied on update: the gateway controller skips the connector until
	// both metrics backend endpoints are set.
	// +optional
	MetricsBackendOtlpEndpoint string `json:"metricsBackendOtlpEndpoint,omitempty"`

	// MetricsBackendPromQLEndpoint is the metrics backend's PromQL query endpoint.
	// Data key "metricsBackendPromQLEndpoint". Must be an http or https URL; an
	// empty value defaults on CREATE ONLY to the in-cluster Prometheus the platform
	// ships (https://prom-prometheus-server.<platform namespace>.svc.cluster.local).
	// It may be emptied on update: the gateway controller skips the connector until
	// both metrics backend endpoints are set.
	// +optional
	MetricsBackendPromQLEndpoint string `json:"metricsBackendPromQLEndpoint,omitempty"`

	// MetricsBackendCertSecretName references the kubernetes.io/tls Secret with the
	// client certificate the gateway presents to the metrics backend for mTLS. Data
	// key "metricsBackendCertSecretName". Optional.
	// +optional
	MetricsBackendCertSecretName string `json:"metricsBackendCertSecretName,omitempty"`

	// MetricsBackendInsecureSkipVerify disables TLS verification against the metrics
	// backend on both the OTLP write and PromQL query paths. Data key
	// "metricsBackendInsecureSkipVerify": true stores "true", false stores no key. A
	// write replaces the Secret from the request, so a "false" the UI wrote earlier
	// does not survive an update; every consumer reads an absent key and "false" the
	// same way.
	// +optional
	MetricsBackendInsecureSkipVerify bool `json:"metricsBackendInsecureSkipVerify,omitempty"`

	// MetricsBackendServerName overrides the TLS server name the query proxy expects
	// on the metrics backend's certificate, for backends fronted by a proxy or an SNI
	// router. It applies to the PromQL reads only; the OTLP endpoint's certificate must
	// still match the endpoint hostname. Data key "metricsBackendServerName". Optional;
	// the UI has no field for it and the gateway documentation describes it.
	// +optional
	MetricsBackendServerName string `json:"metricsBackendServerName,omitempty"`

	// GrafanaCertSecretName references the kubernetes.io/tls Secret Grafana serves;
	// the /grafana/ proxy trusts its ca.crt. Data key "grafanaCertSecretName".
	// Optional.
	// +optional
	GrafanaCertSecretName string `json:"grafanaCertSecretName,omitempty"`

	// ObservabilityGatewayCertSecretName references the kubernetes.io/tls Secret the
	// observability gateway serves on the OTLP write receiver and metrics
	// query-proxy. Data key "observabilityGatewayCertSecretName". Optional, and the
	// two cases are different requests rather than set-versus-unset:
	//
	// Set it to bring your own certificate. The Secret must exist before you name it
	// here: the platform never modifies and never deletes a certificate Secret it did
	// not create, but it does not check that a named Secret exists. Whenever a
	// reconcile finds no Secret under the name (with the default ClusterIP service
	// type that is the first reconcile after the create), the gateway self-issues a
	// certificate under that name and owns it from then on, rotating it when its
	// SANs change and deleting it with the connector. The name is reported in status
	// either way; nothing there says who created the Secret.
	//
	// Leave it empty and the platform owns the certificate: the gateway controller
	// derives a name, self-issues a certificate under it and writes the derived name
	// back into this same Data key. A later read therefore projects the derived name
	// here. Clearing the field again removes the key, and the controller then derives
	// the same name from the connector's own name and writes it back, reusing the
	// certificate already stored under it, so clearing a platform-derived name is not
	// a way to force a rotation. Clearing a name you brought yourself is the real
	// handover: there the controller derives its own name and self-issues under it.
	//
	// status.observability.gatewayCertSecretName reports the same single stored value,
	// alongside the other gateway observations.
	// +optional
	ObservabilityGatewayCertSecretName string `json:"observabilityGatewayCertSecretName,omitempty"`

	// ServiceType selects the observability gateway Service type, ClusterIP or
	// LoadBalancer. Data key "serviceType". Empty keeps the key absent and means
	// ClusterIP, exactly how consumers treat a Secret without the key; the implied
	// default is deliberately not materialized so an untouched UI create and an API
	// create stay byte-identical.
	// +optional
	ServiceType string `json:"serviceType,omitempty"`

	// ServiceAnnotations is raw YAML text with flat "key: value" pairs merged into
	// the gateway Service's annotations. Data key "serviceAnnotations". The gateway
	// controller stamps them onto the Service whatever its type, and the value is
	// stored as sent for any serviceType; it only has a visible effect on a
	// LoadBalancer Service, where the cloud provider reads it. The UI drops the value
	// on save for other service types; the API keeps it.
	// +optional
	ServiceAnnotations string `json:"serviceAnnotations,omitempty"`

	// Destinations is raw YAML text listing additional OTLP destinations the
	// gateway forwards to. Data key "destinations". Each entry has:
	//
	//   - name: unique, lowercase, up to 40 characters
	//   - signal: metrics or logs
	//   - endpoint: an http or https URL
	//   - insecureSkipVerify: optional, skips TLS verification
	//   - certSecretName: optional, a kubernetes.io/tls Secret for mTLS
	//
	// The list is checked with the gateway controller's parser and stored as sent.
	// +optional
	Destinations string `json:"destinations,omitempty"`

	// GatewayResources overrides the resource requests and limits of the gateway's
	// write container. Data key "gatewayResources", stored as the JSON serialization
	// of this value, which the gateway controller parses the same way it parses a YAML
	// block a hand-written Secret carries. Unset leaves the key absent and the
	// controller's defaults in effect. A stored value the controller could not parse
	// projects as unset; the controller reports that failure itself.
	// +optional
	GatewayResources *corev1.ResourceRequirements `json:"gatewayResources,omitempty"`

	// QueryProxyResources overrides the resource requests and limits of the gateway's
	// query proxy container. Data key "queryProxyResources"; the same storage and
	// parsing rules as gatewayResources apply.
	// +optional
	QueryProxyResources *corev1.ResourceRequirements `json:"queryProxyResources,omitempty"`
}

// ConnectorNetBoxSpec configures the connection to a NetBox instance the metal3 node
// provider reads its bare metal inventory from. Every field maps to one Data key of the
// backing Secret (given in each field comment), so a Secret written through this API is
// exactly what pkg/netbox reads: a NodeProvider names it through
// spec.metal3.netBox.secretRef with the connector's name and the platform namespace.
//
// One field is a credential under the Connector type's view rule: apiToken is served to
// a caller who could update this connector and empty for every other caller, and on
// update an empty value keeps the stored one, so a redacted read written back never
// clears the token.
type ConnectorNetBoxSpec struct {
	// URL is the base URL of the NetBox instance, for example https://netbox.example.com.
	// Data key "url". Required, must be an http or https URL.
	// +required
	URL string `json:"url"`

	// APIToken is the NetBox API token requests are authenticated with. Data key
	// "apiToken". Required on create. A credential (see the type comment): an empty
	// value on update keeps the stored token, and a pre-existing Secret without a
	// stored token cannot be updated through this API until the request supplies one.
	// The token must be one created in the NetBox UI: NetBox rejects tokens obtained
	// from its provisioning endpoint.
	// +optional
	APIToken string `json:"apiToken,omitempty"`

	// Insecure skips TLS certificate verification of the NetBox URL. Data key
	// "insecure", always stamped as "true" or "false" exactly like the UI writes it; a
	// missing or unparsable stored value projects as false, which is how the NetBox
	// client reads it too.
	// +optional
	Insecure bool `json:"insecure,omitempty"`
}

// ConnectorStatus reports what the platform knows about a connector. Nothing here is
// settable through this API: a create drops whatever status the request carried and an
// update replaces it with the stored object's, so a client can never write status and
// can never clobber what a controller published.
//
// Conditions are an open list in the house Cluster-API shape, so a controller may add a
// condition type this API does not name without an API change. Two sources fill the
// list. The Configured condition is computed on read by validating the stored payload
// with the type's codec, so it needs no controller and costs no persistence. Everything
// only a controller can know (reachability, reconcile outcomes, the observability
// gateway observations) is published by controllers into a server-owned annotation on
// the backing Secret and unpacked here on read. No caller ever sees that annotation: it
// is server-owned, so it is never served, a write naming it is refused, and it stays on
// the Secret as the controller left it.
type ConnectorStatus struct {
	// Conditions holds the conditions observed for this connector. The list is open:
	// v1 names Configured, Reachable (and Authenticated where credentials fail
	// separately from connectivity) and Reconciled, but a controller may publish
	// other types.
	//
	// The computed Configured condition reports the connector's own
	// metadata.creationTimestamp as its lastTransitionTime, unless a controller has
	// published a Configured in the same state, in which case that condition's
	// transition time is adopted. The API recomputes this condition on every read and
	// keeps no record of when the outcome last flipped, so the creation time is a
	// stand-in that the real transition can only have happened at or after, never a
	// transition the server observed.
	// +optional
	Conditions agentstoragev1.Conditions `json:"conditions,omitempty"`

	// Observability holds the observations that only the observability connector type
	// has and is set only when spec.type is observability. Its fields are what the
	// gateway controller reports, not a copy of the spec: the endpoint, CA and
	// certificate facts exist nowhere in the configuration. The other two types have no
	// type-specific facts in v1 and therefore have no status section of their own.
	// +optional
	Observability *ConnectorObservabilityStatus `json:"observability,omitempty"`
}

// ConnectorObservabilityStatus holds the observability gateway facts the platform
// observes rather than accepts. They are observations and references for display, not
// conditions: readiness rides the conditions list instead.
//
// All but one field stay absent until a controller publishes them, so an unpublished
// connector reports an absent section rather than zero values that would read as
// facts. The exception is GatewayCertSecretName, which the API fills from the
// connector's own stored configuration on every read, with no controller involved; its
// own comment says so in full.
type ConnectorObservabilityStatus struct {
	// GatewayEndpoint is the address collectors send OTLP traffic to and the metrics
	// query proxy is reached at, taken from the gateway Service's load balancer
	// hostname or IP.
	// +optional
	GatewayEndpoint string `json:"gatewayEndpoint,omitempty"`

	// GatewayCA is the PEM-encoded CA certificate a collector must trust to talk to
	// the gateway, so it can be copied straight out of a connector into a collector
	// configuration.
	// +optional
	GatewayCA string `json:"gatewayCA,omitempty"`

	// GatewayCertSecretName is the name of the kubernetes.io/tls Secret the
	// observability gateway is configured to serve, as stored on the connector: either
	// the one an admin brought through
	// spec.observability.observabilityGatewayCertSecretName, or the one the gateway
	// controller derived, self-issued and wrote back when that field was left empty.
	//
	// It is the same single Data key the spec field is backed by, so the two can never
	// report different names; it is repeated here so a consumer reading status finds
	// the certificate reference next to the other gateway observations. Unlike the rest
	// of this section, which stays absent until a controller publishes it, the API
	// fills this field from the connector's stored configuration on every read, so no
	// controller has to publish it and a published copy never wins. It stays absent
	// until the gateway controller has written its derived name back: until then the
	// name in effect is the one the controller and the datasource consumers derive
	// from the connector name, and this field does not report it.
	// +optional
	GatewayCertSecretName string `json:"gatewayCertSecretName,omitempty"`

	// CertExpiresAt is the NotAfter of the certificate the gateway serves. It is a
	// pointer so an unpublished expiry is absent rather than the zero time, which
	// would read as a certificate that expired in year one.
	// +optional
	CertExpiresAt *metav1.Time `json:"certExpiresAt,omitempty"`

	// CertSelfIssued reports whether the platform issued the gateway certificate
	// itself rather than the admin bringing one. It is a pointer because false is a
	// meaningful answer here and must stay distinguishable from "no controller has
	// published cert facts yet".
	// +optional
	CertSelfIssued *bool `json:"certSelfIssued,omitempty"`
}
