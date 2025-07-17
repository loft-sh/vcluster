package cli

import (
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/setup"
	"k8s.io/client-go/tools/clientcmd"
)

type DescribeOutput struct {
	Distro       string   `json:"distro"`
	Version      string   `json:"version"`
	BackingStore string   `json:"backingStore"`
	ImageTags    ImageTag `json:"imageTags"`
}

type ImageTag struct {
	APIServer string `json:"apiServer,omitempty"`
	Syncer    string `json:"syncer,omitempty"`
}

func DescribeHelm(ctx context.Context, flags *flags.GlobalFlags, output io.Writer, name, format string) error {
	namespace := "vcluster-" + name
	if flags.Namespace != "" {
		namespace = flags.Namespace
	}

	kConf := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})
	rawConfig, err := kConf.RawConfig()
	if err != nil {
		return err
	}

	vclusterConfig, err := setup.GetVClusterConfig(ctx, kConf, name, namespace)
	if err != nil {
		return err
	}

	// Convert config to bytes for YAML/JSON output formats
	configBytes, err := yaml.Marshal(vclusterConfig)
	if err != nil {
		return err
	}

	switch format {
	case "yaml":
		_, err = output.Write(configBytes)
		return err
	case "json":
		b, err := yaml.YAMLToJSON(configBytes)
		if err != nil {
			return err
		}
		_, err = output.Write(b)
		return err
	}
	fVcluster, err := find.GetVCluster(ctx, rawConfig.CurrentContext, name, namespace, log.Discard)
	if err != nil {
		return err
	}

	describeOutput := &DescribeOutput{}
	err = extractFromValues(describeOutput, configBytes, vclusterConfig, format, fVcluster.Version, output)
	if err != nil {
		return err
	}

	describeOutput.Version = fVcluster.Version

	b, err := yaml.Marshal(describeOutput)
	if err != nil {
		return err
	}
	_, err = output.Write(b)

	return err
}

func extractFromValues(d *DescribeOutput, configBytes []byte, conf *config.Config, format, version string, output io.Writer) error {

	switch format {
	case "yaml":
		_, err := output.Write(configBytes)
		return err
	case "json":
		err := json.NewEncoder(output).Encode(conf)
		if err != nil {
			return err
		}
	default:
		d.Distro = conf.Distro()
		d.BackingStore = string(conf.BackingStoreType())
		syncer, api := getImageTags(conf, version)
		d.ImageTags = ImageTag{
			Syncer:    syncer,
			APIServer: api,
		}
	}

	return nil
}

func valueOrDefaultRegistry(value, def string) string {
	if value != "" {
		return value
	}
	if def != "" {
		return def
	}
	return "ghcr.io/loft-sh"
}

func valueOrDefaultSyncerImage(value string) string {
	if value != "" {
		return value
	}
	return "vcluster-pro"
}

func getImageTags(c *config.Config, version string) (syncer, api string) {
	syncerConfig := c.ControlPlane.StatefulSet.Image
	defaultRegistry := c.ControlPlane.Advanced.DefaultImageRegistry

	syncer = valueOrDefaultRegistry(syncerConfig.Registry, defaultRegistry) + "/" + valueOrDefaultSyncerImage(syncerConfig.Repository) + ":" + syncerConfig.Tag
	if syncerConfig.Tag == "" {
		// the chart uses the chart version for the syncer tag, so the tag isn't set by default
		syncer += version
	}

	switch c.Distro() {
	case config.K8SDistro:
		k8s := c.ControlPlane.Distro.K8S

		api = valueOrDefaultRegistry(k8s.Image.Registry, defaultRegistry) + "/" + k8s.Image.Repository + ":" + k8s.Image.Tag
		if k8s.Image.Repository == "" {
			// with the platform driver if only the registry is set we won't be able to display complete info
			api = ""
		}

	case config.K3SDistro:
		k3s := c.ControlPlane.Distro.K3S

		api = valueOrDefaultRegistry(k3s.Image.Registry, defaultRegistry) + "/" + k3s.Image.Repository + ":" + k3s.Image.Tag
		if strings.HasPrefix(api, valueOrDefaultRegistry(k3s.Image.Registry, defaultRegistry)+"/:") {
			// with the platform driver if only the registry is set we won't be able to display complete info
			api = ""
		}
	}

	syncer = strings.TrimPrefix(syncer, "/")
	api = strings.TrimPrefix(api, "/")

	syncer = strings.TrimSuffix(syncer, ":")
	api = strings.TrimSuffix(api, ":")

	return syncer, api
}
