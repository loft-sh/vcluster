package storage

import (
	"context"
	"fmt"

	snapshotapi "github.com/loft-sh/api/v4/pkg/snapshot"
	"github.com/loft-sh/api/v4/pkg/snapshot/storage/azure"
	"github.com/loft-sh/api/v4/pkg/snapshot/storage/container"
	"github.com/loft-sh/api/v4/pkg/snapshot/storage/oci"
	"github.com/loft-sh/api/v4/pkg/snapshot/storage/s3"
	"github.com/loft-sh/api/v4/pkg/snapshot/storage/types"
	"k8s.io/klog/v2"
)

func CreateStore(ctx context.Context, options *snapshotapi.Options) (types.Storage, error) {
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
