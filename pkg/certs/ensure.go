package certs

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/pkg/config"
	"golang.org/x/exp/maps"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
	"k8s.io/klog/v2"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/certs"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/kubeconfig"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
	kubeconfigutil "k8s.io/kubernetes/cmd/kubeadm/app/util/kubeconfig"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pkiutil"
)

const (
	CertSecretLabelAppKey          = "app"
	CertSecretLabelAppValue        = "vcluster"
	CertSecretLabelVclusterNameKey = "vcluster-name"
)

func EnsureCerts(
	ctx context.Context,
	currentNamespace string,
	currentNamespaceClient kubernetes.Interface,
	certificateDir string,
	options *config.VirtualClusterConfig,
	kubeadmConfig *kubeadmapi.InitConfiguration,
) error {
	// when we run in standalone mode, we don't have a currentNamespaceClient
	if currentNamespaceClient == nil {
		if !options.ControlPlane.Standalone.Enabled {
			return errors.New("nil currentNamespaceClient")
		}

		// we check if the files are already there
		_, err := os.Stat(filepath.Join(certificateDir, CAKeyName))
		if errors.Is(err, fs.ErrNotExist) {
			// try to generate the certificates
			err = generateCertificates(certificateDir, kubeadmConfig)
			if err != nil {
				return err
			}
		}

		return nil
	}

	// we create a certificate for up to 5 etcd replicas, this should be sufficient for most use cases. Eventually we probably
	// want to update this to the actual etcd number, but for now this is the easiest way to allow up and downscaling without
	// regenerating certificates.
	secretName := CertSecretName(options.Name)
	secret, err := currentNamespaceClient.CoreV1().Secrets(currentNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err == nil {
		// download certs from secret
		err = downloadCertsFromSecret(secret, certificateDir)
		if err != nil {
			return err
		}

		// update kube config
		shouldUpdate, err := updateKubeconfigInSecret(secret)
		if err != nil {
			return err
		} else if !shouldUpdate {
			return nil
		}

		// delete the certs and recreate them
		klog.Info("removing outdated certs")
		err = os.Remove(filepath.Join(certificateDir, "apiserver.crt"))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		err = os.Remove(filepath.Join(certificateDir, "apiserver.key"))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}

		// only create the files if the files are not there yet
		err = certs.CreatePKIAssets(kubeadmConfig)
		if err != nil {
			// ignore the error because some other certs are upsetting the function
			klog.V(1).Info("create pki assets err:", err)
		}
		cert, err := os.ReadFile(filepath.Join(certificateDir, "apiserver.crt"))
		if err != nil {
			return err
		}
		key, err := os.ReadFile(filepath.Join(certificateDir, "apiserver.key"))
		if err != nil {
			return err
		}
		secret.Data["apiserver.crt"] = cert
		secret.Data["apiserver.key"] = key
		_, err = currentNamespaceClient.CoreV1().Secrets(currentNamespace).Update(ctx, secret, metav1.UpdateOptions{})
		if err != nil {
			return err
		}

		return downloadCertsFromSecret(secret, certificateDir)
	}

	// we check if the files are already there
	_, err = os.Stat(filepath.Join(certificateDir, CAKeyName))
	if errors.Is(err, fs.ErrNotExist) {
		// try to generate the certificates
		err = generateCertificates(certificateDir, kubeadmConfig)
		if err != nil {
			return err
		}
	}

	ownerRef := []metav1.OwnerReference{}
	if options.Experimental.SyncSettings.SetOwner {
		// options.ServiceName gets rewritten to the workload service name so we use options.Name as the helm chart
		// directly uses the release name for the service name
		controlPlaneService, err := currentNamespaceClient.CoreV1().Services(currentNamespace).Get(ctx, options.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("get vcluster service: %w", err)
		}
		// client doesn't populate typemeta
		controlPlaneService.APIVersion = "v1"
		controlPlaneService.Kind = "Service"

		ownerRef = append(ownerRef, metav1.OwnerReference{
			APIVersion: "v1",
			Kind:       "Service",
			Name:       controlPlaneService.Name,
			UID:        controlPlaneService.UID,
		})
	}

	// build secret
	secret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:            secretName,
			Namespace:       currentNamespace,
			OwnerReferences: ownerRef,
			Labels: map[string]string{
				CertSecretLabelAppKey:          CertSecretLabelAppValue,
				CertSecretLabelVclusterNameKey: options.Name,
			},
		},
		Data: map[string][]byte{},
	}
	for fromName, toName := range certMap {
		data, err := os.ReadFile(filepath.Join(certificateDir, fromName))
		if err != nil {
			return fmt.Errorf("read %s: %w", fromName, err)
		}

		secret.Data[toName] = data
	}

	// find extra files in the folder and add them to the secret
	extraFiles, err := extraFiles(certificateDir)
	if err != nil {
		return fmt.Errorf("read extra file: %w", err)
	}
	for k, v := range extraFiles {
		secret.Data[k] = v
	}

	// finally create the secret
	secret, err = currentNamespaceClient.CoreV1().Secrets(currentNamespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		if !kerrors.IsAlreadyExists(err) {
			return fmt.Errorf("create certs secret: %w", err)
		}

		// get secret
		secret, err = currentNamespaceClient.CoreV1().Secrets(currentNamespace).Get(ctx, secretName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("retrieve certs secret: %w", err)
		}
	} else {
		klog.Infof("Successfully created certs secret %s/%s", currentNamespace, secretName)
	}

	return downloadCertsFromSecret(secret, certificateDir)
}

