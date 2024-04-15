package setup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd"
	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/config/legacyconfig"
	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/loft-sh/vcluster/pkg/k3s"
	"github.com/loft-sh/vcluster/pkg/util/kubeconfig"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"
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

	// construct in-cluster helm client
	inClusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("in cluster config: %w", err)
	}

	inClusterKubeconfig, err := kubeconfig.ConvertRestConfigToClientConfig(inClusterConfig)
	if err != nil {
		return fmt.Errorf("convert rest config to client config: %w", err)
	}

	inClusterRaw, err := inClusterKubeconfig.RawConfig()
	if err != nil {
		return fmt.Errorf("raw config: %w", err)
	}

	helmBinaryPath, err := cmd.GetHelmBinaryPath(ctx, log.GetInstance())
	if err != nil {
		return fmt.Errorf("helm binary patch: %w", err)
	}

	helm := &helmValuesWrapper{
		Client:   helm.NewClient(&inClusterRaw, log.GetInstance(), helmBinaryPath),
		config:   &inClusterRaw,
		log:      log.GetInstance(),
		helmPath: helmBinaryPath,
	}

	if err := EnsureBackingStoreChanges(
		ctx,
		vConfig.ControlPlaneClient,
		helm,
		vConfig.Name,
		vConfig.ControlPlaneNamespace,
		vConfig.Distro(),
		vConfig.BackingStoreType(),
	); err != nil {
		return err
	}

	return nil
}

type helmValuesWrapper struct {
	helm.Client

	config   *api.Config
	log      log.Logger
	helmPath string
}

func (h *helmValuesWrapper) GetValues(ctx context.Context, releasename string, namespace string, revision int) ([]byte, error) {
	kubeConfig, err := helm.WriteKubeConfig(h.config)
	if err != nil {
		return nil, err
	}
	defer os.Remove(kubeConfig)

	args := []string{"get", "values", releasename, "--namespace", namespace, "--kubeconfig", kubeConfig, "-o", "yaml", "--revision", strconv.Itoa(revision), "--all"}
	return exec.CommandContext(ctx, h.helmPath, args...).CombinedOutput()
}

// History implements HelmValuesClient.
func (h *helmValuesWrapper) History(ctx context.Context, releasename string, namespace string) (HelmHistory, error) {
	kubeConfig, err := helm.WriteKubeConfig(h.config)
	if err != nil {
		return nil, err
	}
	defer os.Remove(kubeConfig)

	args := []string{"history", releasename, "--namespace", namespace, "--max", strconv.Itoa(2), "--kubeconfig", kubeConfig, "-o", "json"}

	output, err := exec.CommandContext(ctx, h.helmPath, args...).CombinedOutput()
	if err != nil {
		return nil, err
	}

	return UnmarshalHelmHistory(output)
}

// HelmValuesClient defines the interface how to interact with helm
type HelmValuesClient interface {
	Exists(name, namespace string) (bool, error)
	History(ctx context.Context, releasename, namespace string) (HelmHistory, error)
	// Returns all computed values from a revision in yaml
	GetValues(ctx context.Context, releasename, namespace string, revision int) ([]byte, error)
}

type HelmHistory []HelmHistoryElement

func UnmarshalHelmHistory(data []byte) (l HelmHistory, err error) {
	err = json.Unmarshal(data, &l)
	return
}

type HelmHistoryElement struct {
	Updated     string `json:"updated"`
	Status      string `json:"status"`
	Chart       string `json:"chart"`
	AppVersion  string `json:"app_version"`
	Description string `json:"description"`
	Revision    int64  `json:"revision"`
}

