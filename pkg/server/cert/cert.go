package cert

import (
	"crypto"
	"crypto/x509"
	"github.com/loft-sh/vcluster/pkg/util/certhelper"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/sets"
	"net"
	"os"
)

func GenServingCerts(caCertFile, caKeyFile, certFile, keyFile, clusterDomain string, SANs []string) (bool, error) {
	regen := false
	commonName := "kube-apiserver"
	extKeyUsage := []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	altNames := &certhelper.AltNames{
		DNSNames: []string{"kubernetes.default.svc." + clusterDomain, "kubernetes.default.svc", "kubernetes.default", "kubernetes", "localhost"},
		IPs:      []net.IP{net.ParseIP("127.0.0.1")},
	}

	addSANs(altNames, SANs)
	caBytes, err := ioutil.ReadFile(caCertFile)
	if err != nil {
		return false, err
	}

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caBytes)

	// check for certificate expiration
	if !regen {
		regen = expired(certFile, pool)
	}

	if !regen {
		regen = sansChanged(certFile, altNames)
	}

	if !regen {
		if exists(certFile, keyFile) {
			return false, nil
		}
	}

	caKeyBytes, err := ioutil.ReadFile(caKeyFile)
	if err != nil {
		return false, err
	}

	caKey, err := certhelper.ParsePrivateKeyPEM(caKeyBytes)
	if err != nil {
		return false, err
	}

	caCert, err := certhelper.ParseCertsPEM(caBytes)
	if err != nil {
		return false, err
	}

	keyBytes, _, err := certhelper.LoadOrGenerateKeyFile(keyFile, regen)
	if err != nil {
		return false, err
	}

	key, err := certhelper.ParsePrivateKeyPEM(keyBytes)
	if err != nil {
		return false, err
	}

	cfg := certhelper.Config{
		CommonName: commonName,
		AltNames:   *altNames,
		Usages:     extKeyUsage,
	}
	cert, err := certhelper.NewSignedCert(cfg, key.(crypto.Signer), caCert[0], caKey.(crypto.Signer))
	if err != nil {
		return false, err
	}

	return true, certhelper.WriteCert(certFile, append(certhelper.EncodeCertPEM(cert), certhelper.EncodeCertPEM(caCert[0])...))
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

func expired(certFile string, pool *x509.CertPool) bool {
	certBytes, err := ioutil.ReadFile(certFile)
	if err != nil {
		return false
	}
	certificates, err := certhelper.ParseCertsPEM(certBytes)
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

func exists(files ...string) bool {
	for _, file := range files {
		if _, err := os.Stat(file); err != nil {
			return false
		}
	}
	return true
}

func sansChanged(certFile string, sans *certhelper.AltNames) bool {
	if sans == nil {
		return false
	}

	certBytes, err := ioutil.ReadFile(certFile)
	if err != nil {
		return false
	}

	certificates, err := certhelper.ParseCertsPEM(certBytes)
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
