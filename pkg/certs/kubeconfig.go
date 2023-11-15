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
	"bytes"
	"crypto"
	"crypto/x509"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
	"k8s.io/klog/v2"
)

// clientCertAuth struct holds info required to build a client certificate to provide authentication info in a kubeconfig object
type clientCertAuth struct {
	CAKey         crypto.Signer
	Organizations []string
}

// tokenAuth struct holds info required to use a token to provide authentication info in a kubeconfig object
type tokenAuth struct {
	Token string `datapolicy:"token"`
}

// kubeConfigSpec struct holds info required to build a KubeConfig object
type kubeConfigSpec struct {
	CACert         *x509.Certificate
	APIServer      string
	ClientName     string
	TokenAuth      *tokenAuth      `datapolicy:"token"`
	ClientCertAuth *clientCertAuth `datapolicy:"security-key"`
}

// CreateJoinControlPlaneKubeConfigFiles will create and write to disk the kubeconfig files required by kubeadm
// join --control-plane workflow, plus the admin kubeconfig file used by the administrator and kubeadm itself; the
// kubelet.conf file must not be created because it will be created and signed by the kubelet TLS bootstrap process.
// When not using external CA mode, if a kubeconfig file already exists it is used only if evaluated equal,
// otherwise an error is returned. For external CA mode, the creation of kubeconfig files is skipped.
func CreateJoinControlPlaneKubeConfigFiles(outDir string, cfg *InitConfiguration) error {
	var externaCA bool
	caKeyPath := filepath.Join(cfg.CertificatesDir, CAKeyName)
	if _, err := os.Stat(caKeyPath); os.IsNotExist(err) {
		externaCA = true
	}

	files := []string{
		AdminKubeConfigFileName,
		ControllerManagerKubeConfigFileName,
		SchedulerKubeConfigFileName,
	}

	for _, file := range files {
		if externaCA {
			klog.Infof("[kubeconfig] External CA mode: Using user provided %s", file)
			continue
		}
		if err := createKubeConfigFiles(outDir, cfg, file); err != nil {
			return err
		}
	}
	return nil
}

// createKubeConfigFiles creates all the requested kubeconfig files.
// If kubeconfig files already exists, they are used only if evaluated equal; otherwise an error is returned.
func createKubeConfigFiles(outDir string, cfg *InitConfiguration, kubeConfigFileNames ...string) error {
	// gets the KubeConfigSpecs, actualized for the current InitConfiguration
	specs, err := getKubeConfigSpecs(cfg)
	if err != nil {
		return err
	}

	for _, kubeConfigFileName := range kubeConfigFileNames {
		// retrieves the KubeConfigSpec for given kubeConfigFileName
		spec, exists := specs[kubeConfigFileName]
		if !exists {
			return errors.Errorf("couldn't retrieve KubeConfigSpec for %s", kubeConfigFileName)
		}

		// builds the KubeConfig object
		config, err := buildKubeConfigFromSpec(spec, cfg.ClusterName, nil)
		if err != nil {
			return err
		}

		// writes the kubeconfig to disk if it does not exist
		if err = createKubeConfigFileIfNotExists(outDir, kubeConfigFileName, config); err != nil {
			return err
		}
	}

	return nil
}

// getKubeConfigSpecs returns all KubeConfigSpecs actualized to the context of the current InitConfiguration
// NB. this method holds the information about how kubeadm creates kubeconfig files.
func getKubeConfigSpecs(cfg *InitConfiguration) (map[string]*kubeConfigSpec, error) {
	caCert, caKey, err := TryLoadCertAndKeyFromDisk(cfg.CertificatesDir, CACertAndKeyBaseName)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create a kubeconfig; the CA files couldn't be loaded")
	}
	// Validate period
	CheckCertificatePeriodValidity(CACertAndKeyBaseName, caCert)

	configs, err := getKubeConfigSpecsBase(cfg)
	if err != nil {
		return nil, err
	}
	for _, spec := range configs {
		spec.CACert = caCert
		spec.ClientCertAuth.CAKey = caKey
	}
	return configs, nil
}

// buildKubeConfigFromSpec creates a kubeconfig object for the given kubeConfigSpec
func buildKubeConfigFromSpec(spec *kubeConfigSpec, clustername string, notAfter *time.Time) (*clientcmdapi.Config, error) {
	// If this kubeconfig should use token
	if spec.TokenAuth != nil {
		// create a kubeconfig with a token
		return CreateWithToken(
			spec.APIServer,
			clustername,
			spec.ClientName,
			EncodeCertPEM(spec.CACert),
			spec.TokenAuth.Token,
		), nil
	}

	// otherwise, create a client certs
	clientCertConfig := newClientCertConfigFromKubeConfigSpec(spec, notAfter)

	clientCert, clientKey, err := NewCertAndKey(spec.CACert, spec.ClientCertAuth.CAKey, &clientCertConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "failure while creating %s client certificate", spec.ClientName)
	}

	encodedClientKey, err := keyutil.MarshalPrivateKeyToPEM(clientKey)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal private key to PEM")
	}
	// create a kubeconfig with the client certs
	return CreateWithCerts(
		spec.APIServer,
		clustername,
		spec.ClientName,
		EncodeCertPEM(spec.CACert),
		encodedClientKey,
		EncodeCertPEM(clientCert),
	), nil
}

