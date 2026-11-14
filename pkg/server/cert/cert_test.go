package cert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/certhelper"
	"gotest.tools/v3/assert"
)

func newTestCACert(t *testing.T) (*x509.Certificate, *ecdsa.PrivateKey) {
	t.Helper()
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NilError(t, err)

	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	assert.NilError(t, err)
	ca, err := x509.ParseCertificate(caDER)
	assert.NilError(t, err)
	return ca, caKey
}

func newTestLeafCert(t *testing.T, ca *x509.Certificate, caKey *ecdsa.PrivateKey, notAfter time.Time, dnsNames []string, ips []net.IP) ([]byte, []byte) {
	t.Helper()
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NilError(t, err)

	leafTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "test-leaf"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     dnsNames,
		IPAddresses:  ips,
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTmpl, ca, &leafKey.PublicKey, caKey)
	assert.NilError(t, err)
	leaf, err := x509.ParseCertificate(leafDER)
	assert.NilError(t, err)

	certPEM := certhelper.EncodeCertPEM(leaf)
	keyPEM, err := certhelper.MakeEllipticPrivateKeyPEM()
	assert.NilError(t, err)
	return certPEM, keyPEM
}

func TestExpired_TrueWithin90Days(t *testing.T) {
	ca, caKey := newTestCACert(t)
	certPEM, _ := newTestLeafCert(t, ca, caKey,
		time.Now().Add(60*24*time.Hour), // 60 days
		[]string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1")})

	pool := x509.NewCertPool()
	pool.AddCert(ca)

	assert.Assert(t, expired(&certPEM, pool), "cert expiring in 60 days should be flagged")
}

func TestExpired_FalseBeyond90Days(t *testing.T) {
	ca, caKey := newTestCACert(t)
	certPEM, _ := newTestLeafCert(t, ca, caKey,
		time.Now().Add(200*24*time.Hour), // 200 days
		[]string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1")})

	pool := x509.NewCertPool()
	pool.AddCert(ca)

	assert.Assert(t, !expired(&certPEM, pool), "cert expiring in 200 days should not be flagged")
}

func TestSansChanged_DetectsNewDNS(t *testing.T) {
	ca, caKey := newTestCACert(t)
	certPEM, _ := newTestLeafCert(t, ca, caKey,
		time.Now().Add(365*24*time.Hour),
		[]string{"a"}, nil)

	sans := &certhelper.AltNames{
		DNSNames: []string{"a", "b"},
	}
	assert.Assert(t, sansChanged(&certPEM, sans), "should detect new DNS SAN 'b'")
}

func TestSansChanged_DetectsNewIP(t *testing.T) {
	ca, caKey := newTestCACert(t)
	certPEM, _ := newTestLeafCert(t, ca, caKey,
		time.Now().Add(365*24*time.Hour),
		nil, []net.IP{net.ParseIP("127.0.0.1")})

	sans := &certhelper.AltNames{
		IPs: []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("10.0.0.1")},
	}
	assert.Assert(t, sansChanged(&certPEM, sans), "should detect new IP SAN 10.0.0.1")
}

func TestSansChanged_FalseWhenMatching(t *testing.T) {
	ca, caKey := newTestCACert(t)
	dnsNames := []string{"a", "b"}
	ips := []net.IP{net.ParseIP("127.0.0.1")}
	certPEM, _ := newTestLeafCert(t, ca, caKey,
		time.Now().Add(365*24*time.Hour),
		dnsNames, ips)

	sans := &certhelper.AltNames{
		DNSNames: dnsNames,
		IPs:      ips,
	}
	assert.Assert(t, !sansChanged(&certPEM, sans), "should not detect change when SANs match")
}
