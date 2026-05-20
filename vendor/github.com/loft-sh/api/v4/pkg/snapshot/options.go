package snapshot

type Options struct {
	Type string `json:"type,omitempty"`

	S3        S3Options        `json:"s3"`
	Container ContainerOptions `json:"container"`
	OCI       OCIOptions       `json:"oci"`
	Azure     AzureOptions     `json:"azure"`

	Release        *HelmRelease `json:"release,omitempty"`
	IncludeVolumes bool         `json:"include-volumes,omitempty"`

	// DelegateFromCLIToCluster indicates that the snapshot options are saved in a Kubernetes Secret because the
	// snapshot/restore operation will be executed in a Kubernetes cluster.
	DelegateFromCLIToCluster bool `json:"delegateFromCLIToCluster,omitempty"`
}

func (o *Options) GetURL() string {
	if o == nil {
		return ""
	}

	switch o.Type {
	case "s3":
		return "s3://" + o.S3.Bucket + "/" + o.S3.Key
	case "container":
		return "container://" + o.Container.Path
	case "oci":
		return "oci://" + o.OCI.Repository
	case "azure":
		return o.Azure.BlobURL
	default:
		return ""
	}
}

type HelmRelease struct {
	ReleaseName      string `json:"releaseName"`
	ReleaseNamespace string `json:"releaseNamespace"`

	ChartName    string `json:"chartName"`
	ChartVersion string `json:"chartVersion"`

	Values []byte `json:"values"`
}

type VClusterConfig struct {
	ChartVersion string `json:"chartVersion"`
	Values       string `json:"values"`
}

type S3Options struct {
	Bucket string `json:"bucket,omitempty"`
	Key    string `json:"key,omitempty"`

	SkipClientCredentials bool `json:"skip-client-credentials,omitempty" url:"skip-client-credentials"`

	AccessKeyID     string `json:"access-key-id,omitempty" url:"access-key-id,base64"`
	SecretAccessKey string `json:"secret-access-key,omitempty" url:"secret-access-key,base64"`
	SessionToken    string `json:"session-token,omitempty" url:"session-token,base64"`

	Region    string `json:"region,omitempty" url:"region"`
	Profile   string `json:"profile,omitempty" url:"profile"`
	S3URL     string `json:"url,omitempty" url:"url,base64"`
	PublicURL string `json:"public-url,omitempty" url:"public-url,base64"`
	KmsKeyID  string `json:"kms-key-id,omitempty" url:"kms-key-id,base64"`
	Tagging   string `json:"tagging,omitempty" url:"tagging,base64"`

	S3ForcePathStyle      bool `json:"force-path-style,omitempty" url:"force-path-style"`
	InsecureSkipTLSVerify bool `json:"insecure-skip-tls-verify,omitempty" url:"insecure-skip-tls-verify"`

	CustomerKeyEncryptionFile string `json:"custom-key-encryption-file,omitempty" url:"custom-key-encryption-file,base64"`
	CredentialsFile           string `json:"credentials-file,omitempty" url:"credentials-file,base64"`
	ServerSideEncryption      string `json:"server-side-encryption,omitempty" url:"server-side-encryption,base64"`
	CACert                    string `json:"ca-cert,omitempty" url:"ca-cert,base64"`
	ChecksumAlgorithm         string `json:"checksum-algorithm,omitempty" url:"checksum-algorithm"`
}

type OCIOptions struct {
	Repository string `json:"repository,omitempty"`

	Username string `json:"username,omitempty" url:"username"`
	Password string `json:"password,omitempty" url:"password,base64"`

	SkipClientCredentials bool `json:"skip-client-credentials,omitempty" url:"skip-client-credentials"`
}

type ContainerOptions struct {
	Path string `json:"path,omitempty"`
}

type AzureOptions struct {
	// BlobURL is the full Azure Blob Storage URL.
	BlobURL string `json:"blob-url,omitempty"`

	// SAS is the Azure storage blob SAS token.
	SAS string `json:"sas,omitempty"`

	// SubscriptionID is the Azure subscription ID where the storage account is located.
	SubscriptionID string `json:"subscription-id,omitempty"`

	// ResourceGroup is the Azure resource group where the storage account is located.
	ResourceGroup string `json:"resource-group,omitempty"`

	// StorageKey is the Azure storage account access key.
	StorageKey string `json:"storage-key,omitempty"`

	// TenantID is the Azure tenant ID for service principal auth.
	TenantID string `json:"tenant-id,omitempty"`

	// ClientID is the Azure client ID for service principal auth.
	ClientID string `json:"client-id,omitempty"`

	// ClientSecret is the client secret for service principal auth.
	ClientSecret string `json:"client-secret,omitempty"`
}
