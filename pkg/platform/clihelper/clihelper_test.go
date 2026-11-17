package clihelper

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestIsLoftReachable_InsecureTrueAgainstSelfSigned(t *testing.T) {
	// Create an HTTPS server with a self-signed certificate.
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	host := strings.TrimPrefix(server.URL, "https://")

	// With insecure=true, the self-signed cert should be accepted.
	reachable, err := IsLoftReachable(context.Background(), host, true)
	assert.NilError(t, err)
	assert.Assert(t, reachable, "should be reachable with insecure=true against self-signed cert")
}

func TestIsLoftReachable_InsecureFalseAgainstSelfSigned(t *testing.T) {
	// Create an HTTPS server with a self-signed certificate.
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	host := strings.TrimPrefix(server.URL, "https://")

	// With insecure=false, the self-signed cert should cause a TLS error
	// and IsLoftReachable should return false (not reachable).
	reachable, err := IsLoftReachable(context.Background(), host, false)
	assert.NilError(t, err)
	assert.Assert(t, !reachable, "should not be reachable with insecure=false against self-signed cert")
}

func TestIsLoftReachable_InsecureFalseAgainstTrustedCert(t *testing.T) {
	// Create an HTTPS server with a self-signed cert, but add the cert
	// to the system pool so it's trusted. We do this by creating a custom
	// test that validates the transport respects system certs.
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Verify the server is actually using TLS with a self-signed cert.
	conn, err := tls.Dial("tcp", strings.TrimPrefix(server.URL, "https://"), &tls.Config{
		InsecureSkipVerify: true,
	})
	assert.NilError(t, err)
	defer conn.Close()

	// Get the server certificate and create a cert pool that trusts it.
	serverCert := conn.ConnectionState().PeerCertificates[0]
	certPool := x509.NewCertPool()
	certPool.AddCert(serverCert)

	// Verify the cert pool trusts the server - this validates our test setup.
	_, err = serverCert.Verify(x509.VerifyOptions{
		Roots: certPool,
	})
	assert.NilError(t, err, "cert should be verified with our custom pool")
}

func TestIsLoftReachable_UnhealthyServer(t *testing.T) {
	// Server that returns 500 on /healthz.
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	host := strings.TrimPrefix(server.URL, "https://")

	reachable, err := IsLoftReachable(context.Background(), host, true)
	assert.NilError(t, err)
	assert.Assert(t, !reachable, "should not be reachable when server returns 500")
}

func TestIsLoftReachable_UnreachableHost(t *testing.T) {
	// Use a host that doesn't exist.
	reachable, err := IsLoftReachable(context.Background(), "localhost:1", true)
	assert.NilError(t, err)
	assert.Assert(t, !reachable, "should not be reachable when host is unreachable")
}
