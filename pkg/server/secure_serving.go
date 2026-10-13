/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"k8s.io/apiserver/pkg/endpoints/metrics"
	"k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/dynamiccertificates"
	"k8s.io/component-base/cli/flag"
	"k8s.io/klog/v2"
)

// Copied from k8s.io/apiserver/pkg/server/secure_serving.go so the HTTP/2
// server can ping its clients. Without pings, a client connection that dies
// without a FIN or RST (a blackholed konnectivity agent) stays open until
// kernel TCP retransmissions give up, about 15 minutes, and every stream
// proxied over it stays open with it.
var (
	http2ReadIdleTimeout = 30 * time.Second
	http2PingTimeout     = 15 * time.Second
)

// serveSecure is server.SecureServingInfo.Serve with HTTP/2 pings enabled.
func serveSecure(s *server.SecureServingInfo, handler http.Handler, shutdownTimeout time.Duration, stopCh <-chan struct{}) (<-chan struct{}, <-chan struct{}, error) {
	if s.Listener == nil {
		return nil, nil, fmt.Errorf("listener must not be nil")
	}

	tlsConfig, err := secureTLSConfig(s, stopCh)
	if err != nil {
		return nil, nil, err
	}

	secureServer, err := newSecureServer(s, handler, tlsConfig)
	if err != nil {
		return nil, nil, err
	}

	klog.Infof("Serving securely on %s", secureServer.Addr)
	return server.RunServer(secureServer, s.Listener, shutdownTimeout, stopCh)
}

func newSecureServer(s *server.SecureServingInfo, handler http.Handler, tlsConfig *tls.Config) (*http.Server, error) {
	secureServer := &http.Server{
		Addr:           s.Listener.Addr().String(),
		Handler:        handler,
		MaxHeaderBytes: 1 << 20,
		TLSConfig:      tlsConfig,

		IdleTimeout:       90 * time.Second, // matches http.DefaultTransport keep-alive timeout
		ReadHeaderTimeout: 32 * time.Second, // just shy of requestTimeoutUpperBound
	}

	if !s.DisableHTTP2 {
		// At least 99% of serialized resources in surveyed clusters were smaller than 256kb.
		// This should be big enough to accommodate most API POST requests in a single frame,
		// and small enough to allow a per connection buffer of this size multiplied by `MaxConcurrentStreams`.
		const resourceBody99Percentile = 256 * 1024

		// Upstream configures x/net/http2.Server; these are the same settings on
		// the net/http HTTP/2 server. IdleTimeout comes from secureServer.
		http2Config := &http.HTTP2Config{
			// shrink the per-stream buffer and max framesize from the 1MB default while still accommodating most API POST requests in a single frame
			MaxReceiveBufferPerStream: resourceBody99Percentile,
			MaxReadFrameSize:          resourceBody99Percentile,

			SendPingTimeout: http2ReadIdleTimeout,
			PingTimeout:     http2PingTimeout,
		}

		// use the overridden concurrent streams setting or make the default of 250 explicit so we can size MaxReceiveBufferPerConnection appropriately
		if s.HTTP2MaxStreamsPerConnection > 0 {
			http2Config.MaxConcurrentStreams = s.HTTP2MaxStreamsPerConnection
		} else {
			// match http2.initialMaxConcurrentStreams used by clients
			// this makes it so that a malicious client can only open 400 streams before we forcibly close the connection
			// https://github.com/golang/net/commit/b225e7ca6dde1ef5a5ae5ce922861bda011cfabd
			http2Config.MaxConcurrentStreams = 100
		}

		// increase the connection buffer size from the 1MB default to handle the specified number of concurrent streams
		http2Config.MaxReceiveBufferPerConnection = http2Config.MaxReceiveBufferPerStream * http2Config.MaxConcurrentStreams
		secureServer.HTTP2 = http2Config
	}

	// use tlsHandshakeErrorWriter to handle messages of tls handshake error
	tlsErrorWriter := &tlsHandshakeErrorWriter{os.Stderr}
	tlsErrorLogger := log.New(tlsErrorWriter, "", 0)
	secureServer.ErrorLog = tlsErrorLogger

	return secureServer, nil
}

