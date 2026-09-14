package cli

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/helm"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/describe"
	"k8s.io/utils/ptr"
)

var (
	defaultRegistry         = "ghcr.io/loft-sh"
	defaultSyncerRepository = "vcluster-pro"
)

type DescribeOutput struct {
	Name           string            `json:"name,omitempty"`
	Namespace      string            `json:"namespace,omitempty"`
	Version        string            `json:"version,omitempty"`
	BackingStore   string            `json:"backingStore,omitempty"`
	Status         string            `json:"status,omitempty"`
	Created        metav1.Time       `json:"created,omitempty"`
	Images         map[string]string `json:"imageTags,omitempty"`
	UserConfigYaml *string           `json:"userConfigYaml,omitempty"`
}

func (do *DescribeOutput) String() string {
	// used tabbedString from k8s.io/kubectl/pkg/describe/describe.go as inspiration
	out := &tabwriter.Writer{}
	buf := &bytes.Buffer{}
	out.Init(buf, 0, 8, 2, ' ', 0)

	w := describe.NewPrefixWriter(out)
	w.Write(describe.LEVEL_0, "Name:\t%s\n", do.Name)
	w.Write(describe.LEVEL_0, "Namespace:\t%s\n", do.Namespace)
	w.Write(describe.LEVEL_0, "Version:\t%s\n", do.Version)
	w.Write(describe.LEVEL_0, "Backing Store:\t%s\n", do.BackingStore)
	w.Write(describe.LEVEL_0, "Created:\t%s\n", do.Created.Time.Format(time.RFC1123Z))
	w.Write(describe.LEVEL_0, "Status:\t%s\n", do.Status)

	if len(do.Images) > 0 {
		w.Write(describe.LEVEL_0, "Images:\n")
		for _, name := range slices.Sorted(maps.Keys(do.Images)) {
			w.Write(describe.LEVEL_1, "%s:\t%s\n", name, do.Images[name])
		}
	}

	if do.UserConfigYaml != nil {
		userConfigYaml, isTruncated := truncateString(*do.UserConfigYaml, "\n", 50)
		w.Write(describe.LEVEL_0, "\n------------------- vcluster.yaml -------------------\n")
		w.Write(describe.LEVEL_0, "%s\n", strings.TrimSuffix(userConfigYaml, "\n"))
		if isTruncated {
			w.Write(describe.LEVEL_0, "... (truncated)\n")
		}
		w.Write(describe.LEVEL_0, "-----------------------------------------------------\n")
		if isTruncated {
			w.Write(describe.LEVEL_0, "Use --config-only to retrieve the full vcluster.yaml only\n")
		} else {
			w.Write(describe.LEVEL_0, "Use --config-only to retrieve just the vcluster.yaml\n")
		}
	}

	out.Flush()
	return buf.String()
}

func DescribeHelm(ctx context.Context, flags *flags.GlobalFlags, output io.Writer, l log.Logger, name string, configOnly bool, format string) error {
	kubeClientConf := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})
	rawConfig, err := kubeClientConf.RawConfig()
	if err != nil {
		return err
	}
	kubeConfig, err := kubeClientConf.ClientConfig()
	if err != nil {
		return err
	}

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}

	vCluster, err := find.GetVCluster(ctx, rawConfig.CurrentContext, name, flags.Namespace, l)
	if err != nil {
		return err
	}

	configSecret, err := kubeClient.CoreV1().Secrets(vCluster.Namespace).Get(ctx, "vc-config-"+vCluster.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to load the vcluster config: %w", err)
	}

	// Log ArgoCD tracking id
	if trackingID, ok := configSecret.Annotations["argocd.argoproj.io/tracking-id"]; ok {
		l.Infof("The %s vcluster is managed via ArgoCD. argocd.argoproj.io/tracking-id: %s.", vCluster.Name, trackingID)
	}

	// Load the user supplied vcluster.yaml from the HelmRelease Config field
	helmRelease, err := helm.NewSecrets(kubeClient).Get(ctx, vCluster.Name, vCluster.Namespace)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return fmt.Errorf("failed to load the user supplied vcluster.yaml: %w", err)
		}
	}

	var userConfigYaml *string
	if helmRelease != nil {
		userConfigBytes, err := yaml.Marshal(helmRelease.Config)
		if err != nil {
			return err
		}

		userConfigYaml = ptr.To(string(userConfigBytes))
	}

	// Return only the user supplied vcluster.yaml, if configOnly is set
	if configOnly {
		if userConfigYaml == nil {
			return fmt.Errorf("failed to load vcluster config")
		}

		if _, err := output.Write([]byte(*userConfigYaml)); err != nil {
			return err
		}

		return nil
	}

	configBytes, ok := configSecret.Data["config.yaml"]
	if !ok {
		return fmt.Errorf("configSecret %s in namespace %s does not contain the expected 'config.yaml' field", configSecret.Name, configSecret.Namespace)
	}

	conf, err := configPartialUnmarshal(configBytes)
	if err != nil {
		return err
	}

	if userConfigYaml == nil {
		l.Warnf("User supplied vcluster.yaml is not available")
	}

	describeOutput := &DescribeOutput{
		Name:           vCluster.Name,
		Namespace:      vCluster.Namespace,
		Created:        vCluster.Created,
		Status:         string(vCluster.Status),
		Version:        vCluster.Version,
		BackingStore:   string(conf.BackingStoreType()),
		Images:         getImagesFromConfig(conf, vCluster.Version),
		UserConfigYaml: userConfigYaml,
	}

	return writeWithFormat(output, format, describeOutput)
}

