package k8s

import (
	"context"
	"fmt"
	"os"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/certs"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

var (
	migratedFromK3sAnnotation = "vcluster.loft.sh/migrated-from-k3s"

	k3sKubeConfig = map[string]string{
		certs.AdminKubeConfigFileName:             "/data/server/cred/admin.kubeconfig",
		certs.ControllerManagerKubeConfigFileName: "/data/server/cred/controller.kubeconfig",
		certs.SchedulerKubeConfigFileName:         "/data/server/cred/scheduler.kubeconfig",
	}

	k3sTLS = map[string]string{
		certs.APIServerCertName: "/data/server/tls/serving-kube-apiserver.crt",
		certs.APIServerKeyName:  "/data/server/tls/serving-kube-apiserver.key",

		certs.ServerCACertName: "/data/server/tls/server-ca.crt",
		certs.ServerCAKeyName:  "/data/server/tls/server-ca.key",

		certs.ClientCACertName: "/data/server/tls/client-ca.crt",
		certs.ClientCAKeyName:  "/data/server/tls/client-ca.key",

		certs.FrontProxyCACertName: "/data/server/tls/request-header-ca.crt",
		certs.FrontProxyCAKeyName:  "/data/server/tls/request-header-ca.key",

		certs.FrontProxyClientCertName: "/data/server/tls/client-auth-proxy.crt",
		certs.FrontProxyClientKeyName:  "/data/server/tls/client-auth-proxy.key",

		certs.ServiceAccountPrivateKeyName: "/data/server/tls/service.current.key",
		certs.ServiceAccountPublicKeyName:  "/data/server/tls/service.key",
	}
)

func MigrateK3sToK8s(ctx context.Context, currentNamespaceClient kubernetes.Interface, currentNamespace string, options *config.VirtualClusterConfig) error {
	// only migrate if we are migrating from k3s to k8s
	if options.Distro() != vclusterconfig.K8SDistro {
		return nil
	} else if _, err := os.Stat("/data/server/tls"); err != nil { // fast path
		return nil
	}

	// migrate data first
	if options.BackingStoreType() == vclusterconfig.StoreTypeEmbeddedDatabase || options.BackingStoreType() == vclusterconfig.StoreTypeEmbeddedEtcd {
		// copy over the data
		err := renameIfExists(constants.K3sSqliteDatabase, constants.K8sSqliteDatabase)
		if err != nil {
			return fmt.Errorf("failed to rename sqlite database: %w", err)
		}
		err = renameIfExists(constants.K3sSqliteDatabase+"-wal", constants.K8sSqliteDatabase+"-wal")
		if err != nil {
			return fmt.Errorf("failed to rename sqlite database: %w", err)
		}
		err = renameIfExists(constants.K3sSqliteDatabase+"-shm", constants.K8sSqliteDatabase+"-shm")
		if err != nil {
			return fmt.Errorf("failed to rename sqlite database: %w", err)
		}
	}

	// get the secret
	secret, err := currentNamespaceClient.CoreV1().Secrets(currentNamespace).Get(ctx, certs.CertSecretName(options.Name), metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			// this is fine, we can just skip this
			return nil
		}

		return fmt.Errorf("failed to get certificate secret %s: %w", certs.CertSecretName(options.Name), err)
	} else if secret.Annotations[migratedFromK3sAnnotation] == "true" { // already migrated
		return nil
	}

	// convert tls secrets
	for inSecretName, fileName := range k3sKubeConfig {
		if _, err := os.Stat(fileName); os.IsNotExist(err) {
			return nil
		}

		secret.Data[inSecretName], err = fillKubeConfig(fileName)
		if err != nil {
			klog.Errorf("failed to fill k3s kube config %s: %s", fileName, err)
			return err
		}
	}

	// convert kube configs
	for inSecretName, fileName := range k3sTLS {
		if _, err := os.Stat(fileName); os.IsNotExist(err) {
			return nil
		}

		secret.Data[inSecretName], err = os.ReadFile(fileName)
		if err != nil {
			klog.Errorf("failed to read k3s tls secret %s: %s", fileName, err)
			return err
		}
	}

	// update secret
	klog.Infof("Migrating k3s distro certificates to k8s...")
	if secret.Annotations == nil {
		secret.Annotations = make(map[string]string)
	}
	secret.Annotations[migratedFromK3sAnnotation] = "true"
	_, err = currentNamespaceClient.CoreV1().Secrets(secret.Namespace).Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		if kerrors.IsConflict(err) {
			klog.Infof("failed to migrate k3s tls secret %s: %s, retrying", secret.Name, err)

			// get the secret again
			secret, err = currentNamespaceClient.CoreV1().Secrets(secret.Namespace).Get(ctx, secret.Name, metav1.GetOptions{})
			if err != nil {
				klog.Errorf("failed to get k3s tls secret %s: %s", secret.Name, err)
				return err
			}

			return MigrateK3sToK8s(ctx, currentNamespaceClient, currentNamespace, options)
		}

		klog.Errorf("failed to migrate k3s tls secret %s: %s", secret.Name, err)
		return err
	}

	// remove old data
	_ = os.RemoveAll("/data/server")
	return nil
}

func renameIfExists(old, new string) error {
	if _, err := os.Stat(old); os.IsNotExist(err) {
		return nil
	}

	return os.Rename(old, new)
}

func fillKubeConfig(kubeConfigPath string) ([]byte, error) {
	config, err := clientcmd.LoadFromFile(kubeConfigPath)
	if err != nil {
		return nil, err
	}

	// exchange kube config server & resolve certificate
	for _, cluster := range config.Clusters {
		if cluster == nil {
			continue
		}

		// fill in data
		if cluster.CertificateAuthorityData == nil && cluster.CertificateAuthority != "" {
			o, err := os.ReadFile(cluster.CertificateAuthority)
			if err != nil {
				return nil, err
			}

			cluster.CertificateAuthority = ""
			cluster.CertificateAuthorityData = o
		}

		cluster.Server = "https://127.0.0.1:6443"
	}

	// resolve auth info cert & key
	for _, authInfo := range config.AuthInfos {
		if authInfo == nil {
			continue
		}

		// fill in data
		if authInfo.ClientCertificateData == nil && authInfo.ClientCertificate != "" {
			o, err := os.ReadFile(authInfo.ClientCertificate)
			if err != nil {
				return nil, err
			}

			authInfo.ClientCertificate = ""
			authInfo.ClientCertificateData = o
		}
		if authInfo.ClientKeyData == nil && authInfo.ClientKey != "" {
			o, err := os.ReadFile(authInfo.ClientKey)
			if err != nil {
				return nil, err
			}

			authInfo.ClientKey = ""
			authInfo.ClientKeyData = o
		}
	}

	return clientcmd.Write(*config)
}
