package certs

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/certhelper"
	"gotest.tools/assert"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	certutil "k8s.io/client-go/util/cert"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmcerts "k8s.io/kubernetes/cmd/kubeadm/app/phases/certs"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pkiutil"
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

func TestSplitCACertPreservesExistingAliases(t *testing.T) {
	dir := t.TempDir()
	caPEM := newSelfSignedCertPEM(t, "ca", time.Now().Add(365*24*time.Hour))
	serverPEM := newSelfSignedCertPEM(t, "server-ca", time.Now().Add(365*24*time.Hour))
	clientPEM := newSelfSignedCertPEM(t, "client-ca", time.Now().Add(365*24*time.Hour))

	writeTestCertFile(t, dir, CACertName, caPEM)
	writeTestCertFile(t, dir, CAKeyName, []byte("ca-key"))
	writeTestCertFile(t, dir, ServerCACertName, serverPEM)
	writeTestCertFile(t, dir, ServerCAKeyName, []byte("server-key"))
	writeTestCertFile(t, dir, ClientCACertName, clientPEM)
	writeTestCertFile(t, dir, ClientCAKeyName, []byte("client-key"))

	err := splitCACert(dir)
	assert.NilError(t, err)

	assert.Equal(t, string(readTestCertFile(t, dir, ServerCACertName)), string(serverPEM))
	assert.Equal(t, string(readTestCertFile(t, dir, ServerCAKeyName)), "server-key")
	assert.Equal(t, string(readTestCertFile(t, dir, ClientCACertName)), string(clientPEM))
	assert.Equal(t, string(readTestCertFile(t, dir, ClientCAKeyName)), "client-key")
}

func TestSplitCACertCreatesMissingAliases(t *testing.T) {
	dir := t.TempDir()
	caPEM := newSelfSignedCertPEM(t, "ca", time.Now().Add(365*24*time.Hour))

	writeTestCertFile(t, dir, CACertName, caPEM)
	writeTestCertFile(t, dir, CAKeyName, []byte("ca-key"))

	err := splitCACert(dir)
	assert.NilError(t, err)

	assert.Equal(t, string(readTestCertFile(t, dir, ServerCACertName)), string(caPEM))
	assert.Equal(t, string(readTestCertFile(t, dir, ServerCAKeyName)), "ca-key")
	assert.Equal(t, string(readTestCertFile(t, dir, ClientCACertName)), string(caPEM))
	assert.Equal(t, string(readTestCertFile(t, dir, ClientCAKeyName)), "ca-key")
}

func TestEnsureAPIServerServingCertSignedByServerCA(t *testing.T) {
	dir := t.TempDir()
	caCert, caKey := writeTestCA(t, dir, CACertAndKeyBaseName, "kubernetes")
	serverCACert, _ := writeTestCA(t, dir, strings.TrimSuffix(ServerCACertName, ".crt"), "k3s-server-ca")
	kubeadmConfig := testKubeadmConfig(dir)

	err := kubeadmcerts.KubeadmCertAPIServer().CreateFromCA(kubeadmConfig, caCert, caKey)
	assert.NilError(t, err)
	apiServerCert := readTestCertificate(t, dir, APIServerCertName)
	assert.NilError(t, apiServerCert.CheckSignatureFrom(caCert))
	assert.Assert(t, apiServerCert.CheckSignatureFrom(serverCACert) != nil)

	err = ensureAPIServerServingCertSignedByServerCA(dir, kubeadmConfig)
	assert.NilError(t, err)

	apiServerCert = readTestCertificate(t, dir, APIServerCertName)
	assert.NilError(t, apiServerCert.CheckSignatureFrom(serverCACert))
	assert.Assert(t, apiServerCert.CheckSignatureFrom(caCert) != nil)
}

func TestGenerateCertificatesPreservesSplitCA(t *testing.T) {
	dir := t.TempDir()
	caCert, caKey := writeTestCA(t, dir, CACertAndKeyBaseName, "k3s-client-ca")
	serverCACert, _ := writeTestCA(t, dir, strings.TrimSuffix(ServerCACertName, ".crt"), "k3s-server-ca")
	err := pkiutil.WriteCertAndKey(dir, strings.TrimSuffix(ClientCACertName, ".crt"), caCert, caKey)
	assert.NilError(t, err)

	err = generateCertificates(dir, testKubeadmConfig(dir))
	assert.NilError(t, err)

	apiServerCert := readTestCertificate(t, dir, APIServerCertName)
	assert.NilError(t, apiServerCert.CheckSignatureFrom(serverCACert))
	assert.Assert(t, apiServerCert.CheckSignatureFrom(caCert) != nil)

	apiServerKubeletClientCert := readTestCertificate(t, dir, APIServerKubeletClientCertName)
	assert.NilError(t, apiServerKubeletClientCert.CheckSignatureFrom(caCert))
	assert.Assert(t, apiServerKubeletClientCert.CheckSignatureFrom(serverCACert) != nil)

	for _, kubeConfigName := range []string{
		AdminKubeConfigFileName,
		ControllerManagerKubeConfigFileName,
		SchedulerKubeConfigFileName,
	} {
		assertKubeConfigTrustsCA(t, filepath.Join(dir, kubeConfigName), readTestCertFile(t, dir, ServerCACertName))
		assertKubeConfigClientCertSignedBy(t, filepath.Join(dir, kubeConfigName), caCert)
	}
}

