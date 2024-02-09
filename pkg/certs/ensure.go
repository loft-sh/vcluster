package certs

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"golang.org/x/exp/maps"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

func EnsureCerts(
	ctx context.Context,
	serviceCIDR string,
	currentNamespace string,
	currentNamespaceClient kubernetes.Interface,
	vClusterName string,
	certificateDir string,
	clusterDomain string,
	etcdSans []string,
) error {
	// we create a certificate for up to 20 etcd replicas, this should be sufficient for most use cases. Eventually we probably
	// want to update this to the actual etcd number, but for now this is the easiest way to allow up and downscaling without
	// regenerating certificates.
	secretName := vClusterName + "-certs"
	secret, err := currentNamespaceClient.CoreV1().Secrets(currentNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err == nil {
		return downloadCertsFromSecret(secret, certificateDir)
	}

	// we check if the files are already there
	_, err = os.Stat(filepath.Join(certificateDir, CAKeyName))
	if errors.Is(err, fs.ErrNotExist) {
		// try to generate the certificates
		err = generateCertificates(serviceCIDR, vClusterName, certificateDir, clusterDomain, etcdSans)
		if err != nil {
			return err
		}
	}

	// build secret
	secret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: currentNamespace,
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

func generateCertificates(
	serviceCIDR string,
	vClusterName string,
	certificateDir string,
	clusterDomain string,
	etcdSans []string,
) error {
	// init config
	cfg, err := SetInitDynamicDefaults()
	if err != nil {
		return err
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
