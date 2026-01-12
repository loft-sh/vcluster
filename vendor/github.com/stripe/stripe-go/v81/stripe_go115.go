//go:build go1.15
// +build go1.15

package stripe

import "net/http"

// This init block is only compiled on Go 1.15 and above
// (per https://golang.org/cmd/go/#hdr-Build_constraints).
//
// Go 1.15 fixes a long-standing bug https://github.com/golang/go/issues/32441
// that led to HTTP/2 being disabled by default in stripe-go
// in https://github.com/stripe/stripe-go/pull/903.
//
// This init is guaranteed to execute after all package-level var
// initializations have been completed, so the initialization of
// the `httpClient` package-level var will have completed before we
// run this init.
//
// Once stripe-go drops support for major Go versions below 1.15, this
// conditional build and init block should be removed, in favor of making
// this HTTP client the default.
func init() {

	// Sets a default HTTP client that can utilize H/2 if the upstream supports it.
	SetHTTPClient(&http.Client{
		Timeout: defaultHTTPTimeout,
	})
}
