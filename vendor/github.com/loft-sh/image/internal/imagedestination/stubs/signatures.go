package stubs

import (
	"context"
	"errors"
)

// NoSignaturesInitialize implements parts of private.ImageDestination
// for transports that don’t support storing signatures.
// See NoSignatures() below.
type NoSignaturesInitialize struct {
	message string
}

// NoSignatures creates a NoSignaturesInitialize, failing with message.
func NoSignatures(message string) NoSignaturesInitialize {
	return NoSignaturesInitialize{
		message: message,
	}
}

// SupportsSignatures returns an error (to be displayed to the user) if the destination certainly can't store signatures.
// Note: It is still possible for PutSignatures to fail if SupportsSignatures returns nil.
func (stub NoSignaturesInitialize) SupportsSignatures(ctx context.Context) error {
	return errors.New(stub.message)
}

// SupportsSignatures implements SupportsSignatures() that returns nil.
// Note that it might be even more useful to return a value dynamically detected based on
type AlwaysSupportsSignatures struct{}

// SupportsSignatures returns an error (to be displayed to the user) if the destination certainly can't store signatures.
// Note: It is still possible for PutSignatures to fail if SupportsSignatures returns nil.
func (stub AlwaysSupportsSignatures) SupportsSignatures(ctx context.Context) error {
	return nil
}
