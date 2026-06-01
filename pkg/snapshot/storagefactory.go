package snapshot

import (
	"context"

	snapshotapi "github.com/loft-sh/api/v4/pkg/snapshot"
	snapshotstorage "github.com/loft-sh/api/v4/pkg/snapshot/storage"
	"github.com/loft-sh/api/v4/pkg/snapshot/storage/types"
)

func CreateStore(ctx context.Context, options *snapshotapi.Options) (types.Storage, error) {
	return snapshotstorage.CreateStore(ctx, options)
}
