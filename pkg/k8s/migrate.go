package k8s

import (
	"context"
	"fmt"
	"os"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/certs"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	if _, err := os.Stat("/data/server/tls"); err != nil { // fast path
		return nil
	} else if options.Config.PrivateNodes.Enabled {
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

func MigrateK3sToK8sStateless(ctx context.Context, currentNamespaceClient kubernetes.Interface, currentNamespace string, vClusterClient client.Client, options *config.VirtualClusterConfig) error {
	if options.BackingStoreType() != vclusterconfig.StoreTypeDeployedEtcd && options.BackingStoreType() != vclusterconfig.StoreTypeExternalEtcd && options.BackingStoreType() != vclusterconfig.StoreTypeExternalDatabase {
		return nil
	} else if options.PrivateNodes.Enabled {
		return nil
	}

	// get k3s secret
	secretName := "vc-k3s-" + options.Name
	k3sSecret, err := currentNamespaceClient.CoreV1().Secrets(currentNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		// this is fine, we can just skip this
		return nil
	} else if k3sSecret.Annotations[migratedFromK3sAnnotation] == "true" { // already migrated
		return nil
	}

	// migrating means deleting all pods in the vcluster and all kube-root-ca.crt configmaps
	klog.Infof("Recreating pods and certificates since we are migrating from k3s to k8s...")

	// delete all configmaps
	configMapList := &corev1.ConfigMapList{}
	err = vClusterClient.List(ctx, configMapList)
	if err != nil {
		return fmt.Errorf("failed to list configmaps in vcluster: %w", err)
	}
	for _, configMap := range configMapList.Items {
		if configMap.Name == "kube-root-ca.crt" {
			err = vClusterClient.Delete(ctx, &configMap)
			if err != nil {
				return fmt.Errorf("failed to delete configmap %s: %w", configMap.Name, err)
			}
		}
	}

	// now delete all pods
	podList := &corev1.PodList{}
	err = vClusterClient.List(ctx, podList)
	if err != nil {
		return fmt.Errorf("failed to list pods in vcluster: %w", err)
	}
	for _, pod := range podList.Items {
		err = vClusterClient.Delete(ctx, &pod)
		if err != nil {
			return fmt.Errorf("failed to delete pod %s: %w", pod.Name, err)
		}
	}

	// patch secret
	oldSecret := k3sSecret.DeepCopy()
	if k3sSecret.Annotations == nil {
		k3sSecret.Annotations = make(map[string]string)
	}
	k3sSecret.Annotations[migratedFromK3sAnnotation] = "true"
	patch := client.MergeFrom(oldSecret)
	patchBytes, err := patch.Data(k3sSecret)
	if err != nil {
		return fmt.Errorf("failed to create patch for k3s secret %s: %w", secretName, err)
	}
	_, err = currentNamespaceClient.CoreV1().Secrets(currentNamespace).Patch(ctx, secretName, patch.Type(), patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to patch k3s secret %s: %w", secretName, err)
	}

	return nil
}

func renameIfExists(oldPath, newPath string) error {
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return nil
	}

	return os.Rename(oldPath, newPath)
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
