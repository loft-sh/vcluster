package snapshot

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/loft-sh/vcluster/pkg/snapshot/file"
	"github.com/loft-sh/vcluster/pkg/snapshot/oci"
	"github.com/loft-sh/vcluster/pkg/snapshot/s3"
	"github.com/spf13/pflag"
)

type Options struct {
	Type string `json:"type,omitempty"`

	S3   s3.Options   `json:"s3"`
	File file.Options `json:"file"`
	OCI  oci.Options  `json:"oci"`
}

func AddFlags(flagSet *pflag.FlagSet, opts *Options) {
	flagSet.StringVar(&opts.Type, "type", opts.Type, "The type of storage to snapshot / restore from. Can be either file, oci or s3")
	s3.AddFlags(flagSet, &opts.S3)
	file.AddFlags(flagSet, &opts.File)
	oci.AddFlags(flagSet, &opts.OCI)
}

func Parse(snapshotURL string, options *Options) error {
	parsedURL, err := url.Parse(snapshotURL)
	if err != nil {
		return fmt.Errorf("error parsing snapshotURL %s: %w", snapshotURL, err)
	}

	if parsedURL.Scheme != "s3" && parsedURL.Scheme != "file" && parsedURL.Scheme != "oci" {
		return fmt.Errorf("scheme needs to be 'oci', 's3' or 'file'")
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
			options.S3.Profile = parsedURL.Query().Get("tagging")
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
	case "file":
		if parsedURL.Host != "" {
			return fmt.Errorf("relative paths are not supported for file snapshots")
		} else if parsedURL.Path == "" {
			return fmt.Errorf("couldn't find path for url")
		}
		options.File.Path = parsedURL.Path
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
	}

	return nil
}

func Validate(options *Options) error {
	// storage needs to be either s3 or file
	if options.Type == "" {
		return fmt.Errorf("--type is required")
	} else if options.Type == "s3" {
		if options.S3.Key == "" {
			return fmt.Errorf("--s3-key must be specified")
		}
		if options.S3.Bucket == "" {
			return fmt.Errorf("--s3-bucket must be specified")
		}
	} else if options.Type == "file" {
		if options.File.Path == "" {
			return fmt.Errorf("--file-path must be specified")
		}
	} else if options.Type == "oci" {
		if options.OCI.Repository == "" {
			return fmt.Errorf("--oci-repository must be specified")
		}
	} else {
		return fmt.Errorf("--type must be either 'file', 'oci' or 's3'")
	}

	return nil
}
