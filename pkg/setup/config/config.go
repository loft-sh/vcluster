package config

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"k8s.io/client-go/util/retry"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
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

	// Get QPS from environment variable or default to 40
	qpsStr := os.Getenv("VCLUSTER_PHYSICAL_CLIENT_QPS")
	qps, err := strconv.ParseFloat(qpsStr, 32)
	if err != nil || qpsStr == "" {
		qps = 40
	}
	inClusterConfig.QPS = float32(qps)

	// Get Burst from environment variable or default to 80
	burstStr := os.Getenv("VCLUSTER_PHYSICAL_CLIENT_BURST")
	burst, err := strconv.Atoi(burstStr)
	if err != nil || burstStr == "" {
		burst = 80
	}
	inClusterConfig.Burst = burst

	// Get Timeout from environment variable or default to 0
	timeoutStr := os.Getenv("VCLUSTER_PHYSICAL_CLIENT_TIMEOUT")
	timeout, err := strconv.Atoi(timeoutStr)
	if err != nil || timeoutStr == "" {
		timeout = 0
	}
	inClusterConfig.Timeout = time.Duration(timeout) * time.Second
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

	// get workload target namespace translator
	if vConfig.Sync.ToHost.Namespaces.Enabled {
		translate.Default, err = pro.GetWithSyncedNamespacesTranslator(vConfig.HostNamespace, vConfig.Sync.ToHost.Namespaces.Mappings)
		if err != nil {
			return err
		}
	} else {
		translate.Default = translate.NewSingleNamespaceTranslator(vConfig.HostNamespace)
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
func EnsureBackingStoreChanges(ctx context.Context, client kubernetes.Interface, name, namespace string, backingStoreType vclusterconfig.StoreType) error {
	if ok, err := CheckUsingSecretAnnotation(ctx, client, name, namespace, backingStoreType); err != nil {
		return fmt.Errorf("using secret annotations: %w", err)
	} else if ok {
		if err := updateSecretAnnotations(ctx, client, name, namespace, backingStoreType); err != nil {
			return fmt.Errorf("update secret annotations: %w", err)
		}

		return nil
	}

	return nil
}

// CheckUsingSecretAnnotation checks for backend store and distro changes using annotations on the vCluster's secret annotations.
// Returns true, if both annotations are set and the check was successful, otherwise false.
func CheckUsingSecretAnnotation(ctx context.Context, client kubernetes.Interface, name, namespace string, backingStoreType vclusterconfig.StoreType) (bool, error) {
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
	if annotatedStore, ok := secret.Annotations[AnnotationStore]; ok {
		if err := vclusterconfig.ValidateStoreChanges(backingStoreType, vclusterconfig.StoreType(annotatedStore)); err != nil {
			return false, err
		}

		okCounter++
	}

	return okCounter == 1, nil
}

// updateSecretAnnotations udates the vCluster's config secret with the currently used distro and backing store type.
func updateSecretAnnotations(ctx context.Context, client kubernetes.Interface, name, namespace string, backingStoreType vclusterconfig.StoreType) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		secret, err := client.CoreV1().Secrets(namespace).Get(ctx, "vc-config-"+name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("get secret: %w", err)
		}

		if secret.Annotations == nil {
			secret.Annotations = map[string]string{}
		}
		if secret.Annotations[AnnotationStore] == string(backingStoreType) {
			return nil
		}

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
