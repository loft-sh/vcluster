package certs

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/certhelper"
	"gotest.tools/assert"
)

// newSelfSignedCertPEM creates a self-signed certificate PEM that expires at the given time.
func newSelfSignedCertPEM(t *testing.T, cn string, notAfter time.Time) []byte {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NilError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	assert.NilError(t, err)

	cert, err := x509.ParseCertificate(derBytes)
	assert.NilError(t, err)

	return certhelper.EncodeCertPEM(cert)
}

// buildSecretData creates a minimal secret data set.
// Only .crt entries get the provided certPEM; .key and .conf entries get placeholder bytes.
func buildSecretData(certPEM []byte) map[string][]byte {
	data := make(map[string][]byte)
	for _, secretKey := range certMap {
		switch {
		case strings.HasSuffix(secretKey, ".crt"):
			data[secretKey] = certPEM
		case strings.HasSuffix(secretKey, ".key"):
			data[secretKey] = []byte("placeholder-key-data")
		case strings.HasSuffix(secretKey, ".pub"):
			data[secretKey] = []byte("placeholder-pub-data")
		default:
			// .conf files (kubeconfigs)
			data[secretKey] = []byte("placeholder-conf-data")
		}
	}
	return data
}

func TestCertsExpiringSoon(t *testing.T) {
	validPEM := newSelfSignedCertPEM(t, "valid-cert", time.Now().Add(365*24*time.Hour))
	expiringSoonPEM := newSelfSignedCertPEM(t, "expiring-soon", time.Now().Add(60*24*time.Hour))
	alreadyExpiredPEM := newSelfSignedCertPEM(t, "already-expired", time.Now().Add(-24*time.Hour))

	for _, tt := range []struct {
		name     string
		modify   func(map[string][]byte)
		expected bool
	}{
		{
			name:     "all certs valid",
			modify:   nil,
			expected: false,
		},
		{
			name: "leaf cert expiring in 60 days",
			modify: func(data map[string][]byte) {
				data[APIServerCertName] = expiringSoonPEM
			},
			expected: true,
		},
		{
			name: "leaf cert already expired",
			modify: func(data map[string][]byte) {
				data[APIServerCertName] = alreadyExpiredPEM
			},
			expected: true,
		},
		{
			name: "CA cert expiring does not trigger renewal",
			modify: func(data map[string][]byte) {
				data[CACertName] = expiringSoonPEM
			},
			expected: false,
		},
		{
			name: "CA cert already expired does not trigger renewal",
			modify: func(data map[string][]byte) {
				data[CACertName] = alreadyExpiredPEM
				data[ServerCACertName] = alreadyExpiredPEM
				data[ClientCACertName] = alreadyExpiredPEM
				data[FrontProxyCACertName] = alreadyExpiredPEM
				data[strings.ReplaceAll(EtcdCACertName, "/", "-")] = alreadyExpiredPEM
			},
			expected: false,
		},
		{
			name: "leaf cert expiring while CA valid triggers renewal",
			modify: func(data map[string][]byte) {
				data[FrontProxyClientCertName] = expiringSoonPEM
			},
			expected: true,
		},
		{
			name: "missing leaf crt entry in secret data",
			modify: func(data map[string][]byte) {
				delete(data, APIServerCertName)
			},
			expected: true,
		},
		{
			name: "missing CA crt entry does not trigger renewal",
			modify: func(data map[string][]byte) {
				delete(data, FrontProxyCACertName)
			},
			expected: false,
		},
		{
			name: "invalid PEM data in leaf cert",
			modify: func(data map[string][]byte) {
				data[APIServerCertName] = []byte("not-valid-pem")
			},
			expected: true,
		},
		{
			name: "empty PEM data in leaf cert",
			modify: func(data map[string][]byte) {
				data[APIServerCertName] = []byte{}
			},
			expected: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			data := buildSecretData(validPEM)
			if tt.modify != nil {
				tt.modify(data)
			}
			result := certsExpiringSoon(data)
			assert.Equal(t, result, tt.expected)
		})
	}
}

func TestIsCAFile(t *testing.T) {
	caFiles := []string{
		CACertName,
		ServerCACertName,
		ClientCACertName,
		FrontProxyCACertName,
		strings.ReplaceAll(EtcdCACertName, "/", "-"),
	}
	for _, f := range caFiles {
		assert.Assert(t, isCAFile(f), "expected %s to be identified as CA file", f)
	}

	leafFiles := []string{
		APIServerCertName,
		APIServerKubeletClientCertName,
		APIServerEtcdClientCertName,
		FrontProxyClientCertName,
		strings.ReplaceAll(EtcdServerCertName, "/", "-"),
		strings.ReplaceAll(EtcdPeerCertName, "/", "-"),
		strings.ReplaceAll(EtcdHealthcheckClientCertName, "/", "-"),
	}
	for _, f := range leafFiles {
		assert.Assert(t, !isCAFile(f), "expected %s to NOT be identified as CA file", f)
	}
}

func TestWarnIfCAExpiring(t *testing.T) {
	validPEM := newSelfSignedCertPEM(t, "valid-ca", time.Now().Add(365*24*time.Hour))
	expiringSoonPEM := newSelfSignedCertPEM(t, "expiring-ca", time.Now().Add(60*24*time.Hour))

	// Should not panic with all valid certs
	data := buildSecretData(validPEM)
	warnIfCAExpiring(data)

	// Should not panic with expiring CA (just logs a warning)
	data[CACertName] = expiringSoonPEM
	warnIfCAExpiring(data)

	// Should handle empty/missing data gracefully
	data[CACertName] = []byte{}
	warnIfCAExpiring(data)

	delete(data, CACertName)
	warnIfCAExpiring(data)
}
