package snapshot

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/loft-sh/vcluster/pkg/snapshot/azure"
	"github.com/loft-sh/vcluster/pkg/snapshot/container"
	"github.com/loft-sh/vcluster/pkg/snapshot/oci"
	"github.com/loft-sh/vcluster/pkg/snapshot/options"
	"github.com/loft-sh/vcluster/pkg/snapshot/s3"
	"github.com/loft-sh/vcluster/pkg/snapshot/types"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"
)

const (
	// SnapshotReleaseKey stores info about the vCluster helm release
	SnapshotReleaseKey = "/vcluster/snapshot/release"
)

type Options struct {
	Type string `json:"type,omitempty"`

	S3        s3.Options        `json:"s3"`
	Container container.Options `json:"container"`
	OCI       oci.Options       `json:"oci"`
	Azure     azure.Options     `json:"azure"`

	Release        *HelmRelease `json:"release,omitempty"`
	IncludeVolumes bool         `json:"include-volumes,omitempty"`
}

func (o *Options) GetURL() string {
	var snapshotURL string
	switch o.Type {
	case "s3":
		snapshotURL = "s3://" + o.S3.Bucket + "/" + o.S3.Key
	case "container":
		snapshotURL = "container://" + o.Container.Path
	case "oci":
		snapshotURL = "oci://" + o.OCI.Repository
	case "azure":
		snapshotURL = o.Azure.BlobURL
	}

	return snapshotURL
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

func CreateStore(ctx context.Context, options *Options) (types.Storage, error) {
	if options.Type == "s3" {
		objectStore := s3.NewStore(klog.FromContext(ctx))
		err := objectStore.Init(&options.S3)
		if err != nil {
			return nil, fmt.Errorf("failed to init s3 object store: %w", err)
		}

		return objectStore, nil
	} else if options.Type == "container" {
		return container.NewStore(&options.Container), nil
	} else if options.Type == "oci" {
		return oci.NewStore(&options.OCI), nil
	} else if options.Type == "azure" {
		objectStore := azure.NewStore(klog.FromContext(ctx))
		err := objectStore.Init(&options.Azure)
		if err != nil {
			return nil, fmt.Errorf("failed to init Azure object store: %w", err)
		}
		return objectStore, nil
	}

	return nil, fmt.Errorf("unknown storage: %s", options.Type)
}

func Parse(snapshotURL string, snapshotOptions *Options) error {
	parsedURL, err := url.Parse(snapshotURL)
	if err != nil {
		return fmt.Errorf("error parsing snapshotURL %s: %w", snapshotURL, err)
	}

	supportedSchemes := []string{"oci", "s3", "container", "https"}
	if !slices.Contains(supportedSchemes, parsedURL.Scheme) {
		return fmt.Errorf("scheme needs to be one of %s", strings.Join(supportedSchemes, ", "))
	}
	snapshotOptions.Type = parsedURL.Scheme

	// depending on the type we parse differently
	switch snapshotOptions.Type {
	case "s3":
		// Zonal: https://BUCKET.s3express-euw1-az1.REGION.amazonaws.com/KEY
		// Global: https://BUCKET.s3.REGION.amazonaws.com/KEY
		// Format: s3://BUCKET/KEY
		if parsedURL.Host == "" {
			return fmt.Errorf("bucket name is missing from url, expected format: s3://BUCKET/KEY")
		} else if parsedURL.Path == "" {
			return fmt.Errorf("bucket key is missing from url, expected format: s3://BUCKET/KEY")
		}

		snapshotOptions.S3.Bucket = parsedURL.Host
		snapshotOptions.S3.Key = strings.TrimPrefix(parsedURL.Path, "/")
		err = options.PopulateStructFromMap(&snapshotOptions.S3, parsedURL.Query(), true)
		if err != nil {
			return fmt.Errorf("error parsing options: %w", err)
		}
	case "container":
		if parsedURL.Host != "" {
			return fmt.Errorf("relative paths are not supported for container snapshots")
		} else if parsedURL.Path == "" {
			return fmt.Errorf("couldn't find path for url")
		}
		snapshotOptions.Container.Path = parsedURL.Path
	case "oci":
		if parsedURL.Path == "" {
			return fmt.Errorf("unexpected format, need oci://my-registry.com/my-repo")
		} else if parsedURL.Host == "" {
			return fmt.Errorf("unexpected format, need oci://my-registry.com/my-repo")
		}
		snapshotOptions.OCI.Repository = path.Join(parsedURL.Host, parsedURL.Path)
		if parsedURL.User != nil {
			snapshotOptions.OCI.Username = parsedURL.User.Username()
			snapshotOptions.OCI.Password, _ = parsedURL.User.Password()
		}
		err = options.PopulateStructFromMap(&snapshotOptions.OCI, parsedURL.Query(), true)
		if err != nil {
			return fmt.Errorf("error parsing options: %w", err)
		}
	case "https":
		// Azure blob storage support
		snapshotOptions.Type = "azure"
		snapshotOptions.Azure.BlobURL = snapshotURL
	}

	return nil
}

func ParseOptionsFromEnv() (*Options, error) {
	snapshotOptions := os.Getenv("VCLUSTER_STORAGE_OPTIONS")
	if snapshotOptions == "" {
		return &Options{}, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(snapshotOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to decode storage options from env: %w", err)
	}

	opts := &Options{}
	err = json.Unmarshal(decoded, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal storage options from env: %w", err)
	}

	return opts, nil
}

func Validate(options *Options, isList bool) error {
	// storage needs to be either s3 or file
	if options.Type == "s3" {
		if !isList && options.S3.Key == "" {
			return fmt.Errorf("key must be specified via s3://BUCKET/KEY")
		}
		if options.S3.Bucket == "" {
			return fmt.Errorf("bucket must be specified via s3://BUCKET/KEY")
		}
	} else if options.Type == "container" {
		if options.Container.Path == "" {
			return fmt.Errorf("path must be specified via container:///PATH")
		}
	} else if options.Type == "oci" {
		if options.OCI.Repository == "" {
			return fmt.Errorf("repository must be specified via oci://repository")
		}
	} else if options.Type == "azure" {
		if options.Azure.BlobURL == "" {
			return fmt.Errorf("blob URL must be specified")
		}
	} else {
		return fmt.Errorf("type must be either 'container', 'oci', 's3', or 'azure'")
	}

	return nil
}

func AddFlags(flags *pflag.FlagSet, options *Options) {
	// AWS S3
	flags.StringVarP(&options.S3.KmsKeyID, "kms-key-id", "", "", "AWS KMS key ID that is configured for given S3 bucket. If set, aws-kms SSE will be used")
	flags.StringVarP(&options.S3.CustomerKeyEncryptionFile, "customer-key-encryption-file", "", "", "AWS customer key encryption file used for SSE-C. Mutually exclusive with kms-key-id")
	flags.StringVarP(&options.S3.ServerSideEncryption, "server-side-encryption", "", "", "AWS Server-Side encryption algorithm")
	flags.BoolVarP(&options.IncludeVolumes, "include-volumes", "", false, "Create CSI volume snapshots (shared and private nodes only)")

	// Azure Blob
	flags.StringVarP(&options.Azure.SubscriptionID, "azure-subscription-id", "", "", "Azure subscription ID where the storage account is located")
	flags.StringVarP(&options.Azure.ResourceGroup, "azure-resource-group", "", "", "Azure resource group where the storage account is located")
}
