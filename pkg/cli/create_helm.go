package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
	"github.com/loft-sh/log/terminal"
	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/config/legacyconfig"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/localkubernetes"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/embed"
	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/loft-sh/vcluster/pkg/telemetry"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/loft-sh/vcluster/pkg/util"
	"github.com/loft-sh/vcluster/pkg/util/cliconfig"
	"github.com/loft-sh/vcluster/pkg/util/clihelper"
	"github.com/loft-sh/vcluster/pkg/util/helmdownloader"
	"golang.org/x/mod/semver"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// CreateOptions holds the create cmd options
type CreateOptions struct {
	Manager string

	KubeConfigContextName string
	ChartVersion          string
	ChartName             string
	ChartRepo             string
	LocalChartDir         string
	Distro                string
	Values                []string
	SetValues             []string

	KubernetesVersion string

	CreateNamespace bool
	UpdateCurrent   bool
	Expose          bool
	ExposeLocal     bool

	Connect bool
	Upgrade bool

	// Platform
	Project         string
	Cluster         string
	Template        string
	TemplateVersion string
	Links           []string
	Annotations     []string
	Labels          []string
	Params          string
	SetParams       []string
}

var CreatedByVClusterAnnotation = "vcluster.loft.sh/created"

var AllowedDistros = []string{"k8s", "k3s", "k0s", "eks"}

type createHelm struct {
	*flags.GlobalFlags
	*CreateOptions

	rawConfig        clientcmdapi.Config
	log              log.Logger
	kubeClientConfig clientcmd.ClientConfig
	kubeClient       *kubernetes.Clientset
	localCluster     bool
}

