package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type DescribeOutput struct {
	Distro       string   `json:"distro"`
	Version      string   `json:"version"`
	BackingStore string   `json:"backingStore"`
	ImageTags    ImageTag `json:"imageTags"`
}

type ImageTag struct {
	APIServer         string `json:"apiServer,omitempty"`
	Syncer            string `json:"syncer,omitempty"`
	Scheduler         string `json:"scheduler,omitempty"`
	ControllerManager string `json:"controllerManager,omitempty"`
}

func DescribeHelm(ctx context.Context, flags *flags.GlobalFlags, output io.Writer, name, format string) error {
	namespace := "vcluster-" + name
	if flags.Namespace != "" {
		namespace = flags.Namespace
	}

	secretName := "vc-config-" + name

	kConf := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})
	rawConfig, err := kConf.RawConfig()
	if err != nil {
		return err
	}
	clientConfig, err := kConf.ClientConfig()
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return err
	}

	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, v1.GetOptions{})
	if err != nil {
		return err
	}

	configBytes, ok := secret.Data["config.yaml"]
	if !ok {
		return fmt.Errorf("secret %s in namespace %s does not contain the expected 'config.yaml' field", secretName, namespace)
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
	err = extractFromValues(describeOutput, configBytes, format, fVcluster.Version, output)
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

func extractFromValues(d *DescribeOutput, configBytes []byte, format, version string, output io.Writer) error {
	conf := &config.Config{}
	err := yaml.Unmarshal(configBytes, conf)
	if err != nil {
		return err
	}

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
		syncer, api, scheduler, controllerManager := getImageTags(conf, version)
		d.ImageTags = ImageTag{
			Syncer:            syncer,
			APIServer:         api,
			Scheduler:         scheduler,
			ControllerManager: controllerManager,
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

func getImageTags(c *config.Config, version string) (syncer, api, scheduler, controllerManager string) {
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

		api = valueOrDefaultRegistry(k8s.APIServer.Image.Registry, defaultRegistry) + "/" + k8s.APIServer.Image.Repository + ":" + k8s.APIServer.Image.Tag
		if k8s.APIServer.Image.Repository == "" {
			// with the platform driver if only the registry is set we won't be able to display complete info
			api = ""
		}

		scheduler = valueOrDefaultRegistry(k8s.Scheduler.Image.Registry, defaultRegistry) + "/" + k8s.Scheduler.Image.Repository + ":" + k8s.Scheduler.Image.Tag
		if k8s.Scheduler.Image.Repository == "" {
			// with the platform driver if only the registry is set we won't be able to display complete info
			scheduler = ""
		}

		controllerManager = valueOrDefaultRegistry(k8s.ControllerManager.Image.Registry, defaultRegistry) + "/" + k8s.ControllerManager.Image.Repository + ":" + k8s.ControllerManager.Image.Tag
		if k8s.ControllerManager.Image.Repository == "" {
			// with the platform driver if only the registry is set we won't be able to display complete info
			controllerManager = ""
		}

	case config.K3SDistro:
		k3s := c.ControlPlane.Distro.K3S

		api = valueOrDefaultRegistry(k3s.Image.Registry, defaultRegistry) + "/" + k3s.Image.Repository + ":" + k3s.Image.Tag
		if strings.HasPrefix(api, valueOrDefaultRegistry(k3s.Image.Registry, defaultRegistry)+"/:") {
			// with the platform driver if only the registry is set we won't be able to display complete info
			api = ""
		}
	case config.K0SDistro:
		k0s := c.ControlPlane.Distro.K0S

		api = valueOrDefaultRegistry(k0s.Image.Registry, defaultRegistry) + "/" + k0s.Image.Repository + ":" + k0s.Image.Tag

		if strings.HasPrefix(api, valueOrDefaultRegistry(k0s.Image.Registry, defaultRegistry)+"/:") {
			// with the platform driver if only the registry is set we won't be able to display complete info
			api = ""
		}
	}

	syncer = strings.TrimPrefix(syncer, "/")
	api = strings.TrimPrefix(api, "/")
	scheduler = strings.TrimPrefix(scheduler, "/")
	controllerManager = strings.TrimPrefix(controllerManager, "/")

	syncer = strings.TrimSuffix(syncer, ":")
	api = strings.TrimSuffix(api, ":")
	scheduler = strings.TrimSuffix(scheduler, ":")
	controllerManager = strings.TrimSuffix(controllerManager, ":")

	return syncer, api, scheduler, controllerManager
}
