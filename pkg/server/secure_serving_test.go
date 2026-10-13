package server

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/net/http2"
	"k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/dynamiccertificates"
	certutil "k8s.io/client-go/util/cert"
)

// A client connection that goes silent without closing must not keep its
// streams open on the server.
func TestServeSecureClosesBlackholedConnection(t *testing.T) {
	oldReadIdle, oldPing := http2ReadIdleTimeout, http2PingTimeout
	http2ReadIdleTimeout, http2PingTimeout = time.Second, time.Second
	t.Cleanup(func() { http2ReadIdleTimeout, http2PingTimeout = oldReadIdle, oldPing })

	certPEM, keyPEM, err := certutil.GenerateSelfSignedCertKey("127.0.0.1", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := dynamiccertificates.NewStaticCertKeyContent("test", certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	streamOpen := make(chan struct{})
	streamClosed := make(chan struct{})
	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(streamOpen)
		<-r.Context().Done()
		close(streamClosed)
	})

	stopCh := make(chan struct{})
	stopped, _, err := serveSecure(&server.SecureServingInfo{Listener: ln, Cert: cert}, handler, time.Second, stopCh)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		close(stopCh)
		<-stopped
	})

	relay := newBlackholeRelay(t, ln.Addr().String())

	clientCtx, cancelClient := context.WithCancel(context.Background())
	t.Cleanup(cancelClient)
	go func() {
		client := &http.Client{Transport: &http2.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
		req, _ := http.NewRequestWithContext(clientCtx, http.MethodGet, "https://"+relay.addr+"/", nil)
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
		}
	}()

	select {
	case <-streamOpen:
	case <-time.After(10 * time.Second):
		t.Fatal("stream never reached the handler")
	}

	relay.blackhole.Store(true)

	select {
	case <-streamClosed:
	case <-time.After(10 * time.Second):
		t.Fatal("server kept the stream of a blackholed client connection open")
	}
}

// blackholeRelay forwards one TCP connection. Once blackholed it keeps
// reading both sides but drops everything, so each peer still gets TCP acks
// and only an application-level ping can tell the connection is dead.
type blackholeRelay struct {
	addr      string
	blackhole atomic.Bool
}

func newBlackholeRelay(t *testing.T, target string) *blackholeRelay {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	r := &blackholeRelay{addr: ln.Addr().String()}
	go func() {
		client, err := ln.Accept()
		if err != nil {
			return
		}
		upstream, err := net.Dial("tcp", target)
		if err != nil {
			_ = client.Close()
			return
		}
		t.Cleanup(func() {
			_ = client.Close()
			_ = upstream.Close()
		})
		go r.pipe(upstream, client)
		go r.pipe(client, upstream)
	}()
	return r
}

func (r *blackholeRelay) pipe(dst io.Writer, src io.Reader) {
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if err != nil {
			return
		}
		if r.blackhole.Load() {
			continue
		}
		if _, err := dst.Write(buf[:n]); err != nil {
			return
		}
	}
}
