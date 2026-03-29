package kubeclient

import "net/http"

// Option configures a client built by this package.
type Option func(*clientOptions)

type clientOptions struct {
	wrapTransport func(http.RoundTripper) http.RoundTripper
}

func applyOptions(opts []Option) *clientOptions {
	o := &clientOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// WithWrapTransport chains an http.RoundTripper wrapper onto any client built
// by this package. Multiple calls are composed in the order they are applied.
func WithWrapTransport(fn func(http.RoundTripper) http.RoundTripper) Option {
	return func(o *clientOptions) {
		if o.wrapTransport == nil {
			o.wrapTransport = fn
			return
		}
		prior := o.wrapTransport
		o.wrapTransport = func(rt http.RoundTripper) http.RoundTripper {
			return fn(prior(rt))
		}
	}
}
