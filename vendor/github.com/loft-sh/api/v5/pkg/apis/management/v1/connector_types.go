package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v5/pkg/apis/loft/storage/v1"
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

	// ImageStore is the os-image-store connector payload. Exactly one payload section
	// must be set. The fields of its protocol
	// block mirror the Data keys of the backing Secret one to one, so a connector
	// written through this API is exactly what the OSImage store loader reads.
	// +optional
	ImageStore *ConnectorImageStoreSpec `json:"imageStore,omitempty"`
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
}
