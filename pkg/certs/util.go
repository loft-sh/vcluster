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

package certs

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"k8s.io/klog/v2"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
	netutils "k8s.io/utils/net"
)

const (
	// PublicKeyBlockType is a possible value for pem.Block.Type.
	PublicKeyBlockType = "PUBLIC KEY"
	// CertificateBlockType is a possible value for pem.Block.Type.
	CertificateBlockType = "CERTIFICATE"
	rsaKeySize           = 2048
)

// CertConfig is a wrapper around certutil.Config extending it with PublicKeyAlgorithm.
type CertConfig struct {
	certutil.Config
	NotAfter           *time.Time
	PublicKeyAlgorithm x509.PublicKeyAlgorithm
}

// NewCertificateAuthority creates new certificate and private key for the certificate authority
func NewCertificateAuthority(config *CertConfig) (*x509.Certificate, crypto.Signer, error) {
	key, err := NewPrivateKey(config.PublicKeyAlgorithm)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to create private key while generating CA certificate")
	}

	cert, err := certutil.NewSelfSignedCACert(config.Config, key)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to create self-signed CA certificate")
	}

	return cert, key, nil
}

// NewCertAndKey creates new certificate and key by passing the certificate authority certificate and key
func NewCertAndKey(caCert *x509.Certificate, caKey crypto.Signer, config *CertConfig) (*x509.Certificate, crypto.Signer, error) {
	if len(config.Usages) == 0 {
		return nil, nil, errors.New("must specify at least one ExtKeyUsage")
	}

	key, err := NewPrivateKey(config.PublicKeyAlgorithm)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to create private key")
	}

	cert, err := NewSignedCert(config, key, caCert, caKey, false)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to sign certificate")
	}

	return cert, key, nil
}

// HasServerAuth returns true if the given certificate is a ServerAuth
func HasServerAuth(cert *x509.Certificate) bool {
	for i := range cert.ExtKeyUsage {
		if cert.ExtKeyUsage[i] == x509.ExtKeyUsageServerAuth {
			return true
		}
	}
	return false
}

// WriteCertAndKey stores certificate and key at the specified location
func WriteCertAndKey(pkiPath string, name string, cert *x509.Certificate, key crypto.Signer) error {
	if err := WriteKey(pkiPath, name, key); err != nil {
		return errors.Wrap(err, "couldn't write key")
	}

	return WriteCert(pkiPath, name, cert)
}

// WriteCert stores the given certificate at the given location
func WriteCert(pkiPath, name string, cert *x509.Certificate) error {
	if cert == nil {
		return errors.New("certificate cannot be nil when writing to file")
	}

	certificatePath := pathForCert(pkiPath, name)
	if err := certutil.WriteCert(certificatePath, EncodeCertPEM(cert)); err != nil {
		return errors.Wrapf(err, "unable to write certificate to file %s", certificatePath)
	}

	return nil
}

// WriteKey stores the given key at the given location
func WriteKey(pkiPath, name string, key crypto.Signer) error {
	if key == nil {
		return errors.New("private key cannot be nil when writing to file")
	}

	privateKeyPath := pathForKey(pkiPath, name)
	encoded, err := keyutil.MarshalPrivateKeyToPEM(key)
	if err != nil {
		return errors.Wrapf(err, "unable to marshal private key to PEM")
	}
	if err := keyutil.WriteKey(privateKeyPath, encoded); err != nil {
		return errors.Wrapf(err, "unable to write private key to file %s", privateKeyPath)
	}

	return nil
}

// WritePublicKey stores the given public key at the given location
func WritePublicKey(pkiPath, name string, key crypto.PublicKey) error {
	if key == nil {
		return errors.New("public key cannot be nil when writing to file")
	}

	publicKeyBytes, err := EncodePublicKeyPEM(key)
	if err != nil {
		return err
	}
	publicKeyPath := pathForPublicKey(pkiPath, name)
	if err := keyutil.WriteKey(publicKeyPath, publicKeyBytes); err != nil {
		return errors.Wrapf(err, "unable to write public key to file %s", publicKeyPath)
	}

	return nil
}

