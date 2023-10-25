package certs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

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
) error {
	// we create a certificate for up to 20 etcd replicas, this should be sufficient for most use cases. Eventually we probably
	// want to update this to the actual etcd number, but for now this is the easiest way to allow up and downscaling without
	// regenerating certificates.
	etcdReplicas := 20
	secretName := vClusterName + "-certs"
	secret, err := currentNamespaceClient.CoreV1().Secrets(currentNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err == nil {
		return downloadCertsFromSecret(secret, certificateDir)
	}

	// init config
	cfg, err := SetInitDynamicDefaults()
	if err != nil {
		return err
	}

	// generate etcd server and peer sans
	etcdService := vClusterName + "-etcd"
	serverSans := []string{etcdService, etcdService + "." + currentNamespace, etcdService + "." + currentNamespace + ".svc"}
	for i := 0; i < etcdReplicas; i++ {
		hostname := etcdService + "-" + strconv.Itoa(i)
		serverSans = append(serverSans, hostname, hostname+"."+etcdService+"-headless", hostname+"."+etcdService+"-headless"+"."+currentNamespace)
	}

	cfg.ClusterName = "kubernetes"
	cfg.NodeRegistration.Name = vClusterName + "-api"
	cfg.Etcd.Local = &LocalEtcd{
		ServerCertSANs: serverSans,
		PeerCertSANs:   serverSans,
	}
	cfg.Networking.ServiceSubnet = serviceCIDR
	cfg.Networking.DNSDomain = clusterDomain
	cfg.ControlPlaneEndpoint = vClusterName + "-api"
	cfg.CertificatesDir = certificateDir
	cfg.LocalAPIEndpoint.AdvertiseAddress = "0.0.0.0"
	cfg.LocalAPIEndpoint.BindPort = 443
	err = CreatePKIAssets(cfg)
	if err != nil {
		return fmt.Errorf("create pki assets: %w", err)
	}

	err = CreateJoinControlPlaneKubeConfigFiles(cfg.CertificatesDir, cfg)
	if err != nil {
		return fmt.Errorf("create kube configs: %w", err)
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

func downloadCertsFromSecret(
	secret *corev1.Secret,
	certificateDir string,
) error {
	for toName, fromName := range certMap {
		if len(secret.Data[fromName]) == 0 {
			return fmt.Errorf("secret is missing %s", fromName)
		}

		err := os.WriteFile(filepath.Join(certificateDir, toName), secret.Data[fromName], 0666)
		if err != nil {
			return fmt.Errorf("write %s: %w", fromName, err)
		}
	}

	return nil
}
