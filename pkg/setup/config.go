package setup

import (
	"context"
	"fmt"
	"k8s.io/client-go/tools/clientcmd"
	"os"

	"k8s.io/client-go/util/retry"

	"github.com/ghodss/yaml"
	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/k3s"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/util/confighelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

func InitClients(vConfig *config.VirtualClusterConfig) error {
	var err error

	// get host cluster client
	vConfig.ControlPlaneClient, err = kubernetes.NewForConfig(vConfig.ControlPlaneConfig)
	if err != nil {
		return err
	}

	// get workload client
	vConfig.WorkloadClient, err = kubernetes.NewForConfig(vConfig.WorkloadConfig)
	if err != nil {
		return err
	}

	// ensure target namespace
	vConfig.WorkloadTargetNamespace = vConfig.Experimental.SyncSettings.TargetNamespace
	if vConfig.WorkloadTargetNamespace == "" {
		vConfig.WorkloadTargetNamespace = vConfig.WorkloadNamespace
	}

	// get workload target namespace translator
	if vConfig.Sync.ToHost.Namespaces.Enabled {
		translate.Default, err = pro.GetWithSyncedNamespacesTranslator(vConfig.WorkloadTargetNamespace, vConfig.Sync.ToHost.Namespaces.Mappings)
		if err != nil {
			return err
		}
	} else {
		translate.Default = translate.NewSingleNamespaceTranslator(vConfig.WorkloadTargetNamespace)
	}

	return nil
}

func InitAndValidateConfig(ctx context.Context, vConfig *config.VirtualClusterConfig) error {
	// set global vCluster name
	translate.VClusterName = vConfig.Name

	// set workload namespace
	err := os.Setenv("NAMESPACE", vConfig.WorkloadNamespace)
	if err != nil {
		return fmt.Errorf("set NAMESPACE env var: %w", err)
	}

	// init clients
	err = InitClients(vConfig)
	if err != nil {
		return err
	}

	if err := EnsureBackingStoreChanges(
		ctx,
		vConfig.ControlPlaneClient,
		vConfig.Name,
		vConfig.ControlPlaneNamespace,
		vConfig.Distro(),
		vConfig.BackingStoreType(),
	); err != nil {
		return err
	}

	// set global owner for use in owner references
	err = SetGlobalOwner(
		ctx,
		vConfig,
	)
	if err != nil {
		return errors.Wrap(err, "finding vcluster pod owner")
	}

	return nil
}

// GetVClusterConfig retrieves and parses the vCluster configuration from either Secret or ConfigMap.
func GetVClusterConfig(ctx context.Context, kConf clientcmd.ClientConfig, name, namespace string) (*vclusterconfig.Config, error) {
	clientConfig, err := kConf.ClientConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return nil, err
	}

	configBytes, err := confighelper.GetVClusterConfigResource(ctx, clientset, name, namespace)
	if err != nil {
		return nil, err
	}

	return unmarshalConfig(configBytes)
}

// unmarshalConfig parses YAML config bytes into a Config object
func unmarshalConfig(configBytes []byte) (*vclusterconfig.Config, error) {
	vclusterConfig := &vclusterconfig.Config{}
	if err := yaml.Unmarshal(configBytes, vclusterConfig); err != nil {
		return nil, fmt.Errorf("failed to parse vCluster configuration: %w", err)
	}
	return vclusterConfig, nil
}

// CheckAnnotations validates the distro and store type annotations from either a Secret or ConfigMap
func CheckAnnotations(annotations map[string]string, distro string, backingStoreType vclusterconfig.StoreType) (bool, error) {
	if annotations == nil {
		annotations = map[string]string{}
	}

	// If we already have an annotation set, we're dealing with an upgrade.
	// Thus we can check if the distro has changed.
	okCounter := 0
	if annotatedDistro, ok := annotations[confighelper.AnnotationDistro]; ok {
		if err := vclusterconfig.ValidateDistroChanges(distro, annotatedDistro); err != nil {
			return false, err
		}

		okCounter++
	}

	if annotatedStore, ok := annotations[confighelper.AnnotationStore]; ok {
		if err := vclusterconfig.ValidateStoreChanges(backingStoreType, vclusterconfig.StoreType(annotatedStore)); err != nil {
			return false, err
		}

		okCounter++
	}

	return okCounter == 2, nil
}

// UpdateConfigAnnotations checks which resource (Secret or ConfigMap) exists and updates its annotations
func UpdateConfigAnnotations(ctx context.Context, client kubernetes.Interface, name, namespace, distro string, backingStoreType vclusterconfig.StoreType) error {
	configName := confighelper.ConfigNamePrefix + name

	// Try Secret first
	secret, secretErr := client.CoreV1().Secrets(namespace).Get(ctx, configName, metav1.GetOptions{})
	if secretErr == nil {
		return UpdateSecretAnnotations(ctx, client, secret, distro, backingStoreType)
	}

	// If Secret not found, try ConfigMap
	if kerrors.IsNotFound(secretErr) {
		configMap, cmErr := client.CoreV1().ConfigMaps(namespace).Get(ctx, configName, metav1.GetOptions{})
		if cmErr != nil {
			return fmt.Errorf("failed to get configuration from either Secret or ConfigMap: Secret error: %v, ConfigMap error: %v", secretErr, cmErr)
		}

		return UpdateConfigMapAnnotations(ctx, client, configMap, distro, backingStoreType)
	}

	return secretErr
}