// CertOrKeyExist returns a boolean whether the cert or the key exists
func CertOrKeyExist(pkiPath, name string) bool {
	certificatePath, privateKeyPath := PathsForCertAndKey(pkiPath, name)

	_, certErr := os.Stat(certificatePath)
	_, keyErr := os.Stat(privateKeyPath)
	if os.IsNotExist(certErr) && os.IsNotExist(keyErr) {
		// The cert and the key do not exist
		return false
	}

	// Both files exist or one of them
	return true
}

// TryLoadCertAndKeyFromDisk tries to load a cert and a key from the disk and validates that they are valid
func TryLoadCertAndKeyFromDisk(pkiPath, name string) (*x509.Certificate, crypto.Signer, error) {
	cert, err := TryLoadCertFromDisk(pkiPath, name)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to load certificate")
	}

	key, err := TryLoadKeyFromDisk(pkiPath, name)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to load key")
	}

	return cert, key, nil
}

// TryLoadCertFromDisk tries to load the cert from the disk
func TryLoadCertFromDisk(pkiPath, name string) (*x509.Certificate, error) {
	certificatePath := pathForCert(pkiPath, name)

	certs, err := certutil.CertsFromFile(certificatePath)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't load the certificate file %s", certificatePath)
	}

	// We are only putting one certificate in the certificate pem file, so it's safe to just pick the first one
	// TODO: Support multiple certs here in order to be able to rotate certs
	cert := certs[0]

	return cert, nil
}

// TryLoadCertChainFromDisk tries to load the cert chain from the disk
func TryLoadCertChainFromDisk(pkiPath, name string) (*x509.Certificate, []*x509.Certificate, error) {
	certificatePath := pathForCert(pkiPath, name)

	certs, err := certutil.CertsFromFile(certificatePath)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "couldn't load the certificate file %s", certificatePath)
	}

	cert := certs[0]
	intermediates := certs[1:]

	return cert, intermediates, nil
}

// TryLoadKeyFromDisk tries to load the key from the disk and validates that it is valid
func TryLoadKeyFromDisk(pkiPath, name string) (crypto.Signer, error) {
	privateKeyPath := pathForKey(pkiPath, name)

	// Parse the private key from a file
	privKey, err := keyutil.PrivateKeyFromFile(privateKeyPath)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't load the private key file %s", privateKeyPath)
	}

	// Allow RSA and ECDSA formats only
	var key crypto.Signer
	switch k := privKey.(type) {
	case *rsa.PrivateKey:
		key = k
	case *ecdsa.PrivateKey:
		key = k
	default:
		return nil, errors.Errorf("the private key file %s is neither in RSA nor ECDSA format", privateKeyPath)
	}

	return key, nil
}

// PathsForCertAndKey returns the paths for the certificate and key given the path and basename.
func PathsForCertAndKey(pkiPath, name string) (string, string) {
	return pathForCert(pkiPath, name), pathForKey(pkiPath, name)
}

func pathForCert(pkiPath, name string) string {
	return filepath.Join(pkiPath, fmt.Sprintf("%s.crt", name))
}

func pathForKey(pkiPath, name string) string {
	return filepath.Join(pkiPath, fmt.Sprintf("%s.key", name))
}

func pathForPublicKey(pkiPath, name string) string {
	return filepath.Join(pkiPath, fmt.Sprintf("%s.pub", name))
}

