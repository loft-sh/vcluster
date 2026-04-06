package start

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	netUrl "net/url"
	"strings"
	"testing"

	types "github.com/loft-sh/api/v4/pkg/auth"
	"gotest.tools/v3/assert"
)

func TestIsTLSError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name: "URL error with certificate verification error",
			err: &netUrl.Error{
				Op:  "Get",
				URL: "https://localhost:9898",
				Err: &tls.CertificateVerificationError{
					UnverifiedCertificates: []*x509.Certificate{},
					Err:                    x509.UnknownAuthorityError{},
				},
			},
			expected: true,
		},
		{
			name: "URL error with unknown authority",
			err: &netUrl.Error{
				Op:  "Get",
				URL: "https://localhost:9898",
				Err: x509.UnknownAuthorityError{},
			},
			expected: true,
		},
		{
			name: "URL error with non-TLS error",
			err: &netUrl.Error{
				Op:  "Get",
				URL: "https://localhost:9898",
				Err: &netUrl.Error{Op: "dial", URL: "localhost", Err: nil},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				assert.Assert(t, !isTLSError(nil))
				return
			}
			assert.Equal(t, isTLSError(tt.err), tt.expected)
		})
	}
}

func TestPasswordLogin_SecureSuccess(t *testing.T) {
	expectedKey := "test-access-key-123"
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/password/login" {
			resp := types.AccessKey{AccessKey: expectedKey}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// passwordLogin with insecure=true should succeed against self-signed cert.
	l := &LoftStarter{}
	key, insecure, err := l.passwordLogin(server.URL, []byte(`{}`), true)
	assert.NilError(t, err)
	assert.Equal(t, key, expectedKey)
	assert.Assert(t, insecure)
}

func TestPasswordLogin_InsecureFalseFailsAgainstSelfSigned(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := types.AccessKey{AccessKey: "key"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// passwordLogin with insecure=false should fail against self-signed cert.
	l := &LoftStarter{}
	_, _, err := l.passwordLogin(server.URL, []byte(`{}`), false)
	assert.Assert(t, err != nil, "expected TLS error")
	assert.Assert(t, isTLSError(err), "error should be a TLS error, got: %v", err)
}

func TestPasswordLogin_FallbackBehavior(t *testing.T) {
	expectedKey := "fallback-key"
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/password/login" {
			resp := types.AccessKey{AccessKey: expectedKey}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	l := &LoftStarter{}

	// Simulate the try-secure-then-fallback pattern from loginViaCLI.
	key, insecure, err := l.passwordLogin(server.URL, []byte(`{}`), false)
	if err != nil && isTLSError(err) {
		// Fall back to insecure — this is the expected path for self-signed certs.
		key, insecure, err = l.passwordLogin(server.URL, []byte(`{}`), true)
	}
	assert.NilError(t, err)
	assert.Equal(t, key, expectedKey)
	assert.Assert(t, insecure, "should have fallen back to insecure")
}

func TestPasswordLogin_EmptyAccessKey(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := types.AccessKey{AccessKey: ""}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	l := &LoftStarter{}
	_, _, err := l.passwordLogin(server.URL, []byte(`{}`), true)
	assert.Assert(t, err != nil, "expected error for empty access key")
	assert.Assert(t, strings.Contains(err.Error(), "couldn't retrieve access key"))
}