// secureTLSConfig is server.SecureServingInfo.tlsConfig.
func secureTLSConfig(s *server.SecureServingInfo, stopCh <-chan struct{}) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		// Can't use SSLv3 because of POODLE and BEAST
		// Can't use TLSv1.0 because of POODLE and BEAST using CBC cipher
		// Can't use TLSv1.1 because of RC4 cipher usage
		MinVersion: tls.VersionTLS12,
		// enable HTTP2 for go's 1.7 HTTP Server
		NextProtos: []string{"h2", "http/1.1"},
	}

	// these are static aspects of the tls.Config
	if s.DisableHTTP2 {
		klog.Info("Forcing use of http/1.1 only")
		tlsConfig.NextProtos = []string{"http/1.1"}
	}
	if s.MinTLSVersion > 0 {
		tlsConfig.MinVersion = s.MinTLSVersion
	}
	if len(s.CipherSuites) > 0 {
		tlsConfig.CipherSuites = s.CipherSuites
		insecureCiphers := flag.InsecureTLSCiphers()
		for i := 0; i < len(s.CipherSuites); i++ {
			for cipherName, cipherID := range insecureCiphers {
				if s.CipherSuites[i] == cipherID {
					klog.Warningf("Use of insecure cipher '%s' detected.", cipherName)
				}
			}
		}
	}
	if len(s.CurvePreferences) > 0 {
		tlsConfig.CurvePreferences = s.CurvePreferences
	}

	if s.ClientCA != nil {
		// Populate PeerCertificates in requests, but don't reject connections without certificates
		// This allows certificates to be validated by authenticators, while still allowing other auth types
		tlsConfig.ClientAuth = tls.RequestClientCert
	}

	if s.ClientCA != nil || s.Cert != nil || len(s.SNICerts) > 0 {
		dynamicCertificateController := dynamiccertificates.NewDynamicServingCertificateController(
			tlsConfig,
			s.ClientCA,
			s.Cert,
			s.SNICerts,
			nil, // TODO see how to plumb an event recorder down in here. For now this results in simply klog messages.
		)

		if s.ClientCA != nil {
			s.ClientCA.AddListener(dynamicCertificateController)
		}
		if s.Cert != nil {
			s.Cert.AddListener(dynamicCertificateController)
		}
		// generate a context from stopCh. This is to avoid modifying files which are relying on apiserver
		// TODO: See if we can pass ctx to the current method
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			select {
			case <-stopCh:
				cancel() // stopCh closed, so cancel our context
			case <-ctx.Done():
			}
		}()
		// start controllers if possible
		if controller, ok := s.ClientCA.(dynamiccertificates.ControllerRunner); ok {
			// runonce to try to prime data.  If this fails, it's ok because we fail closed.
			// Files are required to be populated already, so this is for convenience.
			if err := controller.RunOnce(ctx); err != nil {
				klog.Warningf("Initial population of client CA failed: %v", err)
			}

			go controller.Run(ctx, 1)
		}
		if controller, ok := s.Cert.(dynamiccertificates.ControllerRunner); ok {
			// runonce to try to prime data.  If this fails, it's ok because we fail closed.
			// Files are required to be populated already, so this is for convenience.
			if err := controller.RunOnce(ctx); err != nil {
				klog.Warningf("Initial population of default serving certificate failed: %v", err)
			}

			go controller.Run(ctx, 1)
		}
		for _, sniCert := range s.SNICerts {
			sniCert.AddListener(dynamicCertificateController)
			if controller, ok := sniCert.(dynamiccertificates.ControllerRunner); ok {
				// runonce to try to prime data.  If this fails, it's ok because we fail closed.
				// Files are required to be populated already, so this is for convenience.
				if err := controller.RunOnce(ctx); err != nil {
					klog.Warningf("Initial population of SNI serving certificate failed: %v", err)
				}

				go controller.Run(ctx, 1)
			}
		}

		// runonce to try to prime data.  If this fails, it's ok because we fail closed.
		// Files are required to be populated already, so this is for convenience.
		if err := dynamicCertificateController.RunOnce(); err != nil {
			klog.Warningf("Initial population of dynamic certificates failed: %v", err)
		}
		go dynamicCertificateController.Run(1, stopCh)

		tlsConfig.GetConfigForClient = dynamicCertificateController.GetConfigForClient
	}

	return tlsConfig, nil
}

// tlsHandshakeErrorWriter writes TLS handshake errors to klog with
// trace level - V(5), to avoid flooding of tls handshake errors.
type tlsHandshakeErrorWriter struct {
	out io.Writer
}

const tlsHandshakeErrorPrefix = "http: TLS handshake error"

func (w *tlsHandshakeErrorWriter) Write(p []byte) (int, error) {
	if strings.Contains(string(p), tlsHandshakeErrorPrefix) {
		klog.V(5).Info(string(p))
		metrics.TLSHandshakeErrors.Inc()
		return len(p), nil
	}

	// for non tls handshake error, log it as usual
	return w.out.Write(p)
}
