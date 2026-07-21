package http

import (
	"crypto/tls"
	"net/http"
)

func CloneDefaultTransport() *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// we disable http2 as Kubernetes has problems with this
	transport.ForceAttemptHTTP2 = false
	return transport
}

// Transport returns a cloned default transport with TLS verification
// controlled by the insecure parameter. When insecure is true, TLS
// certificate verification is skipped.
func Transport(insecure bool) *http.Transport {
	newTransport := CloneDefaultTransport()
	if insecure {
		newTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return newTransport
}

func InsecureTransport() *http.Transport {
	return Transport(true)
}
