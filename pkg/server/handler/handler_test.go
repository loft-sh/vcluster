package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gotest.tools/assert"
	"k8s.io/client-go/rest"
)

// fakeResponder satisfies the ErrorResponder interface for tests.
type fakeResponder struct{}

func (f *fakeResponder) Error(w http.ResponseWriter, _ *http.Request, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

// TestHandlerWithErrorResponder_UseLocationHost verifies that the backend receives the correct Host header, not the client's vcluster LB hostname.
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

	h, err := HandlerWithErrorResponder("", cfg, nil, &fakeResponder{})
	assert.NilError(t, err)

	// Simulate a client request with a vcluster LB hostname as Host —
	// this is what the UpgradeAwareHandler used to forward before the fix.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/namespaces/default/pods/test/log", nil)
	req.Host = "vcluster.example.com"

	h.ServeHTTP(httptest.NewRecorder(), req)

	// The backend must receive its own hostname, not the client's LB address.
	assert.Equal(t, receivedHost, backend.Listener.Addr().String(),
		"UpgradeAwareHandler must send Host: <backend hostname> (UseLocationHost=true), not the original client Host header")
}
