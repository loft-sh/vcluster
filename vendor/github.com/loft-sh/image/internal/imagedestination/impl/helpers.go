package impl

import (
	"github.com/loft-sh/image/internal/manifest"
	"github.com/loft-sh/image/internal/private"
)

// OriginalCandidateMatchesTryReusingBlobOptions returns true if the original blob passed to TryReusingBlobWithOptions
// is acceptable based on opts.
func OriginalCandidateMatchesTryReusingBlobOptions(opts private.TryReusingBlobOptions) bool {
	return manifest.CandidateCompressionMatchesReuseConditions(manifest.ReuseConditions{
		PossibleManifestFormats: opts.PossibleManifestFormats,
		RequiredCompression:     opts.RequiredCompression,
	}, opts.OriginalCompression)
}
