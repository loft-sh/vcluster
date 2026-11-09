package blobinfocache

import (
	"github.com/loft-sh/image/pkg/blobinfocache/memory"
	"github.com/loft-sh/image/types"
)

// DefaultCache returns the default BlobInfoCache implementation appropriate for sys.
func DefaultCache(sys *types.SystemContext) types.BlobInfoCache {
	return memory.New()
}

// CleanupDefaultCache removes the blob info cache directory.
// It deletes the cache directory but it does not affect any file or memory buffer currently
// in use.
func CleanupDefaultCache(sys *types.SystemContext) error {
	return nil
}