// EnsureBackingStoreChanges ensures that only a certain set of allowed changes to the backing store and distro occur.
func EnsureBackingStoreChanges(ctx context.Context, client kubernetes.Interface, helm HelmValuesClient, name, namespace, distro string, backingStoreType config.StoreType) error {
	if ok, err := CheckUsingHelm(ctx, helm, name, namespace, distro, backingStoreType); err != nil {
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
func CheckUsingHelm(ctx context.Context, helm HelmValuesClient, name, namespace, distro string, backingStoreType config.StoreType) (bool, error) {
	// (ThomasK33): For the case that we don't have any history elements (len(history) == 0), proceed with other checks
	if history, err := helm.History(ctx, name, namespace); err == nil && len(history) > 0 {
		// (ThomasK33): if there is only one revision, we're dealing with an initial installation
		// at which point we can just exit
		if len(history) == 1 {
			return true, nil
		}

		previousRevision, currentRevision := &history[0], &history[0]
		for _, entry := range history {
			if currentRevision.Revision < entry.Revision {
				previousRevision = currentRevision

				entry := entry
				currentRevision = &entry
			}
		}

		existingValues, err := helm.GetValues(ctx, name, namespace, int(previousRevision.Revision))
		if err != nil {
			return false, fmt.Errorf("get values: %w", err)
		}

		// We need to check if we can deserialize the existing values into multiple kind of config structs (legacy and current ones)

		// Try parsing as 0.20 values
		if success, err := func() (bool, error) {
			previousHelmValues := vclusterconfig.Config{}
			if err := previousHelmValues.DecodeYAML(bytes.NewReader(existingValues)); err != nil {
				return false, nil
			}

			previousConfig := config.VirtualClusterConfig{Config: previousHelmValues}

			if err := validateChanges(
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
		var previousStoreType config.StoreType
		previousDistro := ""

		if strings.Contains(previousRevision.Chart, vclusterconfig.K8SDistro) {
			previousDistro = vclusterconfig.K8SDistro
		} else if strings.Contains(previousRevision.Chart, vclusterconfig.EKSDistro) {
			previousDistro = vclusterconfig.EKSDistro
		} else if strings.Contains(previousRevision.Chart, vclusterconfig.K0SDistro) {
			previousDistro = vclusterconfig.K0SDistro
		} else {
			previousDistro = vclusterconfig.K3SDistro
		}

		switch previousDistro {
		// handles k8s and eks values
		case vclusterconfig.K8SDistro, vclusterconfig.EKSDistro:
			previousConfig := legacyconfig.LegacyK8s{}
			if err := yaml.Unmarshal(existingValues, &previousConfig); err != nil {
				return false, err
			}

			if previousConfig.EmbeddedEtcd.Enabled {
				previousStoreType = config.StoreTypeEmbeddedEtcd
			} else {
				previousStoreType = config.StoreTypeExternalEtcd
			}

		// handles k0s and k3s values
		default:
			previousConfig := legacyconfig.LegacyK0sAndK3s{}
			if err := yaml.Unmarshal(existingValues, &previousConfig); err != nil {
				return false, err
			}

			if previousConfig.EmbeddedEtcd.Enabled {
				previousStoreType = config.StoreTypeEmbeddedEtcd
			} else {
				previousStoreType = config.StoreTypeEmbeddedDatabase
			}
		}

		if err := validateChanges(backingStoreType, previousStoreType, distro, previousDistro); err != nil {
			return false, nil
		}

		return true, nil
	}

	return false, nil
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
func CheckUsingSecretAnnotation(ctx context.Context, client kubernetes.Interface, name, namespace, distro string, backingStoreType config.StoreType) (bool, error) {
	secret, err := client.CoreV1().Secrets(namespace).Get(ctx, "vc-config-"+name, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("get secret: %w", err)
	}

	if secret.Annotations == nil {
		secret.Annotations = map[string]string{}
	}

	okCounter := 0

	// (ThomasK33): If we already have an annotation set, we're dealing with an upgrade.
	// Thus we can check if the distro has changed.
	if annotatedDistro, ok := secret.Annotations[AnnotationDistro]; ok {
		if distro != annotatedDistro {
			return false, fmt.Errorf("seems like you were using %s as a distro before and now have switched to %s, please make sure to not switch between vCluster distros", annotatedDistro, backingStoreType)
		}

		okCounter++
	}

	if annotatedStore, ok := secret.Annotations[AnnotationStore]; ok {
		previousStoreType := config.StoreType(annotatedStore)

		if err := validateChanges(backingStoreType, previousStoreType, "", ""); err != nil {
			return false, err
		}

		okCounter++
	}

	return okCounter == 2, nil
}

// updateSecretAnnotations udates the vCluster's config secret with the currently used distro and backing store type.
func updateSecretAnnotations(ctx context.Context, client kubernetes.Interface, name, namespace, distro string, backingStoreType config.StoreType) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		secret, err := client.CoreV1().Secrets(namespace).Get(ctx, "vc-config-"+name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("get secret: %w", err)
		}

		if secret.Annotations == nil {
			secret.Annotations = map[string]string{}
		}

		secret.Annotations[AnnotationDistro] = distro
		secret.Annotations[AnnotationStore] = string(backingStoreType)

		if _, err := client.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("update secret: %w", err)
		}

		return nil
	})
}

// validateChanges checks whether migrating from one store to the other is allowed.
func validateChanges(currentStoreType, previousStoreType config.StoreType, currentDistro, previousDistro string) error {
	if currentDistro != previousDistro {
		return fmt.Errorf("seems like you were using %s as a distro before and now have switched to %s, please make sure to not switch between vCluster distros", previousDistro, currentDistro)
	}

	if currentStoreType != previousStoreType {
		if currentStoreType != config.StoreTypeEmbeddedEtcd {
			return fmt.Errorf("seems like you were using %s as a store before and now have switched to %s, please make sure to not switch between vCluster stores", previousStoreType, currentStoreType)
		}
		if previousStoreType != config.StoreTypeExternalEtcd && previousStoreType != config.StoreTypeEmbeddedDatabase {
			return fmt.Errorf("seems like you were using %s as a store before and now have switched to %s, please make sure to not switch between vCluster stores", previousStoreType, currentStoreType)
		}
	}

	return nil
}
