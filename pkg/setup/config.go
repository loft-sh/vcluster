package setup

import (
	"context"
	"fmt"
	"os"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/util/translate"
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

	// (ThomasK33): Below we read the secret containing the config and look for annotations inidicating the initial distro or backing store.
	if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		secret, err := vConfig.ControlPlaneClient.CoreV1().Secrets(vConfig.ControlPlaneNamespace).Get(ctx, "vc-config-"+vConfig.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("get secret: %w", err)
		}

		if secret.Annotations == nil {
			secret.Annotations = map[string]string{}
		}

		// (ThomasK33): If we already have an annotation set, we're dealing with an upgrade.
		// Thus we can check if the distro has changed.
		distro := vConfig.Distro()

		if annotatedDistro, ok := secret.Annotations[AnnotationDistro]; ok {
			if distro != annotatedDistro {
				return fmt.Errorf("seems like you were using %s as a distro before and now have switched to %s, please make sure to not switch between vCluster distros", annotatedDistro, vConfig.Distro())
			}
		} else {
			// Otherwise we're dealing with a fresh start, and we can just set the initial used distro.
			secret.Annotations[AnnotationDistro] = distro

			if secret, err = vConfig.ControlPlaneClient.CoreV1().Secrets(vConfig.ControlPlaneNamespace).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
				return fmt.Errorf("update secret: %w", err)
			}
		}

		backingStoreType := vConfig.BackingStoreType()

		if annotatedStore, ok := secret.Annotations[AnnotationStore]; ok {
			if string(backingStoreType) != annotatedStore {
				return fmt.Errorf("seems like you were using %s as a store before and now have switched to %s, please make sure to not switch between vCluster stores", annotatedStore, vConfig.BackingStoreType())
			}
		} else {
			// Otherwise we're dealing with a fresh start, and we can just set the initial used store.
			secret.Annotations[AnnotationStore] = string(backingStoreType)

			if _, err := vConfig.ControlPlaneClient.CoreV1().Secrets(vConfig.ControlPlaneNamespace).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
				return fmt.Errorf("update secret: %w", err)
			}
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}
