package snapshot

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"

	"github.com/loft-sh/vcluster/pkg/snapshot/container"
	"github.com/loft-sh/vcluster/pkg/snapshot/oci"
	"github.com/loft-sh/vcluster/pkg/snapshot/s3"
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

	Release *HelmRelease `json:"release,omitempty"`
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

type Storage interface {
	Target() string
	PutObject(ctx context.Context, body io.Reader) error
	GetObject(ctx context.Context) (io.ReadCloser, error)
}

func CreateStore(ctx context.Context, options *Options) (Storage, error) {
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
	}

	return nil, fmt.Errorf("unknown storage: %s", options.Type)
}

func Parse(snapshotURL string, options *Options) error {
	parsedURL, err := url.Parse(snapshotURL)
	if err != nil {
		return fmt.Errorf("error parsing snapshotURL %s: %w", snapshotURL, err)
	}

	if parsedURL.Scheme != "s3" && parsedURL.Scheme != "container" && parsedURL.Scheme != "oci" {
		return fmt.Errorf("scheme needs to be 'oci', 's3' or 'container'")
	}
	options.Type = parsedURL.Scheme

	// depending on the type we parse differently
	switch options.Type {
	case "s3":
		// Zonal: https://BUCKET.s3express-euw1-az1.REGION.amazonaws.com/KEY
		// Global: https://BUCKET.s3.REGION.amazonaws.com/KEY
		// Format: s3://BUCKET/KEY
		if parsedURL.Host == "" {
			return fmt.Errorf("bucket name is missing from url, expected format: s3://BUCKET/KEY")
		} else if parsedURL.Path == "" {
			return fmt.Errorf("bucket key is missing from url, expected format: s3://BUCKET/KEY")
		}

		options.S3.Bucket = parsedURL.Host
		options.S3.Key = strings.TrimPrefix(parsedURL.Path, "/")
		if parsedURL.Query().Get("region") != "" {
			options.S3.Region = parsedURL.Query().Get("region")
		}
		if parsedURL.Query().Get("profile") != "" {
			options.S3.Profile = parsedURL.Query().Get("profile")
		}
		if parsedURL.Query().Get("tagging") != "" {
			options.S3.Tagging = parsedURL.Query().Get("tagging")
		}
		if parsedURL.Query().Get("access-key-id") != "" {
			options.S3.AccessKeyID = parsedURL.Query().Get("access-key-id")
		}
		if parsedURL.Query().Get("secret-access-key") != "" {
			options.S3.SecretAccessKey = parsedURL.Query().Get("secret-access-key")
		}
		if parsedURL.Query().Get("session-token") != "" {
			options.S3.SessionToken = parsedURL.Query().Get("session-token")
		}
		if parsedURL.Query().Get("skip-client-credentials") != "" {
			options.S3.SkipClientCredentials = parsedURL.Query().Get("skip-client-credentials") == "true"
		}
		if parsedURL.Query().Get("credentials-file") != "" {
			options.S3.CredentialsFile = parsedURL.Query().Get("credentials-file")
		}
		if parsedURL.Query().Get("ca-cert") != "" {
			options.S3.CaCert = parsedURL.Query().Get("ca-cert")
		}
		if parsedURL.Query().Get("checksum-algorithm") != "" {
			options.S3.ChecksumAlgorithm = parsedURL.Query().Get("checksum-algorithm")
		}
		if parsedURL.Query().Get("server-side-encryption") != "" {
			options.S3.ServerSideEncryption = parsedURL.Query().Get("server-side-encryption")
		}
		if parsedURL.Query().Get("kms-key-id") != "" {
			options.S3.KmsKeyID = parsedURL.Query().Get("kms-key-id")
		}
		if parsedURL.Query().Get("public-url") != "" {
			options.S3.PublicURL = parsedURL.Query().Get("public-url")
		}
		if parsedURL.Query().Get("insecure-skip-tls-verify") != "" {
			options.S3.InsecureSkipTLSVerify = parsedURL.Query().Get("insecure-skip-tls-verify") == "true"
		}
		if parsedURL.Query().Get("custom-key-encryption-file") != "" {
			options.S3.CustomerKeyEncryptionFile = parsedURL.Query().Get("custom-key-encryption-file")
		}
		if parsedURL.Query().Get("force-path-style") != "" {
			options.S3.S3ForcePathStyle = parsedURL.Query().Get("force-path-style") == "true"
		}
		if parsedURL.Query().Get("s3-url") != "" {
			options.S3.S3URL = parsedURL.Query().Get("s3-url")
		}
	case "container":
		if parsedURL.Host != "" {
			return fmt.Errorf("relative paths are not supported for container snapshots")
		} else if parsedURL.Path == "" {
			return fmt.Errorf("couldn't find path for url")
		}
		options.Container.Path = parsedURL.Path
	case "oci":
		if parsedURL.Path == "" {
			return fmt.Errorf("unexpected format, need oci://my-registry.com/my-repo")
		} else if parsedURL.Host == "" {
			return fmt.Errorf("unexpected format, need oci://my-registry.com/my-repo")
		}
		options.OCI.Repository = path.Join(parsedURL.Host, parsedURL.Path)
		if parsedURL.User != nil {
			options.OCI.Username = parsedURL.User.Username()
			options.OCI.Password, _ = parsedURL.User.Password()
		}
		if parsedURL.Query().Get("username") != "" {
			options.OCI.Username = parsedURL.Query().Get("username")
		}
		if parsedURL.Query().Get("password") != "" {
			options.OCI.Password = parsedURL.Query().Get("password")
		}
		if parsedURL.Query().Get("skip-client-credentials") != "" {
			options.OCI.SkipClientCredentials = parsedURL.Query().Get("skip-client-credentials") == "true"
		}
	}

	return nil
}

func Validate(options *Options) error {
	// storage needs to be either s3 or file
	if options.Type == "s3" {
		if options.S3.Key == "" {
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
	} else {
		return fmt.Errorf("type must be either 'container', 'oci' or 's3'")
	}

	return nil
}