func CertSecretName(vClusterName string) string {
	return vClusterName + "-certs"
}

func generateCertificates(
	certificateDir string,
	kubeadmConfig *kubeadmapi.InitConfiguration,
) error {
	// only create the files if the files are not there yet
	err := certs.CreatePKIAssets(kubeadmConfig)
	if err != nil {
		return fmt.Errorf("create pki assets: %w", err)
	}

	// create kube config files
	err = kubeconfig.CreateJoinControlPlaneKubeConfigFiles(certificateDir, kubeadmConfig)
	if err != nil {
		return fmt.Errorf("create kube configs: %w", err)
	}

	// create super admin kube config file
	err = kubeconfig.CreateKubeConfigFile(kubeadmconstants.SuperAdminKubeConfigFileName, certificateDir, kubeadmConfig)
	if err != nil {
		return fmt.Errorf("create kube config: %w", err)
	}

	// rename super-admin.conf to admin.conf
	err = os.Rename(filepath.Join(certificateDir, kubeadmconstants.SuperAdminKubeConfigFileName), filepath.Join(certificateDir, kubeadmconstants.AdminKubeConfigFileName))
	if err != nil {
		return fmt.Errorf("rename kube config: %w", err)
	}

	err = splitCACert(certificateDir)
	if err != nil {
		return fmt.Errorf("split ca cert: %w", err)
	}

	return nil
}

// downloadCertsFromSecret writes to the filesystem the content of each field in the secret
// if the field has an equivalent inside the certmap, we write with the corresponding name
// otherwise the file has the same name than the field
func downloadCertsFromSecret(
	secret *corev1.Secret,
	certificateDir string,
) error {
	certMapValues := maps.Values(certMap)
	for secretEntry, fileBytes := range secret.Data {
		name := secretEntry
		if slices.Contains(certMapValues, secretEntry) {
			// we need to replace with the actual name
			for key, sEntry := range certMap {
				// guarranteed to evaluate to true at least once because of slices.contains
				if sEntry == secretEntry {
					if len(fileBytes) == 0 {
						return fmt.Errorf("secret is missing %s", secretEntry)
					}
					name = key
					break
				}
			}
		}

		name = filepath.Join(certificateDir, name)
		err := os.MkdirAll(filepath.Dir(name), 0777)
		if err != nil {
			return fmt.Errorf("create directory %s", filepath.Dir(name))
		}

		err = os.WriteFile(name, fileBytes, 0666)
		if err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}

	err := splitCACert(certificateDir)
	if err != nil {
		return fmt.Errorf("split ca cert: %w", err)
	}

	return nil
}

func splitCACert(certificateDir string) error {
	// make sure to write server-ca and client-ca to file system
	err := copyFileIfNotExists(filepath.Join(certificateDir, CACertName), filepath.Join(certificateDir, ServerCACertName))
	if err != nil {
		return fmt.Errorf("copy %s: %w", ServerCACertName, err)
	}
	err = copyFileIfNotExists(filepath.Join(certificateDir, CAKeyName), filepath.Join(certificateDir, ServerCAKeyName))
	if err != nil {
		return fmt.Errorf("copy %s: %w", ServerCAKeyName, err)
	}
	err = copyFileIfNotExists(filepath.Join(certificateDir, CACertName), filepath.Join(certificateDir, ClientCACertName))
	if err != nil {
		return fmt.Errorf("copy %s: %w", ClientCACertName, err)
	}
	err = copyFileIfNotExists(filepath.Join(certificateDir, CAKeyName), filepath.Join(certificateDir, ClientCAKeyName))
	if err != nil {
		return fmt.Errorf("copy %s: %w", ClientCAKeyName, err)
	}

	return nil
}

