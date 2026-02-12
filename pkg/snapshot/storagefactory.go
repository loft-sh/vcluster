package snapshot

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/snapshot/azure"
	"github.com/loft-sh/vcluster/pkg/snapshot/container"
	"github.com/loft-sh/vcluster/pkg/snapshot/oci"
	"github.com/loft-sh/vcluster/pkg/snapshot/s3"
	"github.com/loft-sh/vcluster/pkg/snapshot/types"
	"k8s.io/klog/v2"
)

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
		objectStore, err := azure.NewStore(ctx, &options.Azure, klog.FromContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("failed to create Azure object store: %w", err)
		}
		return objectStore, nil
	}

	return nil, fmt.Errorf("unknown storage: %s", options.Type)
}
