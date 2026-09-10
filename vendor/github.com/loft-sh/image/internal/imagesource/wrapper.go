package imagesource

import (
	"github.com/loft-sh/image/internal/imagesource/stubs"
	"github.com/loft-sh/image/internal/private"
	"github.com/loft-sh/image/types"
)

// wrapped provides the private.ImageSource operations
// for a source that only implements types.ImageSource
type wrapped struct {
	stubs.NoGetBlobAtInitialize

	types.ImageSource
}

// FromPublic(src) returns an object that provides the private.ImageSource API
//
// Eventually, we might want to expose this function, and methods of the returned object,
// as a public API (or rather, a variant that does not include the already-superseded
// methods of types.ImageSource, and has added more future-proofing), and more strongly
// deprecate direct use of types.ImageSource.
//
// NOTE: The returned API MUST NOT be a public interface (it can be either just a struct
// with public methods, or perhaps a private interface), so that we can add methods
// without breaking any external implementers of a public interface.
func FromPublic(src types.ImageSource) private.ImageSource {
	if src2, ok := src.(private.ImageSource); ok {
		return src2
	}
	return &wrapped{
		NoGetBlobAtInitialize: stubs.NoGetBlobAt(src.Reference()),

		ImageSource: src,
	}
}
