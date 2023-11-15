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
	"crypto/x509"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"

	"k8s.io/client-go/util/keyutil"
	"k8s.io/klog/v2"
)

var (
	// certPeriodValidation is used to store if period validation was done for a certificate
	certPeriodValidationMutex sync.Mutex
	certPeriodValidation      = map[string]struct{}{}
)

// CreatePKIAssets will create and write to disk all PKI assets necessary to establish the control plane.
// If the PKI assets already exists in the target folder, they are used only if evaluated equal; otherwise an error is returned.
func CreatePKIAssets(cfg *InitConfiguration) error {
	klog.V(1).Infoln("creating PKI assets")

	// This structure cannot handle multilevel CA hierarchies.
	// This isn't a problem right now, but may become one in the future.

	var certList Certificates
	if cfg.Etcd.Local == nil {
		certList = GetCertsWithoutEtcd()
	} else {
		certList = GetDefaultCertList()
	}

	certTree, err := certList.AsMap().CertTree()
	if err != nil {
		return err
	}

	if err := certTree.CreateTree(cfg); err != nil {
		return errors.Wrap(err, "error creating PKI assets")
	}

	klog.Infof("Valid certificates and keys now exist in %q", cfg.CertificatesDir)

	// Service accounts are not x509 certs, so handled separately
	return CreateServiceAccountKeyAndPublicKeyFiles(cfg.CertificatesDir, cfg.ClusterConfiguration.PublicKeyAlgorithm())
}

// CreateServiceAccountKeyAndPublicKeyFiles creates new public/private key files for signing service account users.
// If the sa public/private key files already exist in the target folder, they are used only if evaluated equals; otherwise an error is returned.
func CreateServiceAccountKeyAndPublicKeyFiles(certsDir string, keyType x509.PublicKeyAlgorithm) error {
	klog.V(1).Infoln("creating new public/private key files for signing service account users")
	_, err := keyutil.PrivateKeyFromFile(filepath.Join(certsDir, ServiceAccountPrivateKeyName))
	if err == nil {
		// kubeadm doesn't validate the existing certificate key more than this;
		// Basically, if we find a key file with the same path kubeadm thinks those files
		// are equal and doesn't bother writing a new file
		klog.Infof("[certs] Using the existing %q key", ServiceAccountKeyBaseName)
		return nil
	} else if !os.IsNotExist(err) {
		return errors.Wrapf(err, "file %s existed but it could not be loaded properly", ServiceAccountPrivateKeyName)
	}

	// The key does NOT exist, let's generate it now
	key, err := NewPrivateKey(keyType)
	if err != nil {
		return err
	}

	// Write .key and .pub files to disk
	klog.Infof("[certs] Generating %q key and public key", ServiceAccountKeyBaseName)

	if err := WriteKey(certsDir, ServiceAccountKeyBaseName, key); err != nil {
		return err
	}

	return WritePublicKey(certsDir, ServiceAccountKeyBaseName, key.Public())
}

// writeCertificateAuthorityFilesIfNotExist write a new certificate Authority to the given path.
// If there already is a certificate file at the given path; kubeadm tries to load it and check if the values in the
// existing and the expected certificate equals. If they do; kubeadm will just skip writing the file as it's up-to-date,
// otherwise this function returns an error.
func writeCertificateAuthorityFilesIfNotExist(pkiDir string, baseName string, caCert *x509.Certificate, caKey crypto.Signer) error {
	// If cert or key exists, we should try to load them
	if CertOrKeyExist(pkiDir, baseName) {
		// Try to load .crt and .key from the PKI directory
		caCert, _, err := TryLoadCertAndKeyFromDisk(pkiDir, baseName)
		if err != nil {
			return errors.Wrapf(err, "failure loading %s certificate", baseName)
		}
		// Validate period
		CheckCertificatePeriodValidity(baseName, caCert)

		// Check if the existing cert is a CA
		if !caCert.IsCA {
			return errors.Errorf("certificate %s is not a CA", baseName)
		}

		// kubeadm doesn't validate the existing certificate Authority more than this;
		// Basically, if we find a certificate file with the same path; and it is a CA
		// kubeadm thinks those files are equal and doesn't bother writing a new file
		klog.Infof("Using the existing %q certificate and key", baseName)
	} else {
		// Write .crt and .key files to disk
		klog.Infof("Generating %q certificate and key", baseName)
		if err := WriteCertAndKey(pkiDir, baseName, caCert, caKey); err != nil {
			return errors.Wrapf(err, "failure while saving %s certificate and key", baseName)
		}
	}
	return nil
}

