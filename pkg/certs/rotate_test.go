package certs

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/certhelper"
	"gotest.tools/assert"
)

// newCACertPair returns a self-signed root CA and an intermediate CA signed by
// that root, both PEM encoded.
func newCACertPair(t *testing.T) (rootPEM, intermediatePEM []byte) {
	t.Helper()

	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NilError(t, err)
	rootTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "internal-root-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	rootDER, err := x509.CreateCertificate(rand.Reader, rootTmpl, rootTmpl, &rootKey.PublicKey, rootKey)
	assert.NilError(t, err)
	rootCert, err := x509.ParseCertificate(rootDER)
	assert.NilError(t, err)

	intermediateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NilError(t, err)
	intermediateTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(2),
		Subject:               pkix.Name{CommonName: "vcluster-intermediate-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	intermediateDER, err := x509.CreateCertificate(rand.Reader, intermediateTmpl, rootCert, &intermediateKey.PublicKey, rootKey)
	assert.NilError(t, err)
	intermediateCert, err := x509.ParseCertificate(intermediateDER)
	assert.NilError(t, err)

	return certhelper.EncodeCertPEM(rootCert), certhelper.EncodeCertPEM(intermediateCert)
}

func writeCACert(t *testing.T, dir string, pemBytes []byte) {
	t.Helper()
	assert.NilError(t, os.WriteFile(filepath.Join(dir, CACertName), pemBytes, 0600))
}

func TestEnsureCARotationAllowed(t *testing.T) {
	rootPEM, intermediatePEM := newCACertPair(t)

	t.Run("missing CA cert is allowed", func(t *testing.T) {
		dir := t.TempDir()
		assert.NilError(t, ensureCARotationAllowed(dir))
	})

	t.Run("self-signed CA is allowed", func(t *testing.T) {
		dir := t.TempDir()
		writeCACert(t, dir, rootPEM)
		assert.NilError(t, ensureCARotationAllowed(dir))
	})

	t.Run("externally issued CA is refused", func(t *testing.T) {
		dir := t.TempDir()
		writeCACert(t, dir, intermediatePEM)
		err := ensureCARotationAllowed(dir)
		assert.ErrorContains(t, err, "refusing to rotate the CA")
		assert.ErrorContains(t, err, "--force")
	})

	t.Run("bundle with externally issued CA first is refused", func(t *testing.T) {
		dir := t.TempDir()
		writeCACert(t, dir, append(append([]byte{}, intermediatePEM...), rootPEM...))
		err := ensureCARotationAllowed(dir)
		assert.ErrorContains(t, err, "refusing to rotate the CA")
	})

	t.Run("bundle with self-signed CA first is allowed", func(t *testing.T) {
		dir := t.TempDir()
		writeCACert(t, dir, append(append([]byte{}, rootPEM...), intermediatePEM...))
		assert.NilError(t, ensureCARotationAllowed(dir))
	})

	t.Run("garbage CA cert is refused", func(t *testing.T) {
		dir := t.TempDir()
		writeCACert(t, dir, []byte("not a certificate"))
		err := ensureCARotationAllowed(dir)
		assert.ErrorContains(t, err, "parsing CA certificate")
	})
}

func TestIsSelfIssued(t *testing.T) {
	rootPEM, intermediatePEM := newCACertPair(t)

	rootCerts, err := certhelper.ParseCertsPEM(rootPEM)
	assert.NilError(t, err)
	assert.Assert(t, isSelfIssued(rootCerts[0]))

	intermediateCerts, err := certhelper.ParseCertsPEM(intermediatePEM)
	assert.NilError(t, err)
	assert.Assert(t, !isSelfIssued(intermediateCerts[0]))
}