func CreateHelm(ctx context.Context, options *CreateOptions, globalFlags *flags.GlobalFlags, vClusterName string, log log.Logger) error {
	cmd := &createHelm{
		GlobalFlags:   globalFlags,
		CreateOptions: options,

		log: log,
	}

	// make sure we deploy the correct version
	if options.ChartVersion == upgrade.DevelopmentVersion {
		options.ChartVersion = ""
	}

	// check helm binary
	helmBinaryPath, err := helmdownloader.GetHelmBinaryPath(ctx, cmd.log)
	if err != nil {
		return err
	}

	output, err := exec.Command(helmBinaryPath, "version", "--client", "--template", "{{.Version}}").Output()
	if err != nil {
		return err
	}

	err = clihelper.CheckHelmVersion(string(output))
	if err != nil {
		return err
	}

	err = cmd.prepare(ctx, vClusterName)
	if err != nil {
		return err
	}

	release, err := helm.NewSecrets(cmd.kubeClient).Get(ctx, vClusterName, cmd.Namespace)
	if err != nil && !kerrors.IsNotFound(err) {
		return fmt.Errorf("get current helm release: %w", err)
	}

	// check if vcluster already exists
	if !cmd.Upgrade {
		if isVClusterDeployed(release) {
			if cmd.Connect {
				return ConnectHelm(ctx, &ConnectOptions{
					UpdateCurrent:         cmd.UpdateCurrent,
					KubeConfigContextName: cmd.KubeConfigContextName,
					KubeConfig:            "./kubeconfig.yaml",
				}, cmd.GlobalFlags, vClusterName, nil, cmd.log)
			}

			return fmt.Errorf("vcluster %s already exists in namespace %s\n- Use `vcluster create %s -n %s --upgrade` to upgrade the vcluster\n- Use `vcluster connect %s -n %s` to access the vcluster", vClusterName, cmd.Namespace, vClusterName, cmd.Namespace, vClusterName, cmd.Namespace)
		}
	}

	var currentValues string
	if isVClusterDeployed(release) {
		currentValues, err = configValuesYAML(release)
		if err != nil {
			return err
		}

		currentConfig := &config.Config{}
		if err := currentConfig.DecodeYAML(strings.NewReader(currentValues)); err != nil {
			// TODO Delete after vCluster 0.19.x resp. the old config format is out of support.
			// We have to check if we have a virtual cluster running with a legacy config.
			if !errors.Is(err, config.ErrInvalidFileFormat) {
				return err
			}
			var currentDistro string
			if semver.Compare("v"+release.Chart.Metadata.Version, "v0.20.0-alpha.0") == -1 {
				// We have to infer the distro from the current chart name.
				// If we are upgrading a vCluster < v0.20 the old k3s chart is the one without a prefix.
				currentDistro = strings.TrimPrefix(release.Chart.Metadata.Name, "vcluster-")
				if currentDistro == "vcluster" {
					currentDistro = "k3s"
				}
			}

			convertedValues, err := legacyconfig.MigrateLegacyConfig(currentDistro, currentValues)
			if err != nil {
				return err
			}
			// If users run a virtual cluster < v0.20 and they did not provide any values files, we abort the upgrade with
			// a prompt to convert the config first.
			// We do this because we don't want to "automagically" convert the old config implicitly, without the users
			// realizing that they are still using the old format.
			if len(cmd.Values) <= 0 {
				command := fmt.Sprintf("vcluster convert config --distro %s - <<EOF\n%s\nEOF", currentDistro, currentValues)
				return fmt.Errorf("your virtual cluster is running using the old config format, please issue the following to convert your current config values to the new v0.20 format before doing the upgrade again using the converted config file:\n%s", command)
			}
			// The user did provide a values file, so we use the converted values in order to be able to validate them down below.
			currentValues = convertedValues
			// TODO end
		}
	}

	// find out kubernetes version
	kubernetesVersion, err := cmd.getKubernetesVersion()
	if err != nil {
		return err
	}

	// load the default values
	chartOptions, err := cmd.ToChartOptions(kubernetesVersion, cmd.log)
	if err != nil {
		return err
	}
	chartValues, err := config.GetExtraValues(chartOptions)
	if err != nil {
		return err
	}

	// If the user did not pass any values files, add the default values in order to validate them against the current config values
	// regarding a potential distro/backingstore change.
	if len(cmd.Values) <= 0 {
		cmd.Values = append(cmd.Values, base64.StdEncoding.EncodeToString([]byte(chartValues)))
	}

	// Always prepend our current values, which will then be aggregated into newExtraValues and then validated.
	vals := base64.StdEncoding.EncodeToString([]byte(currentValues))
	cmd.Values = append([]string{vals}, cmd.Values...)

	var newExtraValues []string
	for _, value := range cmd.Values {
		decodedString, err := getBase64DecodedString(value)
		// ignore decoding errors and treat it as non-base64 string
		if err != nil {
			newExtraValues = append(newExtraValues, value)
			continue
		}

		// write a temporary values file
		tempFile, err := os.CreateTemp("", "")
		tempValuesFile := tempFile.Name()
		if err != nil {
			return fmt.Errorf("create temp values file: %w", err)
		}
		defer func(name string) {
			_ = os.Remove(name)
		}(tempValuesFile)

		_, err = tempFile.Write([]byte(decodedString))
		if err != nil {
			return fmt.Errorf("write values to temp values file: %w", err)
		}

		err = tempFile.Close()
		if err != nil {
			return fmt.Errorf("close temp values file: %w", err)
		}
		// setting new file to extraValues slice to process it further.
		newExtraValues = append(newExtraValues, tempValuesFile)
	}

	// Check if the passed in values adhere to our config format.
	for _, p := range newExtraValues {
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()

		data, err := io.ReadAll(f)
		if err != nil {
			return err
		}

		newConfig := &config.Config{}
		if err = newConfig.DecodeYAML(bytes.NewBuffer(data)); err != nil {
			if !errors.Is(err, config.ErrInvalidFileFormat) {
				return err
			}
			// TODO Delete after vCluster 0.19.x resp. the old config format is out of support.
			// If the user passed in a legacy config we abort the upgrade and log with a prompt to convert the values first.
			if _, err := tryParseLegacyConfig(data); err == nil {
				return fmt.Errorf("you have passed a config with old values format, please use %q to convert your config first and then re-run the upgrade", "vcluster convert config")
			}
			// TODO end
			return err
		}

		currentConfig := &config.Config{}
		if err = currentConfig.DecodeYAML(strings.NewReader(currentValues)); err != nil {
			return err
		}

		if err := config.ValidateChanges(currentConfig, newConfig); err != nil {
			return err
		}
	}

	// resetting this as the base64 encoded strings should be removed and only valid file names should be kept.
	cmd.Values = newExtraValues

	// we have to upgrade / install the chart
	err = cmd.deployChart(ctx, vClusterName, chartValues, helmBinaryPath)
	if err != nil {
		return err
	}

	// check if we should connect to the vcluster
	if cmd.Connect {
		cmd.log.Donef("Successfully created virtual cluster %s in namespace %s", vClusterName, cmd.Namespace)
		return ConnectHelm(ctx, &ConnectOptions{
			UpdateCurrent:         cmd.UpdateCurrent,
			KubeConfigContextName: cmd.KubeConfigContextName,
			KubeConfig:            "./kubeconfig.yaml",
		}, cmd.GlobalFlags, vClusterName, nil, cmd.log)
	}

	if cmd.localCluster {
		cmd.log.Donef("Successfully created virtual cluster %s in namespace %s. \n- Use 'vcluster connect %s --namespace %s' to access the virtual cluster", vClusterName, cmd.Namespace, vClusterName, cmd.Namespace)
	} else {
		cmd.log.Donef("Successfully created virtual cluster %s in namespace %s. \n- Use 'vcluster connect %s --namespace %s' to access the virtual cluster\n- Use `vcluster connect %s --namespace %s -- kubectl get ns` to run a command directly within the vcluster", vClusterName, cmd.Namespace, vClusterName, cmd.Namespace, vClusterName, cmd.Namespace)
	}

	return nil
}

