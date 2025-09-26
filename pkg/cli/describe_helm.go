package cli

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/loft-sh/vcluster/pkg/util/helmdownloader"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	defaultRegistry         = "ghcr.io/loft-sh"
	defaultSyncerRepository = "vcluster-pro"
)

type DescribeOptions struct {
	OutputFormat string
	AllValues    bool

	// Helm show values options
	GenerateUserSuppliedConfigIfMissing bool
	ChartName                           string
	ChartRepo                           string

	// Platform options
	Project string
}

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

type describeHelm struct {
	name      string
	namespace string

	options     *DescribeOptions
	globalFlags *flags.GlobalFlags

	kubeContext string
	kubeClient  kubernetes.Interface

	log log.Logger
}

func DescribeHelm(ctx context.Context, options *DescribeOptions, globalFlags *flags.GlobalFlags, name string, log log.Logger) ([]byte, error) {
	kConf := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})
	clientConfig, err := kConf.ClientConfig()
	if err != nil {
		return nil, err
	}

	rawConfig, err := kConf.RawConfig()
	if err != nil {
		return nil, err
	}

	kubeClient, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return nil, err
	}

	namespace := "vcluster-" + name
	if globalFlags.Namespace != "" {
		namespace = globalFlags.Namespace
	}

	cmd := &describeHelm{
		name:        name,
		namespace:   namespace,
		options:     options,
		globalFlags: globalFlags,
		kubeContext: rawConfig.CurrentContext,
		kubeClient:  kubeClient,
		log:         log,
	}

	return cmd.run(ctx)
}

func (cmd *describeHelm) run(ctx context.Context) ([]byte, error) {
	// Return vcluster summary
	if cmd.options.OutputFormat == "" {
		describeOutput, err := cmd.getDefaultDescribeOutput(ctx)
		if err != nil {
			return nil, err
		}

		return yaml.Marshal(describeOutput)
	}

	// Return all vCluster config
	if cmd.options.AllValues {
		configSecret, err := cmd.kubeClient.CoreV1().Secrets(cmd.namespace).Get(ctx, "vc-config-"+cmd.name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to load config for vcluster %s: %w", cmd.name, err)
		}

		configBytes, ok := configSecret.Data["config.yaml"]
		if !ok {
			return nil, fmt.Errorf("could not find config.yaml in secret %s", configSecret.Name)
		}

		return cmd.bytesWithFormat(configBytes)
	}

	// Return user supplied vcluster.yaml
	configBytes, err := cmd.getUserSuppliedConfig(ctx)
	if err != nil {
		return nil, err
	}

	return cmd.bytesWithFormat(configBytes)
}

func (cmd *describeHelm) getDefaultDescribeOutput(ctx context.Context) (*DescribeOutput, error) {
	fVcluster, err := find.GetVCluster(ctx, cmd.kubeContext, cmd.name, cmd.namespace, cmd.log)
	if err != nil {
		return nil, err
	}

	configSecret, err := cmd.kubeClient.CoreV1().Secrets(cmd.namespace).Get(ctx, "vc-config-"+cmd.name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to load config for vcluster %s: %w", cmd.name, err)
	}

	configBytes, ok := configSecret.Data["config.yaml"]
	if !ok {
		return nil, fmt.Errorf("could not find config.yaml in secret %s", configSecret.Name)
	}

	conf := &config.Config{}
	err = yaml.Unmarshal(configBytes, conf)
	if err != nil {
		return nil, err
	}

	syncer, api := getImageTags(conf, fVcluster.Version)

	describeOutput := &DescribeOutput{
		Distro:       conf.Distro(),
		BackingStore: string(conf.BackingStoreType()),
		Version:      fVcluster.Version,
		ImageTags: ImageTag{
			Syncer:    syncer,
			APIServer: api,
		},
	}

	return describeOutput, nil
}

