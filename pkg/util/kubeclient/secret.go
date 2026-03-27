package kubeclient

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	DefaultSecretPrefix = "vc-"
	KubeconfigSecretKey = "config"
)

// GetDefaultSecretName returns the name of the kubeconfig Secret for a vcluster with the given suffix.
func GetDefaultSecretName(suffix string) string {
	return DefaultSecretPrefix + suffix
}

// ReadKubeConfig reads the kubeconfig from the vcluster's kubeconfig Secret.
func ReadKubeConfig(ctx context.Context, client *kubernetes.Clientset, suffix, namespace string) (*clientcmdapi.Config, error) {
	secretName := GetDefaultSecretName(suffix)
	secret, err := client.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not Get the %s secret in order to read kubeconfig: %w", secretName, err)
	}
	config, found := secret.Data[KubeconfigSecretKey]
	if !found {
		return nil, fmt.Errorf("could not find the kube config (%s key) in the %s secret", KubeconfigSecretKey, secretName)
	}
	return clientcmd.Load(config)
}