func isVClusterDeployed(release *helm.Release) bool {
	return release != nil &&
		release.Chart != nil &&
		release.Chart.Metadata != nil &&
		(release.Chart.Metadata.Name == "vcluster" || release.Chart.Metadata.Name == "vcluster-k0s" ||
			release.Chart.Metadata.Name == "vcluster-k8s" || release.Chart.Metadata.Name == "vcluster-eks") &&
		release.Secret != nil &&
		release.Secret.Labels != nil &&
		release.Secret.Labels["status"] == "deployed"
}

// configValuesYAML retrieves the helm config values for the given release.
func configValuesYAML(release *helm.Release) (string, error) {
	cfg := release.Config
	vals, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}

	return string(vals), nil
}

// TODO Delete after vCluster 0.19.x resp. the old config format is out of support.
// tryParseLegacyConfig tries to parse a config in legacy format (k0s/k3s or k8s/eks).
func tryParseLegacyConfig(data []byte) (string, error) {
	var vals []byte
	k0sOrk3s := &legacyconfig.LegacyK0sAndK3s{}
	if err := k0sOrk3s.DecodeYAML(bytes.NewBuffer(data)); err != nil {
		k8sOrEKS := &legacyconfig.LegacyK8s{}
		if err := k8sOrEKS.DecodeYAML(bytes.NewBuffer(data)); err != nil {
			return "", err
		}
		vals, err = yaml.Marshal(k8sOrEKS)
		if err != nil {
			return "", err
		}
	} else {
		vals, err = yaml.Marshal(k0sOrk3s)
		if err != nil {
			return "", err
		}
	}

	return string(vals), nil
}

// TODO end

func getBase64DecodedString(values string) (string, error) {
	strDecoded, err := base64.StdEncoding.DecodeString(values)
	if err != nil {
		return "", err
	}
	return string(strDecoded), nil
}

