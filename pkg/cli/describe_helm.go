package cli

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/helm"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/describe"
)

var (
	defaultRegistry         = "ghcr.io/loft-sh"
	defaultSyncerRepository = "vcluster-pro"
)

type DescribeOutput struct {
	Name         string   `json:"name,omitempty"`
	Namespace    string   `json:"namespace,omitempty"`
	Version      string   `json:"version,omitempty"`
	BackingStore string   `json:"backingStore,omitempty"`
	Distro       string   `json:"distro,omitempty"`
	Status       string   `json:"status,omitempty"`
	Created      v1.Time  `json:"created,omitempty"`
	ImageTags    ImageTag `json:"imageTags,omitempty"`
	Connected    bool     `json:"connected,omitempty"`
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
	w.Write(describe.LEVEL_0, "Distro:\t%s\n", do.Distro)
	w.Write(describe.LEVEL_0, "Created:\t%s\n", do.Created.Time.Format(time.RFC1123Z))
	w.Write(describe.LEVEL_0, "Status:\t%s\n", do.Status)
	w.Write(describe.LEVEL_0, "Connected:\t%t\n", do.Connected)

	w.Write(describe.LEVEL_0, "Images:\n")
	w.Write(describe.LEVEL_1, "apiServer:\t%s\n", do.ImageTags.APIServer)
	w.Write(describe.LEVEL_1, "syncer:\t%s\n", do.ImageTags.Syncer)

	out.Flush()
	return buf.String()
}

type ImageTag struct {
	APIServer string `json:"apiServer,omitempty"`
	Syncer    string `json:"syncer,omitempty"`
}

func DescribeHelm(ctx context.Context, flags *flags.GlobalFlags, output io.Writer, l log.Logger, name string, showConfig bool, format string) error {
	namespace := "vcluster-" + name
	if flags.Namespace != "" {
		namespace = flags.Namespace
	}

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

	vCluster, err := find.GetVCluster(ctx, rawConfig.CurrentContext, name, namespace, log.Discard)
	if err != nil {
		return err
	}

	configSecret, err := kubeClient.CoreV1().Secrets(namespace).Get(ctx, "vc-config-"+name, v1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to load the vcluster config: %w", err)
	}

	// Return only the user supplied vcluster.yaml, if showConfig is set
	if showConfig {
		// Log ArgoCD tracking id
		if trackingID, ok := configSecret.Annotations["argocd.argoproj.io/tracking-id"]; ok {
			components := strings.Split(trackingID, ":")
			if len(components) == 3 {
				l.Infof("The %s vcluster is managed via ArgoCD. Please refer to the %s ArgoCD Application configuration.", name, components[0])
			} else {
				l.Infof("The %s vcluster is managed via ArgoCD. argocd.argoproj.io/tracking-id: %s.", name, trackingID)
			}
		}

		// Load the user supplied vcluster.yaml from the HelmRelease Config field
		helmRelease, err := helm.NewSecrets(kubeClient).Get(ctx, name, namespace)
		if err != nil {
			return fmt.Errorf("failed to load the user supplied vcluster.yaml: %w", err)
		}

		return writeWithFormat(output, cmp.Or(format, "yaml"), helmRelease.Config)
	}

	configBytes, ok := configSecret.Data["config.yaml"]
	if !ok {
		return fmt.Errorf("configSecret %s in namespace %s does not contain the expected 'config.yaml' field", configSecret.Name, configSecret.Namespace)
	}

	conf, err := configPartialUnmarshal(configBytes)
	if err != nil {
		return err
	}

	syncer, api := getImageTags(conf, vCluster.Version)

	describeOutput := &DescribeOutput{
		Name:         vCluster.Name,
		Namespace:    vCluster.Namespace,
		Created:      vCluster.Created,
		Status:       string(vCluster.Status),
		Version:      vCluster.Version,
		Distro:       conf.Distro(),
		BackingStore: string(conf.BackingStoreType()),
		ImageTags: ImageTag{
			APIServer: api,
			Syncer:    syncer,
		},
	}

	return writeWithFormat(output, format, describeOutput)
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

// configPartialUnmarshal attempts to unmarshal only the relevant section
// of the config to avoid potential version mismatch error.
func configPartialUnmarshal(configBytes []byte) (*config.Config, error) {
	var partialConfig struct {
		ControlPlane config.ControlPlane `json:"controlPlane,omitempty"`
	}

	if err := yaml.Unmarshal(configBytes, &partialConfig); err != nil {
		return nil, err
	}

	return &config.Config{ControlPlane: partialConfig.ControlPlane}, nil
}

func marshalWithFormat(o any, format string) ([]byte, error) {
	switch format {
	case "json":
		return json.Marshal(o)
	case "yaml":
		return yaml.Marshal(o)
	case "":
		stringer, isStringer := o.(fmt.Stringer)
		if !isStringer {
			return nil, fmt.Errorf("only Stringer implementation is supported")
		}

		return []byte(stringer.String()), nil
	}

	return nil, fmt.Errorf("unknown format %s", format)
}

func writeWithFormat(writer io.Writer, format string, o any) error {
	outputBytes, err := marshalWithFormat(o, format)
	if err != nil {
		return err
	}

	_, err = writer.Write(outputBytes)
	return err
}