func (cmd *describeHelm) getUserSuppliedConfig(ctx context.Context) ([]byte, error) {
	// Attempt to get the user supplied values from the helm release
	helmRelease, err := helm.NewSecrets(cmd.kubeClient).Get(ctx, cmd.name, cmd.namespace)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to load helm release: %w", err)
		}
	}

	if helmRelease != nil {
		return yaml.Marshal(helmRelease.Config)
	}

	configSecret, err := cmd.kubeClient.CoreV1().Secrets(cmd.namespace).Get(ctx, "vc-config-"+cmd.name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to load config for vcluster %s: %w", cmd.name, err)
	}

	// Log ArgoCD tracking id
	if trackingID, ok := configSecret.Annotations["argocd.argoproj.io/tracking-id"]; ok {
		components := strings.Split(trackingID, ":")
		if len(components) == 3 {
			cmd.log.Infof("The %s vcluster is managed via ArgoCD. Please refer to the %s ArgoCD Application configuration.", cmd.name, components[0])
		} else {
			cmd.log.Infof("The %s vcluster is managed via ArgoCD. argocd.argoproj.io/tracking-id: %s.", cmd.name, trackingID)
		}
	}

	// Attempt to generate the user supplied values by creating a diff between default and actual values.
	if cmd.options.GenerateUserSuppliedConfigIfMissing {
		cmd.log.Info("Failed to get the user supplied configuration. Fallback to generating the diff between default and actual values.")
		return cmd.generateUserSuppliedConfig(ctx, configSecret)
	}

	return nil, fmt.Errorf("failed to load the user supplied vcluster.yaml")
}

func (cmd *describeHelm) generateUserSuppliedConfig(ctx context.Context, configSecret *corev1.Secret) ([]byte, error) {
	allValuesBytes, ok := configSecret.Data["config.yaml"]
	if !ok {
		return nil, fmt.Errorf("could not find config.yaml in secret %s", configSecret.Name)
	}

	allValuesRaw := make(map[string]interface{})
	if err := yaml.Unmarshal(allValuesBytes, &allValuesRaw); err != nil {
		return nil, err
	}

	chartVersion, err := cmd.getChartVersionFromChartLabel(configSecret.Labels["chart"])
	if err != nil {
		return nil, err
	}

	defaultValuesBytes, err := cmd.getHelmChartDefaultValues(ctx, chartVersion)
	if err != nil {
		return nil, err
	}

	defaultValuesRaw := make(map[string]interface{})
	if err = yaml.Unmarshal(defaultValuesBytes, &defaultValuesRaw); err != nil {
		return nil, err
	}

	userSuppliedValues := config.RawDiff(defaultValuesRaw, allValuesRaw)
	if userSuppliedValues == nil {
		cmd.log.Warn("Empty user supplied vcluster.yaml")
		return nil, nil
	}

	return yaml.Marshal(userSuppliedValues)
}

func (cmd *describeHelm) getHelmChartDefaultValues(ctx context.Context, chartVersion string) ([]byte, error) {
	helmBinaryPath, err := helmdownloader.GetHelmBinaryPath(ctx, cmd.log)
	if err != nil {
		return nil, err
	}

	helmArgs := []string{"show", "values", cmd.options.ChartName, "--version", chartVersion}

	if cmd.options.ChartRepo != "" {
		helmArgs = append(helmArgs, "--repo", cmd.options.ChartRepo)
	}

	cmd.log.Infof("Getting default values: helm %s", strings.Join(helmArgs, " "))
	helmShowCmd := exec.CommandContext(ctx, helmBinaryPath, helmArgs...)

	var helmShowCmdStderr bytes.Buffer
	helmShowCmd.Stderr = &helmShowCmdStderr

	var helmShowCmdStdout bytes.Buffer
	helmShowCmd.Stdout = &helmShowCmdStdout

	if err := helmShowCmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get default values: %s", helmShowCmdStderr.String())
	}

	return helmShowCmdStdout.Bytes(), nil
}

// getChartVersionFromChartLabel returns chart version from a string with CHART_NAME-CHART_VERSION format.
func (cmd *describeHelm) getChartVersionFromChartLabel(value string) (string, error) {
	components := strings.SplitAfterN(value, "-", 2)
	if len(components) != 2 {
		return "", fmt.Errorf("failed to get ChartVersion from %s ", value)
	}
	return components[1], nil
}

func (cmd *describeHelm) bytesWithFormat(yamlBytes []byte) ([]byte, error) {
	if cmd.options.OutputFormat == "json" {
		return yaml.YAMLToJSON(yamlBytes)
	}

	return yamlBytes, nil
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
