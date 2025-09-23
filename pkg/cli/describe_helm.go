package cli

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	defaultRegistry         = "ghcr.io/loft-sh"
	defaultSyncerRepository = "vcluster-pro"
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
		syncer, api := getImageTags(conf, version)
		d.ImageTags = ImageTag{
			Syncer:    syncer,
			APIServer: api,
		}
	}

	return nil
}

func getImageTags(c *config.Config, version string) (syncer, api string) {
	registryOverride := c.ControlPlane.Advanced.DefaultImageRegistry

	syncerImage := c.ControlPlane.StatefulSet.Image
	syncerImage.Registry = cmp.Or(syncerImage.Registry, registryOverride, defaultRegistry)
	syncerImage.Repository = cmp.Or(syncerImage.Repository, defaultSyncerRepository)
	// the chart uses the chart version for the syncer tag, so the tag isn't set by default
	syncerImage.Tag = cmp.Or(syncerImage.Tag, version)
	syncer = syncerImage.String()

	var kubeImage config.Image
	switch c.Distro() {
	case config.K8SDistro:
		kubeImage = c.ControlPlane.Distro.K8S.Image
	case config.K3SDistro:
		kubeImage = c.ControlPlane.Distro.K3S.Image
	}

	kubeImage.Registry = cmp.Or(kubeImage.Registry, registryOverride, defaultRegistry)
	api = kubeImage.String()

	// with the platform driver if only the registry is set we won't be able to display complete info
	if kubeImage.Repository == "" {
		api = ""
	}

	return syncer, api
}
