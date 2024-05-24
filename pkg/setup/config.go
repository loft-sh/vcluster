package setup

import (
	"context"
	"fmt"
	"os"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/config/legacyconfig"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/loft-sh/vcluster/pkg/k3s"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kblabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

const (
	AnnotationDistro = "vcluster.loft.sh/distro"
	AnnotationStore  = "vcluster.loft.sh/store"
)

func InitAndValidateConfig(ctx context.Context, vConfig *config.VirtualClusterConfig) error {
	var err error

	// set global vCluster name
	translate.VClusterName = vConfig.Name

	// set workload namespace
	err = os.Setenv("NAMESPACE", vConfig.WorkloadNamespace)
	if err != nil {
		return fmt.Errorf("set NAMESPACE env var: %w", err)
	}

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

	// get workload target namespace
	if vConfig.Experimental.MultiNamespaceMode.Enabled {
		translate.Default = translate.NewMultiNamespaceTranslator(vConfig.WorkloadNamespace)
	} else {
		// ensure target namespace
		vConfig.WorkloadTargetNamespace = vConfig.Experimental.SyncSettings.TargetNamespace
		if vConfig.WorkloadTargetNamespace == "" {
			vConfig.WorkloadTargetNamespace = vConfig.WorkloadNamespace
		}

		translate.Default = translate.NewSingleNamespaceTranslator(vConfig.WorkloadTargetNamespace)
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

	return nil
}

// EnsureBackingStoreChanges ensures that only a certain set of allowed changes to the backing store and distro occur.
func EnsureBackingStoreChanges(ctx context.Context, client kubernetes.Interface, name, namespace, distro string, backingStoreType vclusterconfig.StoreType) error {
	if ok, err := CheckUsingHelm(ctx, client, name, namespace, distro, backingStoreType); err != nil {
		return err
	} else if ok {
		return nil
	}

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

// CheckUsingHelm fetches the previous release revision and its computed values, and then reconstructs the distro and storage settings.
func CheckUsingHelm(ctx context.Context, client kubernetes.Interface, name, namespace, distro string, backingStoreType vclusterconfig.StoreType) (bool, error) {
	ls := kblabels.Set{}
	ls["name"] = name
	releases, err := helm.NewSecrets(client).ListUnfiltered(ctx, ls.AsSelector(), namespace)
	if err != nil || len(releases) == 0 {
		return false, nil
	}

	// (ThomasK33): if there is only one revision, we're dealing with an initial installation
	// at which point we can just exit
	if len(releases) == 1 {
		return true, nil
	}

	// We need to check if we can deserialize the existing values into multiple kind of config structs (legacy and current ones)
	previousRelease := releases[len(releases)-2]
	if previousRelease.Config == nil {
		return false, nil
	}

	// marshal previous release config
	previousConfigRaw, err := yaml.Marshal(previousRelease.Config)
	if err != nil {
		return false, nil
	}

	// Try parsing as 0.20 values
	if success, err := func() (bool, error) {
		previousConfig := vclusterconfig.Config{}
		if err := previousConfig.UnmarshalYAMLStrict(previousConfigRaw); err != nil {
			return false, nil
		}

		if err := vclusterconfig.ValidateStoreAndDistroChanges(
			backingStoreType,
			previousConfig.BackingStoreType(),
			distro,
			previousConfig.Distro(),
		); err != nil {
			return false, err
		}

		return true, nil
	}(); err != nil {
		return false, err
	} else if success {
		return true, nil
	}

	// Try parsing as < 0.20 values
	var previousStoreType vclusterconfig.StoreType
	previousDistro := ""

	switch previousRelease.Chart.Metadata.Name {
	case "vcluster-k8s":
		previousDistro = vclusterconfig.K8SDistro
	case "vcluster-eks":
		previousDistro = vclusterconfig.EKSDistro
	case "vcluster-k0s":
		previousDistro = vclusterconfig.K0SDistro
	case "vcluster":
		previousDistro = vclusterconfig.K3SDistro
	default:
		// unknown chart, we should exit here
		return true, nil
	}

	switch previousDistro {
	// handles k8s and eks values
	case vclusterconfig.K8SDistro, vclusterconfig.EKSDistro:
		previousConfig := legacyconfig.LegacyK8s{}
		if err := yaml.Unmarshal(previousConfigRaw, &previousConfig); err != nil {
			return false, err
		}

		if previousConfig.EmbeddedEtcd.Enabled {
			previousStoreType = vclusterconfig.StoreTypeEmbeddedEtcd
		} else {
			previousStoreType = vclusterconfig.StoreTypeExternalEtcd
		}

	// handles k0s and k3s values
	default:
		previousConfig := legacyconfig.LegacyK0sAndK3s{}
		if err := yaml.Unmarshal(previousConfigRaw, &previousConfig); err != nil {
			return false, err
		}

		if previousConfig.EmbeddedEtcd.Enabled {
			previousStoreType = vclusterconfig.StoreTypeEmbeddedEtcd
		} else {
			previousStoreType = vclusterconfig.StoreTypeEmbeddedDatabase
		}
	}

	if err := vclusterconfig.ValidateStoreAndDistroChanges(backingStoreType, previousStoreType, distro, previousDistro); err != nil {
		return false, err
	}

	return true, nil
}

// CheckUsingHeuristic checks for known file path indicating the existence of a previous distro.
//
// It checks for the existence of the default K3s token path or the K0s data directory.
func CheckUsingHeuristic(distro string) (bool, error) {
	// check if previously we were using k3s as a default and now have switched to a different distro
	if distro != vclusterconfig.K3SDistro {
		_, err := os.Stat(k3s.TokenPath)
		if err == nil {
			return false, fmt.Errorf("seems like you were using k3s as a distro before and now have switched to %s, please make sure to not switch between vCluster distros", distro)
		}
	}

	// check if previously we were using k0s as distro
	if distro != vclusterconfig.K0SDistro {
		_, err := os.Stat("/data/k0s")
		if err == nil {
			return false, fmt.Errorf("seems like you were using k0s as a distro before and now have switched to %s, please make sure to not switch between vCluster distros", distro)
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
		if err := vclusterconfig.ValidateStoreAndDistroChanges("", "", distro, annotatedDistro); err != nil {
			return false, err
		}

		okCounter++
	}

	if annotatedStore, ok := secret.Annotations[AnnotationStore]; ok {
		if err := vclusterconfig.ValidateStoreAndDistroChanges(backingStoreType, vclusterconfig.StoreType(annotatedStore), "", ""); err != nil {
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
