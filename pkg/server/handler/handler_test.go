package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gotest.tools/assert"
	"k8s.io/client-go/rest"
)

// TestHandlerWithErrorResponder_UseLocationHost verifies that the backend receives the correct
// Host header, not the client's vcluster LB hostname, for plain HTTP requests (kubectl logs).
func TestHandlerWithErrorResponder_UseLocationHost(t *testing.T) {
	var receivedHost string
	backend := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHost = r.Host
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	cfg := &rest.Config{
		Host:            backend.URL,
		TLSClientConfig: rest.TLSClientConfig{Insecure: true},
	}

	h, err := HandlerWithErrorResponder("", cfg, nil, nil)
	assert.NilError(t, err)

	// Simulate a client request with a vcluster LB hostname as Host —
	// this is what the UpgradeAwareHandler used to forward before the fix.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/namespaces/default/pods/test/log", nil)
	req.Host = "vcluster.example.com"

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, rec.Code, http.StatusOK, "proxy round-trip must succeed")
	// The backend must receive its own hostname, not the client's LB address.
	assert.Equal(t, receivedHost, backend.Listener.Addr().String(),
		"UpgradeAwareHandler must send Host: <backend hostname> (UseLocationHost=true), not the original client Host header")
}

// TestHandlerWithErrorResponder_UseLocationHost_Upgrade verifies that the correct Host header
// is forwarded on SPDY upgrade requests (kubectl exec, attach, port-forward).
// httptest.ResponseRecorder cannot be hijacked, so we drive the handler through a real
// httptest.Server so the connection supports hijacking.
// No auth/impersonation is configured: those layers set HTTP headers on the outbound transport,
// not req.Host, so they do not affect whether UseLocationHost works correctly.
func TestHandlerWithErrorResponder_UseLocationHost_Upgrade(t *testing.T) {
	hostCh := make(chan string, 1)
	backend := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hostCh <- r.Host
		w.Header().Set("Connection", "Upgrade")
		w.Header().Set("Upgrade", "SPDY/3.1")
		w.WriteHeader(http.StatusSwitchingProtocols)
	}))
	defer backend.Close()

	cfg := &rest.Config{
		Host:            backend.URL,
		TLSClientConfig: rest.TLSClientConfig{Insecure: true},
	}

	h, err := HandlerWithErrorResponder("", cfg, nil, nil)
	assert.NilError(t, err)

	// Wrap the handler in a real server so the response writer supports hijacking.
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Host = "vcluster.example.com"
		h.ServeHTTP(w, r)
	}))
	defer proxy.Close()

	req, err := http.NewRequest(http.MethodPost, proxy.URL+"/api/v1/namespaces/default/pods/test/exec", nil)
	assert.NilError(t, err)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "SPDY/3.1")

	//nolint:bodyclose
	http.DefaultClient.Do(req) //nolint:errcheck // 101 is not an error per net/http

	// The backend must receive its own hostname, not the client's LB address.
	receivedHost := <-hostCh
	assert.Equal(t, receivedHost, backend.Listener.Addr().String(),
		"UpgradeAwareHandler must send Host: <backend hostname> on upgrade requests (UseLocationHost=true)")
}
