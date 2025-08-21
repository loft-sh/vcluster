package config

import (
	"context"
	"fmt"
	"os"

	"k8s.io/client-go/util/retry"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/k3s"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	AnnotationDistro = "vcluster.loft.sh/distro"
	AnnotationStore  = "vcluster.loft.sh/store"
)

func InitClientConfig() (*rest.Config, string, error) {
	inClusterConfig, err := ctrl.GetConfig()
	if err != nil {
		return nil, "", fmt.Errorf("getting in cluster config: %w", err)
	}

	// We increase the limits here so that we don't get any problems later on
	inClusterConfig.QPS = 40
	inClusterConfig.Burst = 80
	inClusterConfig.Timeout = 0

	// get current namespace
	currentNamespace, err := clienthelper.CurrentNamespace()
	if err != nil {
		return nil, "", err
	}

	return inClusterConfig, currentNamespace, nil
}

func InitClients(vConfig *config.VirtualClusterConfig) error {
	var err error

	// get host cluster client
	vConfig.HostClient, err = kubernetes.NewForConfig(vConfig.HostConfig)
	if err != nil {
		return err
	}

	// ensure target namespace
	vConfig.HostTargetNamespace = vConfig.Experimental.SyncSettings.TargetNamespace
	if vConfig.HostTargetNamespace == "" {
		vConfig.HostTargetNamespace = vConfig.HostNamespace
	}

	// get workload target namespace translator
	if vConfig.Sync.ToHost.Namespaces.Enabled {
		translate.Default, err = pro.GetWithSyncedNamespacesTranslator(vConfig.HostTargetNamespace, vConfig.Sync.ToHost.Namespaces.Mappings)
		if err != nil {
			return err
		}
	} else {
		translate.Default = translate.NewSingleNamespaceTranslator(vConfig.HostTargetNamespace)
	}

	return nil
}

func InitAndValidateConfig(ctx context.Context, vConfig *config.VirtualClusterConfig) error {
	// set global vCluster name
	translate.VClusterName = vConfig.Name

	// set workload namespace
	err := os.Setenv("NAMESPACE", vConfig.HostNamespace)
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
		vConfig.HostClient,
		vConfig.Name,
		vConfig.HostNamespace,
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

// EnsureBackingStoreChanges ensures that only a certain set of allowed changes to the backing store and distro occur.
func EnsureBackingStoreChanges(ctx context.Context, client kubernetes.Interface, name, namespace, distro string, backingStoreType vclusterconfig.StoreType) error {
	if ok, err := CheckUsingSecretAnnotation(ctx, client, name, namespace, distro, backingStoreType); err != nil {
		return fmt.Errorf("using secret annotations: %w", err)
	} else if ok {
		if err := updateSecretAnnotations(ctx, client, name, namespace, distro, backingStoreType); err != nil {
			return fmt.Errorf("update secret annotations: %w", err)
		}

		return nil
	}

	if ok, err := CheckUsingHeuristic(distro); err != nil {
		return fmt.Errorf("using heuristic: %w", err)
	} else if ok {
		if err := updateSecretAnnotations(ctx, client, name, namespace, distro, backingStoreType); err != nil {
			return fmt.Errorf("update secret annotations: %w", err)
		}

		return nil
	}

	return nil
}

// CheckUsingHeuristic checks for known file path indicating the existence of a previous distro.
//
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

// CheckUsingSecretAnnotation checks for backend store and distro changes using annotations on the vCluster's secret annotations.
// Returns true, if both annotations are set and the check was successful, otherwise false.
func CheckUsingSecretAnnotation(ctx context.Context, client kubernetes.Interface, name, namespace, distro string, backingStoreType vclusterconfig.StoreType) (bool, error) {
	secret, err := client.CoreV1().Secrets(namespace).Get(ctx, "vc-config-"+name, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("get secret: %w", err)
	}

	if secret.Annotations == nil {
		secret.Annotations = map[string]string{}
	}

	// (ThomasK33): If we already have an annotation set, we're dealing with an upgrade.
	// Thus we can check if the distro has changed.
	okCounter := 0
	if annotatedDistro, ok := secret.Annotations[AnnotationDistro]; ok {
		if err := vclusterconfig.ValidateDistroChanges(distro, annotatedDistro); err != nil {
			return false, err
		}

		okCounter++
	}

	if annotatedStore, ok := secret.Annotations[AnnotationStore]; ok {
		if err := vclusterconfig.ValidateStoreChanges(backingStoreType, vclusterconfig.StoreType(annotatedStore)); err != nil {
			return false, err
		}

		okCounter++
	}

	return okCounter == 2, nil
}

// updateSecretAnnotations udates the vCluster's config secret with the currently used distro and backing store type.
func updateSecretAnnotations(ctx context.Context, client kubernetes.Interface, name, namespace, distro string, backingStoreType vclusterconfig.StoreType) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		secret, err := client.CoreV1().Secrets(namespace).Get(ctx, "vc-config-"+name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("get secret: %w", err)
		}

		if secret.Annotations == nil {
			secret.Annotations = map[string]string{}
		}
		if secret.Annotations[AnnotationDistro] == distro && secret.Annotations[AnnotationStore] == string(backingStoreType) {
			return nil
		}

		secret.Annotations[AnnotationDistro] = distro
		secret.Annotations[AnnotationStore] = string(backingStoreType)

		if _, err := client.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("update secret: %w", err)
		}

		return nil
	})
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

	if vConfig.HostNamespace != vConfig.HostTargetNamespace {
		klog.Warningf("Skip setting owner, because current namespace %s != target namespace %s", vConfig.HostNamespace, vConfig.HostTargetNamespace)

		return nil
	}

	if vConfig.HostClient == nil {
		return errors.New("nil HostClient")
	}

	service, err := vConfig.HostClient.CoreV1().Services(vConfig.HostNamespace).Get(ctx, vConfig.Name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "get vcluster service")
	}
	// client doesn't populate typemeta sometimes
	service.APIVersion = "v1"
	service.Kind = "Service"
	translate.Owner = service

	return nil
}