// UpdateSecretAnnotations updates a Secret's annotations with the vCluster distro and backing store type.
func UpdateSecretAnnotations(ctx context.Context, client kubernetes.Interface, secret *corev1.Secret, distro string, backingStoreType vclusterconfig.StoreType) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		// Apply annotations and check if changes were made
		if !confighelper.UpdateAnnotations(&secret.Annotations, distro, string(backingStoreType)) {
			return nil // No changes needed
		}

		// Update the Secret if changes were made
		if _, err := client.CoreV1().Secrets(secret.Namespace).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("update secret: %w", err)
		}

		return nil
	})
}

// UpdateConfigMapAnnotations updates a ConfigMap's annotations with the vCluster distro and backing store type.
func UpdateConfigMapAnnotations(ctx context.Context, client kubernetes.Interface, configMap *corev1.ConfigMap, distro string, backingStoreType vclusterconfig.StoreType) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		// Apply annotations and check if changes were made
		if !confighelper.UpdateAnnotations(&configMap.Annotations, distro, string(backingStoreType)) {
			return nil // No changes needed
		}

		// Update the ConfigMap if changes were made
		if _, err := client.CoreV1().ConfigMaps(configMap.Namespace).Update(ctx, configMap, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("update configmap: %w", err)
		}

		return nil
	})
}

// EnsureBackingStoreChanges ensures that only a certain set of allowed changes to the backing store and distro occur.
// Then updates the annotations on either Secret or ConfigMap based on what exists.
func EnsureBackingStoreChanges(ctx context.Context, client kubernetes.Interface, name, namespace, distro string, backingStoreType vclusterconfig.StoreType) error {
	// First, check using existing config annotations
	if ok, err := CheckUsingConfigAnnotation(ctx, client, name, namespace, distro, backingStoreType); err != nil {
		return fmt.Errorf("using config annotations: %w", err)
	} else if ok {
		// If validation successful, update the annotations
		return UpdateConfigAnnotations(ctx, client, name, namespace, distro, backingStoreType)
	}

	// If no config annotations or validation failed, try heuristic check
	if ok, err := CheckUsingHeuristic(distro); err != nil {
		return fmt.Errorf("using heuristic: %w", err)
	} else if ok {
		// If validation successful, update the annotations
		return UpdateConfigAnnotations(ctx, client, name, namespace, distro, backingStoreType)
	}

	return nil
}

// SetGlobalOwner fetches the owning service and populates in translate.Owner if: the vcluster is configured to setOwner is,
// and if the currentNamespace == targetNamespace (because cross namespace owner refs don't work).
func SetGlobalOwner(ctx context.Context, vConfig *config.VirtualClusterConfig) error {
	if vConfig == nil {
		return errors.New("nil vConfig")
	}

	if !vConfig.Experimental.SyncSettings.SetOwner {
		return nil
	}

	if vConfig.Sync.ToHost.Namespaces.Enabled {
		klog.Warningf("Skip setting owner, because multi namespace mode is enabled")
		return nil
	}

	if vConfig.WorkloadNamespace != vConfig.WorkloadTargetNamespace {
		klog.Warningf("Skip setting owner, because current namespace %s != target namespace %s", vConfig.WorkloadNamespace, vConfig.WorkloadTargetNamespace)

		return nil
	}

	if vConfig.WorkloadClient == nil {
		return errors.New("nil WorkloadClient")
	}

	service, err := vConfig.WorkloadClient.CoreV1().Services(vConfig.WorkloadNamespace).Get(ctx, vConfig.WorkloadService, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "get vcluster service")
	}
	// client doesn't populate typemeta sometimes
	service.APIVersion = "v1"
	service.Kind = "Service"
	translate.Owner = service

	return nil
}

// CheckUsingHeuristic checks for known file path indicating the existence of a previous distro.
// It checks for the existence of the default K3s token path.
func CheckUsingHeuristic(distro string) (bool, error) {
	// check if previously we were using k3s as a default and now have switched to a different distro
	if distro != vclusterconfig.K3SDistro && distro != vclusterconfig.K8SDistro {
		_, err := os.Stat(k3s.TokenPath)
		if err == nil {
			return false, fmt.Errorf("seems like you were using k3s as a distro before and now have switched to %s, please make sure to not switch between vCluster distros", distro)
		}
	}

	return true, nil
}

// CheckUsingConfigAnnotation checks for backend store and distro changes using annotations on the vCluster's configuration resource (Secret or ConfigMap).
// Returns true, if both annotations are set and the check was successful, otherwise false.
func CheckUsingConfigAnnotation(ctx context.Context, client kubernetes.Interface, name, namespace, distro string, backingStoreType vclusterconfig.StoreType) (bool, error) {
	annotations, err := confighelper.GetResourceAnnotations(ctx, client, name, namespace)
	if err != nil {
		return false, err
	}

	return CheckAnnotations(annotations, distro, backingStoreType)
}