func (cmd *createHelm) deployChart(ctx context.Context, vClusterName, chartValues, helmExecutablePath string) error {
	// check if there is a vcluster directory already
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("unable to get current work directory: %w", err)
	}
	if _, err := os.Stat(filepath.Join(workDir, cmd.ChartName)); err == nil {
		return fmt.Errorf("aborting vcluster creation. Current working directory contains a file or a directory with the name equal to the vcluster chart name - \"%s\". Please execute vcluster create command from a directory that doesn't contain a file or directory named \"%s\"", cmd.ChartName, cmd.ChartName)
	}

	if cmd.LocalChartDir == "" {
		chartEmbedded := false
		if cmd.ChartVersion == upgrade.GetVersion() { // use embedded chart if default version
			embeddedChartName := fmt.Sprintf("%s-%s.tgz", cmd.ChartName, upgrade.GetVersion())
			// not using filepath.Join because the embed.FS separator is not OS specific
			embeddedChartPath := fmt.Sprintf("chart/%s", embeddedChartName)
			embeddedChartFile, err := embed.Charts.ReadFile(embeddedChartPath)
			if err != nil && errors.Is(err, fs.ErrNotExist) {
				cmd.log.Infof("Chart not embedded: %q, pulling from helm repository.", err)
			} else if err != nil {
				cmd.log.Errorf("Unexpected error while accessing embedded file: %q", err)
			} else {
				temp, err := os.CreateTemp("", fmt.Sprintf("%s%s", embeddedChartName, "-"))
				if err != nil {
					cmd.log.Errorf("Error creating temp file: %v", err)
				} else {
					defer temp.Close()
					defer os.Remove(temp.Name())
					_, err = temp.Write(embeddedChartFile)
					if err != nil {
						cmd.log.Errorf("Error writing package file to temp: %v", err)
					}
					cmd.LocalChartDir = temp.Name()
					chartEmbedded = true
					cmd.log.Debugf("Using embedded chart: %q", embeddedChartName)
				}
			}
		}

		// rewrite chart location, this is an optimization to avoid
		// downloading the whole index.yaml and parsing it
		if !chartEmbedded && cmd.ChartRepo == constants.LoftChartRepo && cmd.ChartVersion != "" { // specify versioned path to repo url
			cmd.LocalChartDir = constants.LoftChartRepo + "/charts/" + cmd.ChartName + "-" + strings.TrimPrefix(cmd.ChartVersion, "v") + ".tgz"
		}
	}

	if cmd.Upgrade {
		cmd.log.Infof("Upgrade vcluster %s...", vClusterName)
	} else {
		cmd.log.Infof("Create vcluster %s...", vClusterName)
	}

	// we have to upgrade / install the chart
	err = helm.NewClient(&cmd.rawConfig, cmd.log, helmExecutablePath).Upgrade(ctx, vClusterName, cmd.Namespace, helm.UpgradeOptions{
		Chart:       cmd.ChartName,
		Repo:        cmd.ChartRepo,
		Version:     cmd.ChartVersion,
		Path:        cmd.LocalChartDir,
		Values:      chartValues,
		ValuesFiles: cmd.Values,
		SetValues:   cmd.SetValues,
		Debug:       cmd.Debug,
	})
	if err != nil {
		return err
	}

	return nil
}

func (cmd *createHelm) ToChartOptions(kubernetesVersion *version.Info, log log.Logger) (*config.ExtraValuesOptions, error) {
	if !util.Contains(cmd.Distro, AllowedDistros) {
		return nil, fmt.Errorf("unsupported distro %s, please select one of: %s", cmd.Distro, strings.Join(AllowedDistros, ", "))
	}

	// check if we should create with node port
	clusterType := localkubernetes.DetectClusterType(&cmd.rawConfig)
	if cmd.ExposeLocal && clusterType.LocalKubernetes() {
		cmd.log.Infof("Detected local kubernetes cluster %s. Will deploy vcluster with a NodePort & sync real nodes", clusterType)
		cmd.localCluster = true
	}

	return &config.ExtraValuesOptions{
		Distro:    cmd.Distro,
		Expose:    cmd.Expose,
		SyncNodes: cmd.localCluster,
		NodePort:  cmd.localCluster,
		KubernetesVersion: config.KubernetesVersion{
			Major: kubernetesVersion.Major,
			Minor: kubernetesVersion.Minor,
		},
		DisableTelemetry:    cliconfig.GetConfig(log).TelemetryDisabled,
		InstanceCreatorType: "vclusterctl",
		MachineID:           telemetry.GetMachineID(log),
	}, nil
}

