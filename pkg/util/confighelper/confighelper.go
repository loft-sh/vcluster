package confighelper

import (
	"context"
	"fmt"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// ConfigFileName is the name of the file within the Secret or ConfigMap containing the vCluster configuration
	ConfigFileName = "config.yaml"
	// ConfigNamePrefix is the prefix for vCluster configuration resources
	ConfigNamePrefix = "vc-config-"
	// AnnotationDistro is the annotation key for the vCluster distro type
	AnnotationDistro = "vcluster.loft.sh/distro"
	// AnnotationStore is the annotation key for the vCluster store type
	AnnotationStore = "vcluster.loft.sh/store"
)

// GetResourceAnnotations retrieves annotations from either Secret or ConfigMap
func GetResourceAnnotations(ctx context.Context, client kubernetes.Interface, name, namespace string) (map[string]string, error) {
	configName := ConfigNamePrefix + name

	// Try to get annotations from Secret first
	secret, secretErr := client.CoreV1().Secrets(namespace).Get(ctx, configName, metav1.GetOptions{})
	if secretErr == nil {
		return secret.Annotations, nil
	}

	// If Secret not found, try ConfigMap
	if kerrors.IsNotFound(secretErr) {
		configMap, cmErr := client.CoreV1().ConfigMaps(namespace).Get(ctx, configName, metav1.GetOptions{})
		if cmErr != nil {
			return nil, fmt.Errorf("failed to get configuration from either Secret or ConfigMap: Secret error: %v, ConfigMap error: %v", secretErr, cmErr)
		}

		return configMap.Annotations, nil
	}

	return nil, fmt.Errorf("get secret: %w", secretErr)
}

// UpdateAnnotations is a generic helper that sets the distro and store type annotations
func UpdateAnnotations(annotations *map[string]string, distro string, backingStoreType string) bool {
	if *annotations == nil {
		*annotations = map[string]string{}
	}

	// Check if updates are needed
	if (*annotations)[AnnotationDistro] == distro && (*annotations)[AnnotationStore] == backingStoreType {
		return false // No changes needed
	}

	// Update the annotations
	(*annotations)[AnnotationDistro] = distro
	(*annotations)[AnnotationStore] = backingStoreType
	return true // Changes were made
}

// GetVClusterConfigResource retrieves the data content from either the vCluster config Secret or ConfigMap
func GetVClusterConfigResource(ctx context.Context, clientset kubernetes.Interface, name, namespace string) ([]byte, error) {
	configName := ConfigNamePrefix + name

	// Try Secret first
	secret, secretErr := clientset.CoreV1().Secrets(namespace).Get(ctx, configName, metav1.GetOptions{})
	if secretErr == nil {
		configBytes, ok := secret.Data[ConfigFileName]
		if !ok {
			return nil, fmt.Errorf("secret %s in namespace %s does not contain the expected %s field", configName, namespace, ConfigFileName)
		}
		return configBytes, nil
	}

	if kerrors.IsNotFound(secretErr) {
		// Try ConfigMap if Secret not found
		configMap, cmErr := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, configName, metav1.GetOptions{})
		if cmErr != nil {
			return nil, fmt.Errorf("failed to get configuration from either Secret or ConfigMap: Secret error: %v, ConfigMap error: %v", secretErr, cmErr)
		}

		configYaml, ok := configMap.Data[ConfigFileName]
		if !ok {
			return nil, fmt.Errorf("configMap %s in namespace %s does not contain the expected %s field", configName, namespace, ConfigFileName)
		}

		return []byte(configYaml), nil
	}

	return nil, secretErr
}