func getImagesFromConfig(c *config.Config, version string) map[string]string {
	result := make(map[string]string)

	registryOverride := c.ControlPlane.Advanced.DefaultImageRegistry

	syncerFromConfig := c.ControlPlane.StatefulSet.Image
	syncer := config.Image{
		Registry:   cmp.Or(syncerFromConfig.Registry, registryOverride, defaultRegistry),
		Repository: cmp.Or(syncerFromConfig.Repository, defaultSyncerRepository),
		// the chart uses the chart version for the syncerRef tag, so the tag isn't set by default
		Tag: cmp.Or(syncerFromConfig.Tag, version),
	}
	syncerRef := syncer.String()
	if syncerRef != "" {
		result["syncer"] = syncerRef
	}

	apiFromConfig := c.ControlPlane.Distro.K8S.Image

	// with the platform driver if only the registry is set we won't be able to display complete info
	if apiFromConfig.Repository != "" {
		api := config.Image{
			Registry:   cmp.Or(apiFromConfig.Registry, registryOverride, defaultRegistry),
			Repository: apiFromConfig.Repository,
			Tag:        apiFromConfig.Tag,
		}
		apiRef := api.String()
		if apiRef != "" {
			result["apiServer"] = apiRef
		}
	}

	return result
}

// configPartialUnmarshal attempts to unmarshal only the relevant section
// of the config to avoid potential version mismatch error.
func configPartialUnmarshal(configBytes []byte) (*config.Config, error) {
	var partialConfig struct {
		ControlPlane struct {
			Advanced struct {
				DefaultImageRegistry string `json:"defaultImageRegistry"`
			} `json:"advanced,omitempty"`
			BackingStore config.BackingStore `json:"backingStore,omitempty"`
			Distro       config.Distro       `json:"distro,omitempty"`
			StatefulSet  struct {
				Image config.Image `json:"image"`
			} `json:"statefulSet,omitempty"`
		} `json:"controlPlane,omitempty"`
	}

	if err := yaml.Unmarshal(configBytes, &partialConfig); err != nil {
		return nil, err
	}

	return &config.Config{
		ControlPlane: config.ControlPlane{
			Advanced: config.ControlPlaneAdvanced{
				DefaultImageRegistry: partialConfig.ControlPlane.Advanced.DefaultImageRegistry,
			},
			BackingStore: partialConfig.ControlPlane.BackingStore,
			Distro:       partialConfig.ControlPlane.Distro,
			StatefulSet: config.ControlPlaneStatefulSet{
				Image: partialConfig.ControlPlane.StatefulSet.Image,
			},
		},
	}, nil
}

func marshalWithFormat(o fmt.Stringer, format string) ([]byte, error) {
	switch format {
	case "json":
		return json.Marshal(o)
	case "yaml":
		return yaml.Marshal(o)
	case "":
		return []byte(o.String()), nil
	}

	return nil, fmt.Errorf("unknown format %s", format)
}

func writeWithFormat(writer io.Writer, format string, o fmt.Stringer) error {
	outputBytes, err := marshalWithFormat(o, format)
	if err != nil {
		return err
	}

	_, err = writer.Write(outputBytes)
	return err
}

func truncateString(s string, sep string, max int) (string, bool) {
	lines := strings.SplitN(s, sep, max+1)
	count := len(lines)

	if count <= max {
		return s, false
	}

	return strings.Join(lines[:max], sep), true
}
