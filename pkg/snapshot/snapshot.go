package snapshot

import (
	"github.com/loft-sh/vcluster/pkg/snapshot/file"
	"github.com/loft-sh/vcluster/pkg/snapshot/s3"
)

type Options struct {
	S3   s3.Options   `json:"s3"`
	File file.Options `json:"file"`
}