// GetControlPlaneEndpoint returns a properly formatted endpoint for the control plane built according following rules:
// - If the controlPlaneEndpoint is defined, use it.
// - if the controlPlaneEndpoint is defined but without a port number, use the controlPlaneEndpoint + localEndpoint.BindPort is used.
// - Otherwise, in case the controlPlaneEndpoint is not defined, use the localEndpoint.AdvertiseAddress + the localEndpoint.BindPort.
func GetControlPlaneEndpoint(controlPlaneEndpoint string, localEndpoint *APIEndpoint) (string, error) {
	// get the URL of the local endpoint
	localAPIEndpoint, err := GetLocalAPIEndpoint(localEndpoint)
	if err != nil {
		return "", err
	}

	// if the controlplane endpoint is defined
	if len(controlPlaneEndpoint) > 0 {
		// parse the controlplane endpoint
		var host, port string
		var err error
		if host, port, err = ParseHostPort(controlPlaneEndpoint); err != nil {
			return "", errors.Wrapf(err, "invalid value %q given for controlPlaneEndpoint", controlPlaneEndpoint)
		}

		// if a port is provided within the controlPlaneAddress warn the users we are using it, else use the bindport
		localEndpointPort := strconv.Itoa(int(localEndpoint.BindPort))
		if port != "" {
			if port != localEndpointPort {
				klog.Infof("[endpoint] WARNING: port specified in controlPlaneEndpoint overrides bindPort in the controlplane address")
			}
		} else {
			port = localEndpointPort
		}

		// overrides the control-plane url using the controlPlaneAddress (and eventually the bindport)
		return formatURL(host, port).String(), nil
	}

	return localAPIEndpoint, nil
}

// parseAPIEndpoint parses an APIEndpoint and returns the AdvertiseAddress as net.IP and the BindPort as string.
// If the BindPort or AdvertiseAddress are invalid it returns an error.
func parseAPIEndpoint(localEndpoint *APIEndpoint) (net.IP, string, error) {
	// parse the bind port
	bindPortString := strconv.Itoa(int(localEndpoint.BindPort))
	if _, err := ParsePort(bindPortString); err != nil {
		return nil, "", errors.Wrapf(err, "invalid value %q given for api.bindPort", localEndpoint.BindPort)
	}

	// parse the AdvertiseAddress
	var ip = net.ParseIP(localEndpoint.AdvertiseAddress)
	if ip == nil {
		return nil, "", errors.Errorf("invalid value `%s` given for api.advertiseAddress", localEndpoint.AdvertiseAddress)
	}

	return ip, bindPortString, nil
}

// formatURL takes a host and a port string and creates a net.URL using https scheme
func formatURL(host, port string) *url.URL {
	return &url.URL{
		Scheme: "https",
		Host:   net.JoinHostPort(host, port),
	}
}

// GetLocalAPIEndpoint parses an APIEndpoint and returns it as a string,
// or returns and error in case it cannot be parsed.
func GetLocalAPIEndpoint(localEndpoint *APIEndpoint) (string, error) {
	// get the URL of the local endpoint
	localEndpointIP, localEndpointPort, err := parseAPIEndpoint(localEndpoint)
	if err != nil {
		return "", err
	}
	url := formatURL(localEndpointIP.String(), localEndpointPort)
	return url.String(), nil
}

// GetKubernetesServiceCIDR returns the default Service CIDR for the Kubernetes internal service
func GetKubernetesServiceCIDR(svcSubnetList string) (*net.IPNet, error) {
	// The default service address family for the cluster is the address family of the first
	// service cluster IP range configured via the `--service-cluster-ip-range` flag
	// of the kube-controller-manager and kube-apiserver.
	svcSubnets, err := netutils.ParseCIDRs(strings.Split(svcSubnetList, ","))
	if err != nil {
		return nil, errors.Wrapf(err, "unable to parse ServiceSubnet %v", svcSubnetList)
	}
	if len(svcSubnets) == 0 {
		return nil, errors.New("received empty ServiceSubnet")
	}
	return svcSubnets[0], nil
}

