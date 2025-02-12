package snapshot

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/snapshot/file"
	"github.com/loft-sh/vcluster/pkg/snapshot/oci"
	"github.com/loft-sh/vcluster/pkg/snapshot/s3"
	"github.com/spf13/pflag"
)

type Options struct {
	S3   s3.Options   `json:"s3"`
	File file.Options `json:"file"`
	OCI  oci.Options  `json:"oci"`
}

func AddFlags(flagSet *pflag.FlagSet, opts *Options) {
	s3.AddFlags(flagSet, &opts.S3)
	file.AddFlags(flagSet, &opts.File)
	oci.AddFlags(flagSet, &opts.OCI)
}

func Validate(storage string, options *Options) error {
	// storage needs to be either s3 or file
	if storage == "s3" {
		if options.S3.Key == "" {
			return fmt.Errorf("--s3-key must be specified")
		}
		if options.S3.Bucket == "" {
			return fmt.Errorf("--s3-bucket must be specified")
		}
	} else if storage == "file" {
		if options.File.Path == "" {
			return fmt.Errorf("--file-path must be specified")
		}
	} else if storage == "oci" {
		if options.OCI.Repository == "" {
			return fmt.Errorf("--oci-repository must be specified")
		}
	} else {
		return fmt.Errorf("--storage must be either 'file', 'oci' or 's3'")
	}

	return nil
}