func copyFileIfNotExists(src, dst string) error {
	_, err := os.Stat(dst)
	if os.IsNotExist(err) {
		srcBytes, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("read %s: %w", src, err)
		}

		return os.WriteFile(dst, srcBytes, 0666)
	}
	return nil
}

func extraFiles(certificateDir string) (map[string][]byte, error) {
	files := make(map[string][]byte)
	entries, err := os.ReadDir(certificateDir)
	if err != nil {
		return nil, err
	}

	for _, v := range entries {
		if v.IsDir() {
			// ignore subdirectories for now
			// etcd files should be picked up by the map
			continue
		}

		// if it's not in the cert map, add to the map
		name := v.Name()
		_, ok := certMap[name]
		if !ok {
			b, err := os.ReadFile(filepath.Join(certificateDir, name))
			if err != nil {
				return nil, err
			}

			files[name] = b
		}
	}

	return files, err
}

func updateKubeconfigToLocalhost(config *clientcmdapi.Config) bool {
	updated := false
	// not sure what that would do in case of multiple clusters,
	// but this is not expected AFAIU
	for k, v := range config.Clusters {
		if v == nil {
			continue
		}

		if v.Server != "https://127.0.0.1:6443" {
			if config.Clusters[k] == nil {
				config.Clusters[k] = &clientcmdapi.Cluster{}
			}

			config.Clusters[k].Server = "https://127.0.0.1:6443"
			updated = true
		}
	}
	return updated
}

func updateKubeconfigInSecret(secret *corev1.Secret) (shouldUpdate bool, err error) {
	shouldUpdate = false
	for k, v := range secret.Data {
		if !strings.HasSuffix(k, ".conf") {
			continue
		}
		config := &clientcmdapi.Config{}
		err = runtime.DecodeInto(clientcmdlatest.Codec, v, config)
		if err != nil {
			return false, err
		}
		hasChanged := updateKubeconfigToLocalhost(config)
		if !hasChanged {
			continue
		}
		shouldUpdate = true

		marshalled, err := runtime.Encode(clientcmdlatest.Codec, config)
		if err != nil {
			return false, err
		}
		secret.Data[k] = marshalled
	}
	return shouldUpdate, nil
}

// KubeConfigOptions struct holds info required to build a KubeConfig object
type KubeConfigOptions struct {
	CACert        string
	CAKey         string
	Organizations []string
	APIServer     string
	ClientName    string
}

// CreateKubeConfig creates a kubeconfig object and writes it to disk
func CreateKubeConfig(spec *KubeConfigOptions, path string) error {
	config, err := BuildKubeConfig(spec)
	if err != nil {
		return fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	return kubeconfigutil.WriteToDisk(path, config)
}

// BuildKubeConfig creates a kubeconfig object for the given kubeConfigSpec
func BuildKubeConfig(spec *KubeConfigOptions) (*clientcmdapi.Config, error) {
	caCert, err := certutil.CertsFromFile(spec.CACert)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate: %w", err)
	}
	caKeyRaw, err := keyutil.PrivateKeyFromFile(spec.CAKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA key: %w", err)
	}

	// Allow RSA and ECDSA formats only
	var caKey crypto.Signer
	switch k := caKeyRaw.(type) {
	case *rsa.PrivateKey:
		caKey = k
	case *ecdsa.PrivateKey:
		caKey = k
	default:
		return nil, fmt.Errorf("the private key file %s is neither in RSA nor ECDSA format", spec.CAKey)
	}

	// we need to set the not after to the start time + 10 years
	notAfter := kubeadmutil.StartTimeUTC().Add(time.Hour * 24 * 365 * 10)

	// otherwise, create a client cert
	clientCertConfig := pkiutil.CertConfig{
		Config: certutil.Config{
			CommonName:   spec.ClientName,
			Organization: spec.Organizations,
			Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		},
		NotAfter:            notAfter,
		EncryptionAlgorithm: kubeadmapi.EncryptionAlgorithmRSA2048,
	}

	clientCert, clientKey, err := pkiutil.NewCertAndKey(caCert[0], caKey, &clientCertConfig)
	if err != nil {
		return nil, fmt.Errorf("failure while creating %s client certificate: %w", spec.ClientName, err)
	}

	encodedClientKey, err := keyutil.MarshalPrivateKeyToPEM(clientKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key to PEM: %w", err)
	}
	// create a kubeconfig with the client certs
	return kubeconfigutil.CreateWithCerts(
		spec.APIServer,
		"default",
		spec.ClientName,
		pkiutil.EncodeCertPEM(caCert[0]),
		encodedClientKey,
		pkiutil.EncodeCertPEM(clientCert),
	), nil
}
