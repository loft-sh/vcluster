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

func InsecureTransport() *http.Transport {
	newTransport := CloneDefaultTransport()
	newTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	return newTransport
}