// writeCertificateFilesIfNotExist write a new certificate to the given path.
// If there already is a certificate file at the given path; kubeadm tries to load it and check if the values in the
// existing and the expected certificate equals. If they do; kubeadm will just skip writing the file as it's up-to-date,
// otherwise this function returns an error.
func writeCertificateFilesIfNotExist(pkiDir string, baseName string, signingCert *x509.Certificate, cert *x509.Certificate, key crypto.Signer, cfg *CertConfig) error {
	// Checks if the signed certificate exists in the PKI directory
	if CertOrKeyExist(pkiDir, baseName) {
		// Try to load key from the PKI directory
		_, err := TryLoadKeyFromDisk(pkiDir, baseName)
		if err != nil {
			return errors.Wrapf(err, "failure loading %s key", baseName)
		}

		// Try to load certificate from the PKI directory
		signedCert, intermediates, err := TryLoadCertChainFromDisk(pkiDir, baseName)
		if err != nil {
			return errors.Wrapf(err, "failure loading %s certificate", baseName)
		}
		// Validate period
		CheckCertificatePeriodValidity(baseName, signedCert)

		// Check if the existing cert is signed by the given CA
		if err := VerifyCertChain(signedCert, intermediates, signingCert); err != nil {
			return errors.Errorf("certificate %s is not signed by corresponding CA", baseName)
		}

		// Check if the certificate has the correct attributes
		if err := validateCertificateWithConfig(signedCert, baseName, cfg); err != nil {
			return err
		}

		klog.Infof("[certs] Using the existing %q certificate and key", baseName)
	} else {
		// Write .crt and .key files to disk
		klog.Infof("[certs] Generating %q certificate and key", baseName)

		if err := WriteCertAndKey(pkiDir, baseName, cert, key); err != nil {
			return errors.Wrapf(err, "failure while saving %s certificate and key", baseName)
		}
		if HasServerAuth(cert) {
			klog.Infof("[certs] %s serving cert is signed for DNS names %v and IPs %v", baseName, cert.DNSNames, cert.IPAddresses)
		}
	}

	return nil
}

type certKeyLocation struct {
	pkiDir   string
	baseName string
	uxName   string
}

// validateSignedCertWithCA tries to load a certificate and private key and
// validates that the cert is signed by the given caCert
func validateSignedCertWithCA(l certKeyLocation, caCert *x509.Certificate) error {
	// Try to load key from the PKI directory
	_, err := TryLoadKeyFromDisk(l.pkiDir, l.baseName)
	if err != nil {
		return errors.Wrapf(err, "failure loading key for %s", l.baseName)
	}

	// Try to load certificate from the PKI directory
	signedCert, intermediates, err := TryLoadCertChainFromDisk(l.pkiDir, l.baseName)
	if err != nil {
		return errors.Wrapf(err, "failure loading certificate for %s", l.uxName)
	}
	// Validate period
	CheckCertificatePeriodValidity(l.uxName, signedCert)

	// Check if the cert is signed by the CA
	if err := VerifyCertChain(signedCert, intermediates, caCert); err != nil {
		return errors.Wrapf(err, "certificate %s is not signed by corresponding CA", l.uxName)
	}
	return nil
}

// validateCertificateWithConfig makes sure that a given certificate is valid at
// least for the SANs defined in the configuration.
func validateCertificateWithConfig(cert *x509.Certificate, baseName string, cfg *CertConfig) error {
	for _, dnsName := range cfg.AltNames.DNSNames {
		if err := cert.VerifyHostname(dnsName); err != nil {
			return errors.Wrapf(err, "certificate %s is invalid", baseName)
		}
	}
	for _, ipAddress := range cfg.AltNames.IPs {
		if err := cert.VerifyHostname(ipAddress.String()); err != nil {
			return errors.Wrapf(err, "certificate %s is invalid", baseName)
		}
	}
	return nil
}

// CheckCertificatePeriodValidity takes a certificate and prints a warning if its period
// is not valid related to the current time. It does so only if the certificate was not validated already
// by keeping track with a cache.
func CheckCertificatePeriodValidity(baseName string, cert *x509.Certificate) {
	certPeriodValidationMutex.Lock()
	defer certPeriodValidationMutex.Unlock()
	if _, exists := certPeriodValidation[baseName]; exists {
		return
	}
	certPeriodValidation[baseName] = struct{}{}

	klog.V(5).Infof("validating certificate period for %s certificate", baseName)
	if err := ValidateCertPeriod(cert, 0); err != nil {
		klog.Warningf("WARNING: could not validate bounds for certificate %s: %v", baseName, err)
	}
}
