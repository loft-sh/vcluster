package translate

import (
	"errors"
	"fmt"
)

var errUnsupportedReference = errors.New("unsupported gateway api reference")

// UnsupportedReferenceError is returned when a Gateway API reference points at a
// group/kind vCluster cannot translate to a host object (anything other than the
// supported core kinds). It is a terminal condition: retrying cannot succeed until
// the user changes the spec, so callers must surface it and stop rather than requeue.
type UnsupportedReferenceError struct {
	msg string
}

// Error returns the unsupported-reference error message.
func (e *UnsupportedReferenceError) Error() string {
	return e.msg
}

// Is reports whether target is the package unsupported-reference sentinel.
func (e *UnsupportedReferenceError) Is(target error) bool {
	return target == errUnsupportedReference
}

// IsUnsupportedReference reports whether err indicates a reference kind vCluster
// does not support translating.
func IsUnsupportedReference(err error) bool {
	return errors.Is(err, errUnsupportedReference)
}

func unsupportedReferencef(format string, args ...any) error {
	return &UnsupportedReferenceError{msg: fmt.Sprintf(format, args...)}
}
