package certs

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/loft-sh/vcluster/pkg/config"
	"golang.org/x/exp/maps"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
	"k8s.io/klog/v2"
)

func EnsureCerts(
	ctx context.Context,
	serviceCIDR string,
	currentNamespace string,
	currentNamespaceClient kubernetes.Interface,
	vClusterName string,
	certificateDir string,
	etcdSans []string,
	options *config.VirtualClusterConfig,
) error {
	// we create a certificate for up to 20 etcd replicas, this should be sufficient for most use cases. Eventually we probably
	// want to update this to the actual etcd number, but for now this is the easiest way to allow up and downscaling without
	// regenerating certificates.
	secretName := vClusterName + "-certs"
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
		cfg, err := createConfig(serviceCIDR, vClusterName, certificateDir, options.Networking.Advanced.ClusterDomain, etcdSans)
		if err != nil {
			return err
		}
		err = os.Remove(filepath.Join(certificateDir, "apiserver.crt"))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		err = os.Remove(filepath.Join(certificateDir, "apiserver.key"))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}

		// only create the files if the files are not there yet
		err = CreatePKIAssets(cfg)
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
		err = generateCertificates(serviceCIDR, vClusterName, certificateDir, options.Networking.Advanced.ClusterDomain, etcdSans)
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
		controlPlaneService.TypeMeta.APIVersion = "v1"
		controlPlaneService.TypeMeta.Kind = "Service"

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

func createConfig(serviceCIDR, vClusterName, certificateDir, clusterDomain string, etcdSans []string) (*InitConfiguration, error) {
	// init config
	cfg, err := SetInitDynamicDefaults()
	if err != nil {
		return nil, err
	}

	cfg.ClusterName = "kubernetes"
	cfg.NodeRegistration.Name = vClusterName
	cfg.Etcd.Local = &LocalEtcd{
		ServerCertSANs: etcdSans,
		PeerCertSANs:   etcdSans,
	}
	cfg.Networking.ServiceSubnet = serviceCIDR
	cfg.Networking.DNSDomain = clusterDomain
	cfg.ControlPlaneEndpoint = "127.0.0.1:6443"
	cfg.CertificatesDir = certificateDir
	cfg.LocalAPIEndpoint.AdvertiseAddress = "0.0.0.0"
	cfg.LocalAPIEndpoint.BindPort = 443

	return cfg, nil
}

func generateCertificates(
	serviceCIDR string,
	vClusterName string,
	certificateDir string,
	clusterDomain string,
	etcdSans []string,
) error {
	// init config
	cfg, err := createConfig(serviceCIDR, vClusterName, certificateDir, clusterDomain, etcdSans)
	if err != nil {
		return err
	}

	// only create the files if the files are not there yet
	err = CreatePKIAssets(cfg)
	if err != nil {
		return fmt.Errorf("create pki assets: %w", err)
	}

	err = CreateJoinControlPlaneKubeConfigFiles(cfg.CertificatesDir, cfg)
	if err != nil {
		return fmt.Errorf("create kube configs: %w", err)
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
		if v.Server != "https://127.0.0.1:6443" {
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
