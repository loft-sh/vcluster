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

	snapshotapi "github.com/loft-sh/api/v4/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/snapshot/azure"
	"github.com/loft-sh/vcluster/pkg/snapshot/oci"
	"github.com/loft-sh/vcluster/pkg/snapshot/options"
	"github.com/loft-sh/vcluster/pkg/snapshot/s3"
	"github.com/spf13/pflag"
)

func SetURLAndFillCredentials(ctx context.Context, snapshotOptions *snapshotapi.Options, url string, credentialsRequiredInCluster bool) error {
	err := Parse(url, snapshotOptions)
	if err != nil {
		return fmt.Errorf("failed to parse snapshot URL: %w", err)
	}
	err = Validate(snapshotOptions, false)
	if err != nil {
		return fmt.Errorf("invalid snapshot URL: %w", err)
	}
	switch snapshotOptions.Type {
	case "oci":
		oci.FillCredentials(&snapshotOptions.OCI, true)
	case "s3":
		s3.FillCredentials(&snapshotOptions.S3, true)
	case "azure":
		err := azure.FillCredentials(ctx, &snapshotOptions.Azure, credentialsRequiredInCluster)
		if err != nil {
			return fmt.Errorf("failed to fill azure credentials: %w", err)
		}
	}
	return nil
}

func Parse(snapshotURL string, snapshotOptions *snapshotapi.Options) error {
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

func ParseOptionsFromEnv() (*snapshotapi.Options, error) {
	snapshotOptions := os.Getenv(constants.VClusterStorageOptionsEnv)
	if snapshotOptions == "" {
		return &snapshotapi.Options{}, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(snapshotOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to decode storage options from env: %w", err)
	}

	opts := &snapshotapi.Options{}
	err = json.Unmarshal(decoded, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal storage options from env: %w", err)
	}

	return opts, nil
}

func Validate(options *snapshotapi.Options, isList bool) error {
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

func AddFlags(flags *pflag.FlagSet, options *snapshotapi.Options) {
	// AWS S3
	flags.StringVarP(&options.S3.KmsKeyID, "kms-key-id", "", "", "AWS KMS key ID that is configured for given S3 bucket. If set, aws-kms SSE will be used")
	flags.StringVarP(&options.S3.CustomerKeyEncryptionFile, "customer-key-encryption-file", "", "", "AWS customer key encryption file used for SSE-C. Mutually exclusive with kms-key-id")
	flags.StringVarP(&options.S3.ServerSideEncryption, "server-side-encryption", "", "", "AWS Server-Side encryption algorithm")
	flags.BoolVarP(&options.IncludeVolumes, "include-volumes", "", false, "Create CSI volume snapshots (shared and private nodes only). Deprecated: volume snapshot and restore will be removed in an upcoming release.")
	flags.StringVarP(&options.SnapshotTempDir, "snapshot-temp-dir", "", "", "Temporary directory for snapshot operations. If set to empty string, the OS default directory for temporary files will be used")
	azure.AddFlags(flags, &options.Azure)
}
