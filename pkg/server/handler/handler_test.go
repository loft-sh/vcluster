package handler

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"gotest.tools/assert"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/client-go/rest"
)

func TestImpersonatingHandlerUnauthenticatedAndInvalidConfig(t *testing.T) {
	h := ImpersonatingHandler("", &rest.Config{Host: "http://[::1]:badport"})

	// Unauthenticated requests fail with 401 before checking proxy initialization errors.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pods", nil)
	h.ServeHTTP(rec, req)
	assert.Equal(t, rec.Code, http.StatusUnauthorized)

	// Authenticated request surfaces initialization error as 500.
	rec = httptest.NewRecorder()
	req = req.WithContext(request.WithUser(req.Context(), &user.DefaultInfo{Name: "alice"}))
	h.ServeHTTP(rec, req)
	assert.Equal(t, rec.Code, http.StatusInternalServerError)
}

func TestImpersonatingHandlerProxiesMTLSAndUpgrade(t *testing.T) {
	cfg, cleanup := newTestMTLSBackend(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.EqualFold(r.Header.Get("Connection"), "Upgrade") && r.Header.Get("Upgrade") == "websocket" {
			hj, ok := w.(http.Hijacker)
			if !ok {
				http.Error(w, "hijack unsupported", http.StatusInternalServerError)
				return
			}
			conn, rw, err := hj.Hijack()
			if err != nil {
				return
			}
			defer func() { _ = conn.Close() }()
			_, _ = rw.WriteString("HTTP/1.1 101 Switching Protocols\r\n" +
				"Connection: Upgrade\r\n" +
				"Upgrade: websocket\r\n" +
				"X-Impersonated-User: " + r.Header.Get("Impersonate-User") + "\r\n\r\n")
			_ = rw.Flush()
			return
		}
		w.Header().Set("X-Path", r.URL.Path)
		w.Header().Set("X-Impersonate-User", r.Header.Get("Impersonate-User"))
		w.Header().Set("X-Impersonate-Group", strings.Join(r.Header.Values("Impersonate-Group"), ","))
		w.Header().Set("X-Impersonate-Extra", strings.Join(r.Header.Values("Impersonate-Extra-Scope"), ","))
		w.WriteHeader(http.StatusOK)
	})
	defer cleanup()

	h := ImpersonatingHandler("/proxy", cfg)

	// Verify concurrent non-upgrade requests carry distinct per-request impersonation headers and prefix stripping.
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			userName := "user-a"
			group := "team-a"
			if idx%2 == 1 {
				userName = "user-b"
				group = "team-b"
			}
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/proxy/apis/v1", nil)
			req = req.WithContext(request.WithUser(req.Context(), &user.DefaultInfo{
				Name:   userName,
				Groups: []string{group},
				Extra:  map[string][]string{"scope": {"read"}},
			}))
			h.ServeHTTP(rec, req)
			assert.Equal(t, rec.Code, http.StatusOK)
			assert.Equal(t, rec.Header().Get("X-Path"), "/apis/v1")
			assert.Equal(t, rec.Header().Get("X-Impersonate-User"), userName)
			assert.Equal(t, rec.Header().Get("X-Impersonate-Group"), group)
			assert.Equal(t, rec.Header().Get("X-Impersonate-Extra"), "read")
		}(i)
	}
	wg.Wait()

	// Verify upgrade requests lazily construct UpgradeTransport with impersonation headers over mTLS.
	upgradeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(request.WithUser(r.Context(), &user.DefaultInfo{Name: "upgrade-user"}))
		h.ServeHTTP(w, r)
	}))
	defer upgradeSrv.Close()

	uConn, err := net.Dial("tcp", upgradeSrv.Listener.Addr().String())
	assert.NilError(t, err)
	defer func() { _ = uConn.Close() }()
	rawReq := "GET /proxy/api/v1/pods/p/exec HTTP/1.1\r\n" +
		"Host: " + upgradeSrv.Listener.Addr().String() + "\r\n" +
		"Connection: Upgrade\r\n" +
		"Upgrade: websocket\r\n\r\n"
	_, err = io.WriteString(uConn, rawReq)
	assert.NilError(t, err)

	resp, err := http.ReadResponse(bufio.NewReader(uConn), &http.Request{Method: http.MethodGet})
	assert.NilError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, resp.StatusCode, http.StatusSwitchingProtocols)
	assert.Equal(t, resp.Header.Get("X-Impersonated-User"), "upgrade-user")
}

func BenchmarkImpersonatingHandler(b *testing.B) {
	cfg, cleanup := newTestMTLSBackend(b, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer cleanup()

	h := ImpersonatingHandler("", cfg)
	ctx := request.WithUser(b.Context(), &user.DefaultInfo{
		Name:   "bench-user",
		Groups: []string{"system:authenticated"},
	})

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/livez", nil).WithContext(ctx)
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", rec.Code)
		}
	}
}

func newTestMTLSBackend(tb testing.TB, handler http.HandlerFunc) (*rest.Config, func()) {
	tb.Helper()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		tb.Fatal(err)
	}
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		tb.Fatal(err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		tb.Fatal(err)
	}
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})

	signLeaf := func(serial int64, cn string, ips []net.IP, eku []x509.ExtKeyUsage) ([]byte, []byte) {
		key, keyErr := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if keyErr != nil {
			tb.Fatal(keyErr)
		}
		tmpl := &x509.Certificate{
			SerialNumber: big.NewInt(serial),
			Subject:      pkix.Name{CommonName: cn},
			NotBefore:    time.Now().Add(-time.Hour),
			NotAfter:     time.Now().Add(time.Hour),
			KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
			ExtKeyUsage:  eku,
			IPAddresses:  ips,
		}
		der, certErr := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caKey)
		if certErr != nil {
			tb.Fatal(certErr)
		}
		keyDER, marshalErr := x509.MarshalECPrivateKey(key)
		if marshalErr != nil {
			tb.Fatal(marshalErr)
		}
		return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
			pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	}

	srvCertPEM, srvKeyPEM := signLeaf(2, "server", []net.IP{net.ParseIP("127.0.0.1")}, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth})
	cliCertPEM, cliKeyPEM := signLeaf(3, "client", nil, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth})

	srvCert, err := tls.X509KeyPair(srvCertPEM, srvKeyPEM)
	if err != nil {
		tb.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(caCert)

	srv := httptest.NewUnstartedServer(handler)
	srv.TLS = &tls.Config{
		Certificates: []tls.Certificate{srvCert},
		ClientCAs:    pool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}
	srv.StartTLS()

	cfg := &rest.Config{
		Host: srv.URL,
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   caPEM,
			CertData: cliCertPEM,
			KeyData:  cliKeyPEM,
		},
	}
	return cfg, srv.Close
}
