package http

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestTransport_Secure(t *testing.T) {
	tr := Transport(false)
	if tr.TLSClientConfig != nil {
		assert.Assert(t, !tr.TLSClientConfig.InsecureSkipVerify, "TLS verification should be enabled when insecure=false")
	}
	assert.Assert(t, !tr.ForceAttemptHTTP2, "HTTP/2 should be disabled")
}

func TestTransport_Insecure(t *testing.T) {
	tr := Transport(true)
	assert.Assert(t, tr.TLSClientConfig != nil, "TLSClientConfig should be set when insecure=true")
	assert.Assert(t, tr.TLSClientConfig.InsecureSkipVerify, "TLS verification should be skipped when insecure=true")
	assert.Assert(t, !tr.ForceAttemptHTTP2, "HTTP/2 should be disabled")
}

func TestInsecureTransport(t *testing.T) {
	tr := InsecureTransport()
	assert.Assert(t, tr.TLSClientConfig != nil, "TLSClientConfig should be set")
	assert.Assert(t, tr.TLSClientConfig.InsecureSkipVerify, "InsecureTransport should skip TLS verification")
}

func TestCloneDefaultTransport(t *testing.T) {
	tr := CloneDefaultTransport()
	assert.Assert(t, !tr.ForceAttemptHTTP2, "HTTP/2 should be disabled")
}