func newClientCertConfigFromKubeConfigSpec(spec *kubeConfigSpec, notAfter *time.Time) CertConfig {
	return CertConfig{
		Config: certutil.Config{
			CommonName:   spec.ClientName,
			Organization: spec.ClientCertAuth.Organizations,
			Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		},
		NotAfter: notAfter,
	}
}

// validateKubeConfig check if the kubeconfig file exist and has the expected CA and server URL
func validateKubeConfig(outDir, filename string, config *clientcmdapi.Config) error {
	kubeConfigFilePath := filepath.Join(outDir, filename)

	if _, err := os.Stat(kubeConfigFilePath); err != nil {
		return err
	}

	// The kubeconfig already exists, let's check if it has got the same CA and server URL
	currentConfig, err := clientcmd.LoadFromFile(kubeConfigFilePath)
	if err != nil {
		return errors.Wrapf(err, "failed to load kubeconfig file %s that already exists on disk", kubeConfigFilePath)
	}

	expectedCtx, exists := config.Contexts[config.CurrentContext]
	if !exists {
		return errors.Errorf("failed to find expected context %s", config.CurrentContext)
	}
	expectedCluster := expectedCtx.Cluster
	currentCtx, exists := currentConfig.Contexts[currentConfig.CurrentContext]
	if !exists {
		return errors.Errorf("failed to find CurrentContext in Contexts of the kubeconfig file %s", kubeConfigFilePath)
	}
	currentCluster := currentCtx.Cluster
	if currentConfig.Clusters[currentCluster] == nil {
		return errors.Errorf("failed to find the given CurrentContext Cluster in Clusters of the kubeconfig file %s", kubeConfigFilePath)
	}

	// Make sure the compared CAs are whitespace-trimmed. The function clientcmd.LoadFromFile() just decodes
	// the base64 CA and places it raw in the v1.Config object. In case the user has extra whitespace
	// in the CA they used to create a kubeconfig this comparison to a generated v1.Config will otherwise fail.
	caCurrent := bytes.TrimSpace(currentConfig.Clusters[currentCluster].CertificateAuthorityData)
	caExpected := bytes.TrimSpace(config.Clusters[expectedCluster].CertificateAuthorityData)

	// If the current CA cert on disk doesn't match the expected CA cert, error out because we have a file, but it's stale
	if !bytes.Equal(caCurrent, caExpected) {
		return errors.Errorf("a kubeconfig file %q exists already but has got the wrong CA cert", kubeConfigFilePath)
	}
	// If the current API Server location on disk doesn't match the expected API server, show a warning
	if currentConfig.Clusters[currentCluster].Server != config.Clusters[expectedCluster].Server {
		klog.Warningf("a kubeconfig file %q exists already but has an unexpected API Server URL: expected: %s, got: %s",
			kubeConfigFilePath, config.Clusters[expectedCluster].Server, currentConfig.Clusters[currentCluster].Server)
	}

	return nil
}

// createKubeConfigFileIfNotExists saves the KubeConfig object into a file if there isn't any file at the given path.
// If there already is a kubeconfig file at the given path; kubeadm tries to load it and check if the values in the
// existing and the expected config equals. If they do; kubeadm will just skip writing the file as it's up-to-date,
// but if a file exists but has old content or isn't a kubeconfig file, this function returns an error.
func createKubeConfigFileIfNotExists(outDir, filename string, config *clientcmdapi.Config) error {
	kubeConfigFilePath := filepath.Join(outDir, filename)

	err := validateKubeConfig(outDir, filename, config)
	if err != nil {
		// Check if the file exist, and if it doesn't, just write it to disk
		if !os.IsNotExist(err) {
			return err
		}
		klog.Infof("[kubeconfig] Writing %q kubeconfig file", filename)
		err = WriteToDisk(kubeConfigFilePath, config)
		if err != nil {
			return errors.Wrapf(err, "failed to save kubeconfig file %q on disk", kubeConfigFilePath)
		}
		return nil
	}
	// kubeadm doesn't validate the existing kubeconfig file more than this (kubeadm trusts the client certs to be valid)
	// Basically, if we find a kubeconfig file with the same path; the same CA cert and the same server URL;
	// kubeadm thinks those files are equal and doesn't bother writing a new file
	klog.Infof("[kubeconfig] Using existing kubeconfig file: %q", kubeConfigFilePath)

	return nil
}

func getKubeConfigSpecsBase(cfg *InitConfiguration) (map[string]*kubeConfigSpec, error) {
	controlPlaneEndpoint, err := GetControlPlaneEndpoint(cfg.ControlPlaneEndpoint, &cfg.LocalAPIEndpoint)
	if err != nil {
		return nil, err
	}

	return map[string]*kubeConfigSpec{
		AdminKubeConfigFileName: {
			APIServer:  controlPlaneEndpoint,
			ClientName: "kubernetes-admin",
			ClientCertAuth: &clientCertAuth{
				Organizations: []string{SystemPrivilegedGroup},
			},
		},
		ControllerManagerKubeConfigFileName: {
			APIServer:      controlPlaneEndpoint,
			ClientName:     ControllerManagerUser,
			ClientCertAuth: &clientCertAuth{},
		},
		SchedulerKubeConfigFileName: {
			APIServer:      controlPlaneEndpoint,
			ClientName:     SchedulerUser,
			ClientCertAuth: &clientCertAuth{},
		},
	}, nil
}