// GetAPIServerVirtualIP returns the IP of the internal Kubernetes API service
func GetAPIServerVirtualIP(svcSubnetList string) (net.IP, error) {
	svcSubnet, err := GetKubernetesServiceCIDR(svcSubnetList)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get internal Kubernetes Service IP from the given service CIDR")
	}
	internalAPIServerVirtualIP, err := netutils.GetIndexedIP(svcSubnet, 1)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get the first IP address from the given CIDR: %s", svcSubnet.String())
	}
	return internalAPIServerVirtualIP, nil
}

// GetAPIServerAltNames builds an AltNames object for to be used when generating apiserver certificate
func GetAPIServerAltNames(cfg *InitConfiguration) (*certutil.AltNames, error) {
	// advertise address
	advertiseAddress := net.ParseIP(cfg.LocalAPIEndpoint.AdvertiseAddress)
	if advertiseAddress == nil {
		return nil, errors.Errorf("error parsing LocalAPIEndpoint AdvertiseAddress %v: is not a valid textual representation of an IP address",
			cfg.LocalAPIEndpoint.AdvertiseAddress)
	}

	internalAPIServerVirtualIP, err := GetAPIServerVirtualIP(cfg.Networking.ServiceSubnet)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get first IP address from the given CIDR: %v", cfg.Networking.ServiceSubnet)
	}

	// create AltNames with defaults DNSNames/IPs
	altNames := &certutil.AltNames{
		DNSNames: []string{
			cfg.NodeRegistration.Name,
			"kubernetes",
			"kubernetes.default",
			"kubernetes.default.svc",
			fmt.Sprintf("kubernetes.default.svc.%s", cfg.Networking.DNSDomain),
		},
		IPs: []net.IP{
			internalAPIServerVirtualIP,
			advertiseAddress,
		},
	}

	// add cluster controlPlaneEndpoint if present (dns or ip)
	if len(cfg.ControlPlaneEndpoint) > 0 {
		if host, _, err := ParseHostPort(cfg.ControlPlaneEndpoint); err == nil {
			if ip := net.ParseIP(host); ip != nil {
				altNames.IPs = append(altNames.IPs, ip)
			} else {
				altNames.DNSNames = append(altNames.DNSNames, host)
			}
		} else {
			return nil, errors.Wrapf(err, "error parsing cluster controlPlaneEndpoint %q", cfg.ControlPlaneEndpoint)
		}
	}

	appendSANsToAltNames(altNames, cfg.APIServer.CertSANs, APIServerCertName)

	return altNames, nil
}

// ParseHostPort parses a network address of the form "host:port", "ipv4:port", "[ipv6]:port" into host and port;
// ":port" can be eventually omitted.
// If the string is not a valid representation of network address, ParseHostPort returns an error.
func ParseHostPort(hostport string) (string, string, error) {
	var host, port string
	var err error

	// try to split host and port
	if host, port, err = net.SplitHostPort(hostport); err != nil {
		// if SplitHostPort returns an error, the entire hostport is considered as host
		host = hostport
	}

	// if port is defined, parse and validate it
	if port != "" {
		if _, err := ParsePort(port); err != nil {
			return "", "", errors.Errorf("hostport %s: port %s must be a valid number between 1 and 65535, inclusive", hostport, port)
		}
	}

	// if host is a valid IP, returns it
	if ip := net.ParseIP(host); ip != nil {
		return host, port, nil
	}

	// if host is a validate RFC-1123 subdomain, returns it
	if errs := validation.IsDNS1123Subdomain(host); len(errs) == 0 {
		return host, port, nil
	}

	return "", "", errors.Errorf("hostport %s: host '%s' must be a valid IP address or a valid RFC-1123 DNS subdomain", hostport, host)
}

// ParsePort parses a string representing a TCP port.
// If the string is not a valid representation of a TCP port, ParsePort returns an error.
func ParsePort(port string) (int, error) {
	portInt, err := netutils.ParsePort(port, true)
	if err == nil && (1 <= portInt && portInt <= 65535) {
		return portInt, nil
	}

	return 0, errors.New("port must be a valid number between 1 and 65535, inclusive")
}

