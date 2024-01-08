package cert

import (
	"crypto"
	"crypto/x509"
	"fmt"
	"net"
	"os"

	"github.com/loft-sh/vcluster/pkg/util/certhelper"
	"k8s.io/apimachinery/pkg/util/sets"
)

func GenServingCerts(caCertFile, caKeyFile string, currentCert, currentKey []byte, clusterDomain string, SANs []string) ([]byte, []byte, bool, error) {
	regen := false
	commonName := "kube-apiserver"
	extKeyUsage := []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}

	dnsNames := []string{
		"kubernetes.default.svc." + clusterDomain,
		"kubernetes.default.svc",
		"kubernetes.default",
		"kubernetes",
		"localhost",
	}

	altNames := &certhelper.AltNames{
		DNSNames: dnsNames,
		IPs:      []net.IP{net.ParseIP("127.0.0.1")},
	}

	addSANs(altNames, SANs)
	caBytes, err := os.ReadFile(caCertFile)
	if err != nil {
		return nil, nil, false, err
	}

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caBytes)

	// check for certificate expiration
	if !regen {
		regen = expired(&currentCert, pool)
	}

	if !regen {
		regen = sansChanged(&currentCert, altNames)
	}

	if !regen {
		if len(currentCert) > 0 && len(currentKey) > 0 {
			return currentCert, currentKey, false, nil
		}
	}

	caKeyBytes, err := os.ReadFile(caKeyFile)
	if err != nil {
		return nil, nil, false, err
	}

	caKey, err := certhelper.ParsePrivateKeyPEM(caKeyBytes)
	if err != nil {
		return nil, nil, false, err
	}

	caCert, err := certhelper.ParseCertsPEM(caBytes)
	if err != nil {
		return nil, nil, false, err
	}

	privateKey := currentKey
	if regen || len(currentKey) == 0 {
		privateKey, err = certhelper.MakeEllipticPrivateKeyPEM()
		if err != nil {
			return nil, nil, false, fmt.Errorf("error generating key: %w", err)
		}
	}
	key, err := certhelper.ParsePrivateKeyPEM(privateKey)
	if err != nil {
		return nil, nil, false, err
	}

	cfg := certhelper.Config{
		CommonName: commonName,
		AltNames:   *altNames,
		Usages:     extKeyUsage,
	}
	cert, err := certhelper.NewSignedCert(cfg, key.(crypto.Signer), caCert[0], caKey.(crypto.Signer))
	if err != nil {
		return nil, nil, false, err
	}
	certificate := append(certhelper.EncodeCertPEM(cert), certhelper.EncodeCertPEM(caCert[0])...)
	return certificate, privateKey, true, nil
}

func addSANs(altNames *certhelper.AltNames, sans []string) {
	for _, san := range sans {
		ip := net.ParseIP(san)
		if ip == nil {
			altNames.DNSNames = append(altNames.DNSNames, san)
		} else {
			altNames.IPs = append(altNames.IPs, ip)
		}
	}
}

func expired(certBytes *[]byte, pool *x509.CertPool) bool {
	certificates, err := certhelper.ParseCertsPEM(*certBytes)
	if err != nil {
		return false
	}
	_, err = certificates[0].Verify(x509.VerifyOptions{
		Roots: pool,
		KeyUsages: []x509.ExtKeyUsage{
			x509.ExtKeyUsageAny,
		},
	})
	if err != nil {
		return true
	}
	return certhelper.IsCertExpired(certificates[0])
}

func sansChanged(certBytes *[]byte, sans *certhelper.AltNames) bool {
	if sans == nil {
		return false
	}

	certificates, err := certhelper.ParseCertsPEM(*certBytes)
	if err != nil {
		return false
	}

	if len(certificates) == 0 {
		return false
	}

	if !sets.NewString(certificates[0].DNSNames...).HasAll(sans.DNSNames...) {
		return true
	}

	ips := sets.NewString()
	for _, ip := range certificates[0].IPAddresses {
		ips.Insert(ip.String())
	}

	for _, ip := range sans.IPs {
		if !ips.Has(ip.String()) {
			return true
		}
	}

	return false
}
