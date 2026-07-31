// Package webhook contains admission webhook tests.
package webhook

import (
	"crypto"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math"
	"math/big"
	"os"
	"time"

	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
)

const (
	certificateBlockType = "CERTIFICATE"
	rsaKeySize           = 2048
	duration365d         = time.Hour * 24 * 365
)

type certContext struct {
	cert        []byte
	key         []byte
	signingCert []byte
}

func newPrivateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(cryptorand.Reader, rsaKeySize)
}

func encodeCertPEM(cert *x509.Certificate) []byte {
	block := pem.Block{
		Type:  certificateBlockType,
		Bytes: cert.Raw,
	}
	return pem.EncodeToMemory(&block)
}

func newSignedCert(cfg *certutil.Config, key crypto.Signer, caCert *x509.Certificate, caKey crypto.Signer) (*x509.Certificate, error) {
	serial, err := cryptorand.Int(cryptorand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	if len(cfg.CommonName) == 0 {
		return nil, errors.New("must specify a CommonName")
	}
	if len(cfg.Usages) == 0 {
		return nil, errors.New("must specify at least one ExtKeyUsage")
	}

	certTmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName:   cfg.CommonName,
			Organization: cfg.Organization,
		},
		DNSNames:     cfg.AltNames.DNSNames,
		IPAddresses:  cfg.AltNames.IPs,
		SerialNumber: serial,
		NotBefore:    caCert.NotBefore,
		NotAfter:     time.Now().Add(duration365d).UTC(),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  cfg.Usages,
	}
	certDERBytes, err := x509.CreateCertificate(cryptorand.Reader, &certTmpl, caCert, key.Public(), caKey)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}

func setupServerCert(namespaceName, svcName string) *certContext {
	certDir, err := os.MkdirTemp("", "test-e2e-server-cert")
	if err != nil {
		panic("failed to create temp dir for cert generation: " + err.Error())
	}
	defer os.RemoveAll(certDir)

	signingKey, err := newPrivateKey()
	if err != nil {
		panic("failed to create CA private key: " + err.Error())
	}
	signingCert, err := certutil.NewSelfSignedCACert(certutil.Config{CommonName: "e2e-server-cert-ca"}, signingKey)
	if err != nil {
		panic("failed to create CA cert: " + err.Error())
	}

	key, err := newPrivateKey()
	if err != nil {
		panic("failed to create private key: " + err.Error())
	}
	signedCert, err := newSignedCert(
		&certutil.Config{
			CommonName: svcName + "." + namespaceName + ".svc",
			AltNames:   certutil.AltNames{DNSNames: []string{svcName + "." + namespaceName + ".svc"}},
			Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		},
		key, signingCert, signingKey,
	)
	if err != nil {
		panic("failed to create cert: " + err.Error())
	}

	privateKeyPEM, err := keyutil.MarshalPrivateKeyToPEM(key)
	if err != nil {
		panic("failed to marshal key: " + err.Error())
	}

	return &certContext{
		cert:        encodeCertPEM(signedCert),
		key:         privateKeyPEM,
		signingCert: encodeCertPEM(signingCert),
	}
}
