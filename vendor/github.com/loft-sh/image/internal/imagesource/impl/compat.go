package impl

import (
	"github.com/loft-sh/image/internal/private"
)

// Compat implements the obsolete parts of types.ImageSource
// for implementations of private.ImageSource.
// See AddCompat below.
type Compat struct {
	src private.ImageSourceInternalOnly
}

// AddCompat initializes Compat to implement the obsolete parts of types.ImageSource
// for implementations of private.ImageSource.
//
// Use it like this:
//
//	type yourSource struct {
//		impl.Compat
//		…
//	}
//
//	src := &yourSource{…}
//	src.Compat = impl.AddCompat(src)
func AddCompat(src private.ImageSourceInternalOnly) Compat {
	return Compat{src}
}