func TestHasSplitClusterSigningCA(t *testing.T) {
	dir := t.TempDir()
	serverCA := newSelfSignedCertPEM(t, "server-ca", time.Now().Add(365*24*time.Hour))
	clientCA := newSelfSignedCertPEM(t, "client-ca", time.Now().Add(365*24*time.Hour))

	serverCAPath := filepath.Join(dir, "server-ca.crt")
	clientCAPath := filepath.Join(dir, "client-ca.crt")
	writeTestCertFile(t, dir, "server-ca.crt", serverCA)
	writeTestCertFile(t, dir, "client-ca.crt", serverCA)

	split, err := HasSplitClusterSigningCA(serverCAPath, clientCAPath)
	assert.NilError(t, err)
	assert.Equal(t, split, false)

	writeTestCertFile(t, dir, "client-ca.crt", clientCA)
	split, err = HasSplitClusterSigningCA(serverCAPath, clientCAPath)
	assert.NilError(t, err)
	assert.Equal(t, split, true)
}

func TestEnsureKubeConfigTrustsCAClearsCAPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "admin.conf")
	serverCA := []byte("server-ca")

	err := clientcmd.WriteToFile(clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"cluster": {
				Server:                   "https://127.0.0.1:6443",
				CertificateAuthority:     "/old/ca.crt",
				CertificateAuthorityData: []byte("old-ca"),
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"admin": {},
		},
		Contexts: map[string]*clientcmdapi.Context{
			"context": {
				Cluster:  "cluster",
				AuthInfo: "admin",
			},
		},
		CurrentContext: "context",
	}, path)
	assert.NilError(t, err)

	err = ensureKubeConfigTrustsCA(path, serverCA)
	assert.NilError(t, err)

	assertKubeConfigTrustsCA(t, path, serverCA)
}

func writeTestCA(t *testing.T, dir, baseName, commonName string) (*x509.Certificate, crypto.Signer) {
	t.Helper()

	cert, key, err := pkiutil.NewCertificateAuthority(&pkiutil.CertConfig{
		Config: certutil.Config{
			CommonName: commonName,
		},
		NotAfter:            time.Now().Add(365 * 24 * time.Hour),
		EncryptionAlgorithm: kubeadmapi.EncryptionAlgorithmRSA2048,
	})
	assert.NilError(t, err)

	err = pkiutil.WriteCertAndKey(dir, baseName, cert, key)
	assert.NilError(t, err)

	return cert, key
}

func assertKubeConfigTrustsCA(t *testing.T, path string, caPEM []byte) {
	t.Helper()

	kubeConfig, err := clientcmd.LoadFromFile(path)
	assert.NilError(t, err)
	for _, cluster := range kubeConfig.Clusters {
		assert.DeepEqual(t, cluster.CertificateAuthorityData, caPEM)
		assert.Equal(t, cluster.CertificateAuthority, "")
	}
}

func assertKubeConfigClientCertSignedBy(t *testing.T, path string, caCert *x509.Certificate) {
	t.Helper()

	kubeConfig, err := clientcmd.LoadFromFile(path)
	assert.NilError(t, err)
	for _, authInfo := range kubeConfig.AuthInfos {
		clientCert := parseTestCertificate(t, authInfo.ClientCertificateData)
		assert.NilError(t, clientCert.CheckSignatureFrom(caCert))
	}
}

func testKubeadmConfig(dir string) *kubeadmapi.InitConfiguration {
	return &kubeadmapi.InitConfiguration{
		ClusterConfiguration: kubeadmapi.ClusterConfiguration{
			CertificatesDir:     dir,
			EncryptionAlgorithm: kubeadmapi.EncryptionAlgorithmRSA2048,
			Networking: kubeadmapi.Networking{
				ServiceSubnet: "10.96.0.0/12",
				DNSDomain:     "cluster.local",
			},
		},
		NodeRegistration: kubeadmapi.NodeRegistrationOptions{
			Name: "test-node",
		},
		LocalAPIEndpoint: kubeadmapi.APIEndpoint{
			AdvertiseAddress: "127.0.0.1",
			BindPort:         6443,
		},
	}
}

func writeTestCertFile(t *testing.T, dir, name string, data []byte) {
	t.Helper()

	path := filepath.Join(dir, name)
	err := os.MkdirAll(filepath.Dir(path), 0777)
	assert.NilError(t, err)

	err = os.WriteFile(path, data, 0666)
	assert.NilError(t, err)
}

func readTestCertFile(t *testing.T, dir, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(dir, name))
	assert.NilError(t, err)
	return data
}

func readTestCertificate(t *testing.T, dir, name string) *x509.Certificate {
	t.Helper()

	return parseTestCertificate(t, readTestCertFile(t, dir, name))
}

func parseTestCertificate(t *testing.T, certPEM []byte) *x509.Certificate {
	t.Helper()

	block, _ := pem.Decode(certPEM)
	assert.Assert(t, block != nil)
	assert.Equal(t, block.Type, "CERTIFICATE")

	cert, err := x509.ParseCertificate(block.Bytes)
	assert.NilError(t, err)
	return cert
}
