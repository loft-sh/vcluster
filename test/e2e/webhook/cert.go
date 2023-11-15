/*
Copyright 2017 The Kubernetes Authors.

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

package webhook

import (
	"crypto/x509"
	"os"

	"github.com/loft-sh/vcluster/test/framework"
	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
)

type certContext struct {
	cert        []byte
	key         []byte
	signingCert []byte
}

// Setup the server cert. For example, user apiservers and admission webhooks
// can use the cert to prove their identify to the kube-apiserver
func setupServerCert(f *framework.Framework, namespaceName, serviceName string) *certContext {
	certDir, err := os.MkdirTemp("", "test-e2e-server-cert")
	if err != nil {
		f.Log.Fatalf("Failed to create a temp dir for cert generation %v", err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(certDir)
	signingKey, err := NewPrivateKey()
	if err != nil {
		f.Log.Fatalf("Failed to create CA private key %v", err)
	}
	signingCert, err := cert.NewSelfSignedCACert(cert.Config{CommonName: "e2e-server-cert-ca"}, signingKey)
	if err != nil {
		f.Log.Fatalf("Failed to create CA cert for apiserver %v", err)
	}
	caCertFile, err := os.CreateTemp(certDir, "ca.crt")
	if err != nil {
		f.Log.Fatalf("Failed to create a temp file for ca cert generation %v", err)
	}
	if err := os.WriteFile(caCertFile.Name(), EncodeCertPEM(signingCert), 0644); err != nil {
		f.Log.Fatalf("Failed to write CA cert %v", err)
	}
	key, err := NewPrivateKey()
	if err != nil {
		f.Log.Fatalf("Failed to create private key for %v", err)
	}
	signedCert, err := NewSignedCert(
		&cert.Config{
			CommonName: serviceName + "." + namespaceName + ".svc",
			AltNames:   cert.AltNames{DNSNames: []string{serviceName + "." + namespaceName + ".svc"}},
			Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		},
		key, signingCert, signingKey,
	)
	if err != nil {
		f.Log.Fatalf("Failed to create cert%v", err)
	}
	certFile, err := os.CreateTemp(certDir, "server.crt")
	if err != nil {
		f.Log.Fatalf("Failed to create a temp file for cert generation %v", err)
	}
	keyFile, err := os.CreateTemp(certDir, "server.key")
	if err != nil {
		f.Log.Fatalf("Failed to create a temp file for key generation %v", err)
	}
	if err = os.WriteFile(certFile.Name(), EncodeCertPEM(signedCert), 0600); err != nil {
		f.Log.Fatalf("Failed to write cert file %v", err)
	}
	privateKeyPEM, err := keyutil.MarshalPrivateKeyToPEM(key)
	if err != nil {
		f.Log.Fatalf("Failed to marshal key %v", err)
	}
	if err = os.WriteFile(keyFile.Name(), privateKeyPEM, 0644); err != nil {
		f.Log.Fatalf("Failed to write key file %v", err)
	}
	return &certContext{
		cert:        EncodeCertPEM(signedCert),
		key:         privateKeyPEM,
		signingCert: EncodeCertPEM(signingCert),
	}
}