// GetEtcdAltNames builds an AltNames object for generating the etcd server certificate.
// `advertise address` and localhost are included in the SAN since this is the interfaces the etcd static pod listens on.
// The user can override the listen address with `Etcd.ExtraArgs` and add SANs with `Etcd.ServerCertSANs`.
func GetEtcdAltNames(cfg *InitConfiguration) (*certutil.AltNames, error) {
	return getAltNames(cfg, EtcdServerCertName)
}

// GetEtcdPeerAltNames builds an AltNames object for generating the etcd peer certificate.
// Hostname and `API.AdvertiseAddress` are included if the user chooses to promote the single node etcd cluster into a multi-node one (stacked etcd).
// The user can override the listen address with `Etcd.ExtraArgs` and add SANs with `Etcd.PeerCertSANs`.
func GetEtcdPeerAltNames(cfg *InitConfiguration) (*certutil.AltNames, error) {
	return getAltNames(cfg, EtcdPeerCertName)
}

// getAltNames builds an AltNames object with the cfg and certName.
func getAltNames(cfg *InitConfiguration, certName string) (*certutil.AltNames, error) {
	// advertise address
	advertiseAddress := net.ParseIP(cfg.LocalAPIEndpoint.AdvertiseAddress)
	if advertiseAddress == nil {
		return nil, errors.Errorf("error parsing LocalAPIEndpoint AdvertiseAddress %v: is not a valid textual representation of an IP address",
			cfg.LocalAPIEndpoint.AdvertiseAddress)
	}

	// create AltNames with defaults DNSNames/IPs
	altNames := &certutil.AltNames{
		DNSNames: []string{cfg.NodeRegistration.Name, "localhost"},
		IPs:      []net.IP{advertiseAddress, net.IPv4(127, 0, 0, 1), net.IPv6loopback},
	}

	if cfg.Etcd.Local != nil {
		if certName == EtcdServerCertName {
			appendSANsToAltNames(altNames, cfg.Etcd.Local.ServerCertSANs, EtcdServerCertName)
		} else if certName == EtcdPeerCertName {
			appendSANsToAltNames(altNames, cfg.Etcd.Local.PeerCertSANs, EtcdPeerCertName)
		}
	}
	return altNames, nil
}

// appendSANsToAltNames parses SANs from as list of strings and adds them to altNames for use on a specific cert
// altNames is passed in with a pointer, and the struct is modified
// valid IP address strings are parsed and added to altNames.IPs as net.IP's
// RFC-1123 compliant DNS strings are added to altNames.DNSNames as strings
// RFC-1123 compliant wildcard DNS strings are added to altNames.DNSNames as strings
// certNames is used to print user facing warnings and should be the name of the cert the altNames will be used for
func appendSANsToAltNames(altNames *certutil.AltNames, SANs []string, certName string) {
	for _, altname := range SANs {
		if ip := net.ParseIP(altname); ip != nil {
			altNames.IPs = append(altNames.IPs, ip)
		} else if len(validation.IsDNS1123Subdomain(altname)) == 0 {
			altNames.DNSNames = append(altNames.DNSNames, altname)
		} else if len(validation.IsWildcardDNS1123Subdomain(altname)) == 0 {
			altNames.DNSNames = append(altNames.DNSNames, altname)
		} else {
			klog.Infof(
				"[certificates] WARNING: '%s' was not added to the '%s' SAN, because it is not a valid IP or RFC-1123 compliant DNS entry",
				altname,
				certName,
			)
		}
	}
}

// EncodeCertPEM returns PEM-endcoded certificate data
func EncodeCertPEM(cert *x509.Certificate) []byte {
	block := pem.Block{
		Type:  CertificateBlockType,
		Bytes: cert.Raw,
	}
	return pem.EncodeToMemory(&block)
}