func (cmd *createHelm) prepare(ctx context.Context, vClusterName string) error {
	// first load the kube config
	kubeClientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{
		CurrentContext: cmd.Context,
	})

	// load the raw config
	rawConfig, err := kubeClientConfig.RawConfig()
	if err != nil {
		return fmt.Errorf("there is an error loading your current kube config (%w), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}
	if cmd.Context != "" {
		rawConfig.CurrentContext = cmd.Context
	}

	// check if vcluster in vcluster
	_, _, previousContext := find.VClusterFromContext(rawConfig.CurrentContext)
	if previousContext != "" {
		if terminal.IsTerminalIn {
			switchBackOption := "No, switch back to context " + previousContext
			out, err := cmd.log.Question(&survey.QuestionOptions{
				Question:     "You are creating a vcluster inside another vcluster, is this desired?",
				DefaultValue: switchBackOption,
				Options:      []string{switchBackOption, "Yes"},
			})
			if err != nil {
				return err
			}

			if out == switchBackOption {
				cmd.Context = previousContext
				kubeClientConfig = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{
					CurrentContext: cmd.Context,
				})
				rawConfig, err = kubeClientConfig.RawConfig()
				if err != nil {
					return fmt.Errorf("there is an error loading your current kube config (%w), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
				}
				err = find.SwitchContext(&rawConfig, cmd.Context)
				if err != nil {
					return fmt.Errorf("switch context: %w", err)
				}
			}
		} else {
			cmd.log.Warnf("You are creating a vcluster inside another vcluster, is this desired?")
		}
	}

	// load the rest config
	kubeConfig, err := kubeClientConfig.ClientConfig()
	if err != nil {
		return fmt.Errorf("there is an error loading your current kube config (%w), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}

	client, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}

	cmd.kubeClient = client
	cmd.kubeClientConfig = kubeClientConfig
	cmd.rawConfig = rawConfig

	// ensure namespace
	err = cmd.ensureNamespace(ctx, vClusterName)
	if err != nil {
		return err
	}

	return nil
}

func (cmd *createHelm) ensureNamespace(ctx context.Context, vClusterName string) error {
	var err error
	if cmd.Namespace == "" {
		cmd.Namespace, _, err = cmd.kubeClientConfig.Namespace()
		if err != nil {
			return err
		} else if cmd.Namespace == "" || cmd.Namespace == "default" {
			cmd.Namespace = "vcluster-" + vClusterName
			cmd.log.Debugf("Will use namespace %s to create the vcluster...", cmd.Namespace)
		}
	}

	// make sure namespace exists
	namespace, err := cmd.kubeClient.CoreV1().Namespaces().Get(ctx, cmd.Namespace, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return cmd.createNamespace(ctx)
		} else if !kerrors.IsForbidden(err) {
			return err
		}
	} else if namespace.DeletionTimestamp != nil {
		cmd.log.Infof("Waiting until namespace is terminated...")
		err := wait.PollUntilContextTimeout(ctx, time.Second, time.Minute*2, false, func(ctx context.Context) (bool, error) {
			namespace, err := cmd.kubeClient.CoreV1().Namespaces().Get(ctx, cmd.Namespace, metav1.GetOptions{})
			if err != nil {
				if kerrors.IsNotFound(err) {
					return true, nil
				}

				return false, err
			}

			return namespace.DeletionTimestamp == nil, nil
		})
		if err != nil {
			return err
		}

		// create namespace
		return cmd.createNamespace(ctx)
	}

	return nil
}

func (cmd *createHelm) createNamespace(ctx context.Context) error {
	// try to create the namespace
	cmd.log.Infof("Creating namespace %s", cmd.Namespace)
	_, err := cmd.kubeClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: cmd.Namespace,
			Annotations: map[string]string{
				CreatedByVClusterAnnotation: "true",
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("create namespace: %w", err)
	}
	return nil
}

func (cmd *createHelm) getKubernetesVersion() (*version.Info, error) {
	var (
		kubernetesVersion *version.Info
		err               error
	)
	if cmd.KubernetesVersion != "" {
		if cmd.KubernetesVersion[0] != 'v' {
			cmd.KubernetesVersion = "v" + cmd.KubernetesVersion
		}

		if !semver.IsValid(cmd.KubernetesVersion) {
			return nil, fmt.Errorf("please use valid semantic versioning format, e.g. vX.X")
		}

		majorMinorVer := semver.MajorMinor(cmd.KubernetesVersion)

		if splittedVersion := strings.Split(cmd.KubernetesVersion, "."); len(splittedVersion) > 2 {
			cmd.log.Warnf("currently we only support major.minor version (%s) and not the patch version (%s)", majorMinorVer, cmd.KubernetesVersion)
		}

		parsedVersion, err := config.ParseKubernetesVersionInfo(majorMinorVer)
		if err != nil {
			return nil, err
		}

		kubernetesVersion = &version.Info{
			Major: parsedVersion.Major,
			Minor: parsedVersion.Minor,
		}
	}

	if kubernetesVersion == nil {
		kubernetesVersion, err = cmd.kubeClient.ServerVersion()
		if err != nil {
			return nil, err
		}
	}

	return kubernetesVersion, nil
}
