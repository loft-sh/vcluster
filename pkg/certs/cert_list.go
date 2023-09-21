/*
Copyright 2018 The Kubernetes Authors.
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
	"crypto/x509"

	"github.com/pkg/errors"

	certutil "k8s.io/client-go/util/cert"
)

type configMutatorsFunc func(*InitConfiguration, *CertConfig) error

// KubeadmCert represents a certificate that Kubeadm will create to function properly.
type KubeadmCert struct {
	Name     string
	LongName string
	BaseName string
	CAName   string
	// Some attributes will depend on the InitConfiguration, only known at runtime.
	// These functions will be run in series, passed both the InitConfiguration and a cert Config.
	configMutators []configMutatorsFunc
	config         CertConfig
}

// GetConfig returns the definition for the given cert given the provided InitConfiguration
func (k *KubeadmCert) GetConfig(ic *InitConfiguration) (*CertConfig, error) {
	for _, f := range k.configMutators {
		if err := f(ic, &k.config); err != nil {
			return nil, err
		}
	}

	k.config.PublicKeyAlgorithm = ic.ClusterConfiguration.PublicKeyAlgorithm()
	return &k.config, nil
}

// CreateFromCA makes and writes a certificate using the given CA cert and key.
func (k *KubeadmCert) CreateFromCA(ic *InitConfiguration, caCert *x509.Certificate, caKey crypto.Signer) error {
	cfg, err := k.GetConfig(ic)
	if err != nil {
		return errors.Wrapf(err, "couldn't create %q certificate", k.Name)
	}
	cert, key, err := NewCertAndKey(caCert, caKey, cfg)
	if err != nil {
		return err
	}
	err = writeCertificateFilesIfNotExist(
		ic.CertificatesDir,
		k.BaseName,
		caCert,
		cert,
		key,
		cfg,
	)

	if err != nil {
		return errors.Wrapf(err, "failed to write or validate certificate %q", k.Name)
	}

	return nil
}

// CreateAsCA creates a certificate authority, writing the files to disk and also returning the created CA so it can be used to sign child certs.
func (k *KubeadmCert) CreateAsCA(ic *InitConfiguration) (*x509.Certificate, crypto.Signer, error) {
	cfg, err := k.GetConfig(ic)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "couldn't get configuration for %q CA certificate", k.Name)
	}
	caCert, caKey, err := NewCertificateAuthority(cfg)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "couldn't generate %q CA certificate", k.Name)
	}

	err = writeCertificateAuthorityFilesIfNotExist(
		ic.CertificatesDir,
		k.BaseName,
		caCert,
		caKey,
	)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "couldn't write out %q CA certificate", k.Name)
	}

	return caCert, caKey, nil
}

// CertificateTree is represents a one-level-deep tree, mapping a CA to the certs that depend on it.
type CertificateTree map[*KubeadmCert]Certificates

// CreateTree creates the CAs, certs signed by the CAs, and writes them all to disk.
func (t CertificateTree) CreateTree(ic *InitConfiguration) error {
	for ca, leaves := range t {
		cfg, err := ca.GetConfig(ic)
		if err != nil {
			return err
		}

		var caKey crypto.Signer

		caCert, err := TryLoadCertFromDisk(ic.CertificatesDir, ca.BaseName)
		if err == nil {
			// Validate period
			CheckCertificatePeriodValidity(ca.BaseName, caCert)

			// Cert exists already, make sure it's valid
			if !caCert.IsCA {
				return errors.Errorf("certificate %q is not a CA", ca.Name)
			}
			// Try and load a CA Key
			caKey, err = TryLoadKeyFromDisk(ic.CertificatesDir, ca.BaseName)
			if err != nil {
				// If there's no CA key, make sure every certificate exists.
				for _, leaf := range leaves {
					cl := certKeyLocation{
						pkiDir:   ic.CertificatesDir,
						baseName: leaf.BaseName,
						uxName:   leaf.Name,
					}
					if err := validateSignedCertWithCA(cl, caCert); err != nil {
						return errors.Wrapf(err, "could not load expected certificate %q or validate the existence of key %q for it", leaf.Name, ca.Name)
					}
				}
				continue
			}
			// CA key exists; just use that to create new certificates.
		} else {
			// CACert doesn't already exist, create a new cert and key.
			caCert, caKey, err = NewCertificateAuthority(cfg)
			if err != nil {
				return err
			}

			err = writeCertificateAuthorityFilesIfNotExist(
				ic.CertificatesDir,
				ca.BaseName,
				caCert,
				caKey,
			)
			if err != nil {
				return err
			}
		}

		for _, leaf := range leaves {
			if err := leaf.CreateFromCA(ic, caCert, caKey); err != nil {
				return err
			}
		}
	}
	return nil
}

// CertificateMap is a flat map of certificates, keyed by Name.
type CertificateMap map[string]*KubeadmCert

// CertTree returns a one-level-deep tree, mapping a CA cert to an array of certificates that should be signed by it.
func (m CertificateMap) CertTree() (CertificateTree, error) {
	caMap := make(CertificateTree)

	for _, cert := range m {
		if cert.CAName == "" {
			if _, ok := caMap[cert]; !ok {
				caMap[cert] = []*KubeadmCert{}
			}
		} else {
			ca, ok := m[cert.CAName]
			if !ok {
				return nil, errors.Errorf("certificate %q references unknown CA %q", cert.Name, cert.CAName)
			}
			caMap[ca] = append(caMap[ca], cert)
		}
	}

	return caMap, nil
}

// Certificates is a list of Certificates that Kubeadm should create.
type Certificates []*KubeadmCert

// AsMap returns the list of certificates as a map, keyed by name.
func (c Certificates) AsMap() CertificateMap {
	certMap := make(map[string]*KubeadmCert)
	for _, cert := range c {
		certMap[cert.Name] = cert
	}

	return certMap
}

// GetDefaultCertList returns  all of the certificates kubeadm requires to function.
func GetDefaultCertList() Certificates {
	return Certificates{
		KubeadmCertRootCA(),
		KubeadmCertAPIServer(),
		KubeadmCertKubeletClient(),
		// Front Proxy certs
		KubeadmCertFrontProxyCA(),
		KubeadmCertFrontProxyClient(),
		// etcd certs
		KubeadmCertEtcdCA(),
		KubeadmCertEtcdServer(),
		KubeadmCertEtcdPeer(),
		KubeadmCertEtcdHealthcheck(),
		KubeadmCertEtcdAPIClient(),
	}
}

// GetCertsWithoutEtcd returns all of the certificates kubeadm needs when etcd is hosted externally.
func GetCertsWithoutEtcd() Certificates {
	return Certificates{
		KubeadmCertRootCA(),
		KubeadmCertAPIServer(),
		KubeadmCertKubeletClient(),
		// Front Proxy certs
		KubeadmCertFrontProxyCA(),
		KubeadmCertFrontProxyClient(),
	}
}

// KubeadmCertRootCA is the definition of the Kubernetes Root CA for the API Server and kubelet.
func KubeadmCertRootCA() *KubeadmCert {
	return &KubeadmCert{
		Name:     "ca",
		LongName: "self-signed Kubernetes CA to provision identities for other Kubernetes components",
		BaseName: CACertAndKeyBaseName,
		config: CertConfig{
			Config: certutil.Config{
				CommonName: "kubernetes",
			},
		},
	}
}

// KubeadmCertAPIServer is the definition of the cert used to serve the Kubernetes API.
func KubeadmCertAPIServer() *KubeadmCert {
	return &KubeadmCert{
		Name:     "apiserver",
		LongName: "certificate for serving the Kubernetes API",
		BaseName: APIServerCertAndKeyBaseName,
		CAName:   "ca",
		config: CertConfig{
			Config: certutil.Config{
				CommonName: APIServerCertCommonName,
				Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			},
		},
		configMutators: []configMutatorsFunc{
			makeAltNamesMutator(GetAPIServerAltNames),
		},
	}
}

// KubeadmCertKubeletClient is the definition of the cert used by the API server to access the kubelet.
func KubeadmCertKubeletClient() *KubeadmCert {
	return &KubeadmCert{
		Name:     "apiserver-kubelet-client",
		LongName: "certificate for the API server to connect to kubelet",
		BaseName: APIServerKubeletClientCertAndKeyBaseName,
		CAName:   "ca",
		config: CertConfig{
			Config: certutil.Config{
				CommonName:   APIServerKubeletClientCertCommonName,
				Organization: []string{SystemPrivilegedGroup},
				Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			},
		},
	}
}

// KubeadmCertFrontProxyCA is the definition of the CA used for the front end proxy.
func KubeadmCertFrontProxyCA() *KubeadmCert {
	return &KubeadmCert{
		Name:     "front-proxy-ca",
		LongName: "self-signed CA to provision identities for front proxy",
		BaseName: FrontProxyCACertAndKeyBaseName,
		config: CertConfig{
			Config: certutil.Config{
				CommonName: "front-proxy-ca",
			},
		},
	}
}

// KubeadmCertFrontProxyClient is the definition of the cert used by the API server to access the front proxy.
func KubeadmCertFrontProxyClient() *KubeadmCert {
	return &KubeadmCert{
		Name:     "front-proxy-client",
		BaseName: FrontProxyClientCertAndKeyBaseName,
		LongName: "certificate for the front proxy client",
		CAName:   "front-proxy-ca",
		config: CertConfig{
			Config: certutil.Config{
				CommonName: FrontProxyClientCertCommonName,
				Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			},
		},
	}
}

// KubeadmCertEtcdCA is the definition of the root CA used by the hosted etcd server.
func KubeadmCertEtcdCA() *KubeadmCert {
	return &KubeadmCert{
		Name:     "etcd-ca",
		LongName: "self-signed CA to provision identities for etcd",
		BaseName: EtcdCACertAndKeyBaseName,
		config: CertConfig{
			Config: certutil.Config{
				CommonName: "etcd-ca",
			},
		},
	}
}

// KubeadmCertEtcdServer is the definition of the cert used to serve etcd to clients.
func KubeadmCertEtcdServer() *KubeadmCert {
	return &KubeadmCert{
		Name:     "etcd-server",
		LongName: "certificate for serving etcd",
		BaseName: EtcdServerCertAndKeyBaseName,
		CAName:   "etcd-ca",
		config: CertConfig{
			Config: certutil.Config{
				// TODO: etcd 3.2 introduced an undocumented requirement for ClientAuth usage on the
				// server cert: https://github.com/coreos/etcd/issues/9785#issuecomment-396715692
				// Once the upstream issue is resolved, this should be returned to only allowing
				// ServerAuth usage.
				Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
			},
		},
		configMutators: []configMutatorsFunc{
			makeAltNamesMutator(GetEtcdAltNames),
			setCommonNameToNodeName(),
		},
	}
}

// KubeadmCertEtcdPeer is the definition of the cert used by etcd peers to access each other.
func KubeadmCertEtcdPeer() *KubeadmCert {
	return &KubeadmCert{
		Name:     "etcd-peer",
		LongName: "certificate for etcd nodes to communicate with each other",
		BaseName: EtcdPeerCertAndKeyBaseName,
		CAName:   "etcd-ca",
		config: CertConfig{
			Config: certutil.Config{
				Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
			},
		},
		configMutators: []configMutatorsFunc{
			makeAltNamesMutator(GetEtcdPeerAltNames),
			setCommonNameToNodeName(),
		},
	}
}

// KubeadmCertEtcdHealthcheck is the definition of the cert used by Kubernetes to check the health of the etcd server.
func KubeadmCertEtcdHealthcheck() *KubeadmCert {
	return &KubeadmCert{
		Name:     "etcd-healthcheck-client",
		LongName: "certificate for liveness probes to healthcheck etcd",
		BaseName: EtcdHealthcheckClientCertAndKeyBaseName,
		CAName:   "etcd-ca",
		config: CertConfig{
			Config: certutil.Config{
				CommonName:   EtcdHealthcheckClientCertCommonName,
				Organization: []string{SystemPrivilegedGroup},
				Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			},
		},
	}
}

// KubeadmCertEtcdAPIClient is the definition of the cert used by the API server to access etcd.
func KubeadmCertEtcdAPIClient() *KubeadmCert {
	return &KubeadmCert{
		Name:     "apiserver-etcd-client",
		LongName: "certificate the apiserver uses to access etcd",
		BaseName: APIServerEtcdClientCertAndKeyBaseName,
		CAName:   "etcd-ca",
		config: CertConfig{
			Config: certutil.Config{
				CommonName:   APIServerEtcdClientCertCommonName,
				Organization: []string{SystemPrivilegedGroup},
				Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			},
		},
	}
}

func makeAltNamesMutator(f func(*InitConfiguration) (*certutil.AltNames, error)) configMutatorsFunc {
	return func(mc *InitConfiguration, cc *CertConfig) error {
		altNames, err := f(mc)
		if err != nil {
			return err
		}
		cc.AltNames = *altNames
		return nil
	}
}

func setCommonNameToNodeName() configMutatorsFunc {
	return func(mc *InitConfiguration, cc *CertConfig) error {
		cc.CommonName = mc.NodeRegistration.Name
		return nil
	}
}