// EncodePublicKeyPEM returns PEM-encoded public data
func EncodePublicKeyPEM(key crypto.PublicKey) ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return []byte{}, err
	}
	block := pem.Block{
		Type:  PublicKeyBlockType,
		Bytes: der,
	}
	return pem.EncodeToMemory(&block), nil
}

// NewPrivateKey returns a new private key.
var NewPrivateKey = GeneratePrivateKey

func GeneratePrivateKey(keyType x509.PublicKeyAlgorithm) (crypto.Signer, error) {
	if keyType == x509.ECDSA {
		return ecdsa.GenerateKey(elliptic.P256(), cryptorand.Reader)
	}

	return rsa.GenerateKey(cryptorand.Reader, rsaKeySize)
}

// NewSignedCert creates a signed certificate using the given CA certificate and key
func NewSignedCert(cfg *CertConfig, key crypto.Signer, caCert *x509.Certificate, caKey crypto.Signer, isCA bool) (*x509.Certificate, error) {
	serial, err := cryptorand.Int(cryptorand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	if len(cfg.CommonName) == 0 {
		return nil, errors.New("must specify a CommonName")
	}

	keyUsage := x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
	if isCA {
		keyUsage |= x509.KeyUsageCertSign
	}

	RemoveDuplicateAltNames(&cfg.AltNames)

	notAfter := time.Now().Add(CertificateValidity).UTC()
	if cfg.NotAfter != nil {
		notAfter = *cfg.NotAfter
	}

	certTmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName:   cfg.CommonName,
			Organization: cfg.Organization,
		},
		DNSNames:              cfg.AltNames.DNSNames,
		IPAddresses:           cfg.AltNames.IPs,
		SerialNumber:          serial,
		NotBefore:             caCert.NotBefore,
		NotAfter:              notAfter,
		KeyUsage:              keyUsage,
		ExtKeyUsage:           cfg.Usages,
		BasicConstraintsValid: true,
		IsCA:                  isCA,
	}
	certDERBytes, err := x509.CreateCertificate(cryptorand.Reader, &certTmpl, caCert, key.Public(), caKey)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}

// RemoveDuplicateAltNames removes duplicate items in altNames.
func RemoveDuplicateAltNames(altNames *certutil.AltNames) {
	if altNames == nil {
		return
	}

	if altNames.DNSNames != nil {
		altNames.DNSNames = sets.NewString(altNames.DNSNames...).List()
	}

	ipsKeys := make(map[string]struct{})
	var ips []net.IP
	for _, one := range altNames.IPs {
		if _, ok := ipsKeys[one.String()]; !ok {
			ipsKeys[one.String()] = struct{}{}
			ips = append(ips, one)
		}
	}
	altNames.IPs = ips
}

// ValidateCertPeriod checks if the certificate is valid relative to the current time
// (+/- offset)
func ValidateCertPeriod(cert *x509.Certificate, offset time.Duration) error {
	period := fmt.Sprintf("NotBefore: %v, NotAfter: %v", cert.NotBefore, cert.NotAfter)
	now := time.Now().Add(offset)
	if now.Before(cert.NotBefore) {
		return errors.Errorf("the certificate is not valid yet: %s", period)
	}
	if now.After(cert.NotAfter) {
		return errors.Errorf("the certificate has expired: %s", period)
	}
	return nil
}

// VerifyCertChain verifies that a certificate has a valid chain of
// intermediate CAs back to the root CA
func VerifyCertChain(cert *x509.Certificate, intermediates []*x509.Certificate, root *x509.Certificate) error {
	rootPool := x509.NewCertPool()
	rootPool.AddCert(root)

	intermediatePool := x509.NewCertPool()
	for _, c := range intermediates {
		intermediatePool.AddCert(c)
	}

	verifyOptions := x509.VerifyOptions{
		Roots:         rootPool,
		Intermediates: intermediatePool,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}

	if _, err := cert.Verify(verifyOptions); err != nil {
		return err
	}

	return nil
}
