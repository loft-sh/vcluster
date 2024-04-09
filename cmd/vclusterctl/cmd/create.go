package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/loft-sh/log/survey"
	"github.com/loft-sh/log/terminal"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/app/localkubernetes"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/find"
	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/embed"
	"github.com/loft-sh/vcluster/pkg/util/cliconfig"
	"github.com/loft-sh/vcluster/pkg/util/clihelper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	loftctlUtil "github.com/loft-sh/loftctl/v3/pkg/util"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/loft-sh/vcluster/pkg/util"
	"golang.org/x/mod/semver"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/loft-sh/vcluster/pkg/telemetry"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

var CreatedByVClusterAnnotation = "vcluster.loft.sh/created"

const LoftChartRepo = "https://charts.loft.sh"

var AllowedDistros = []string{"k8s", "k3s", "k0s", "eks"}

// Options holds the create cmd options
type Options struct {
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
}

type Values struct {
	Init Init `json:"init" mapstructure:"init"`
}

type Init struct {
	Manifests string      `json:"manifests" mapstructure:"manifests"`
	Helm      []HelmChart `json:"helm"      mapstructure:"helm"`
}

type HelmChart struct {
	Bundle    string      `json:"bundle,omitempty"    mapstructure:"bundle,omitempty"`
	Name      string      `json:"name,omitempty"      mapstructure:"name"`
	Repo      string      `json:"repo,omitempty"      mapstructure:"repo"`
	Version   string      `json:"version,omitempty"   mapstructure:"version"`
	Namespace string      `json:"namespace,omitempty" mapstructure:"namespace"`
	Values    string      `json:"values,omitempty"    mapstructure:"values"`
	Release   HelmRelease `json:"release,omitempty"   mapstructure:"release"`
}

type HelmRelease struct {
	Name      string `json:"name,omitempty"      mapstructure:"name"`
	Namespace string `json:"namespace,omitempty" mapstructure:"namespace"`
}

// CreateCmd holds the login cmd flags
type CreateCmd struct {
	*flags.GlobalFlags
	rawConfig        clientcmdapi.Config
	log              log.Logger
	kubeClientConfig clientcmd.ClientConfig
	kubeClient       *kubernetes.Clientset
	Options
	localCluster bool
}

// NewCreateCmd creates a new command
func NewCreateCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &CreateCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "create" + loftctlUtil.VClusterNameOnlyUseLine,
		Short: "Create a new virtual cluster",
		Long: `
#######################################################
################### vcluster create ###################
#######################################################
Creates a new virtual cluster

Example:
vcluster create test --namespace test
#######################################################
	`,
		Args: loftctlUtil.VClusterNameOnlyValidator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	// generic flags
	cobraCmd.Flags().StringVar(&cmd.KubeConfigContextName, "kube-config-context-name", "", "If set, will override the context name of the generated virtual cluster kube config with this name")
	cobraCmd.Flags().StringVar(&cmd.ChartVersion, "chart-version", upgrade.GetVersion(), "The virtual cluster chart version to use (e.g. v0.9.1)")
	cobraCmd.Flags().StringVar(&cmd.ChartName, "chart-name", "vcluster", "The virtual cluster chart name to use")
	cobraCmd.Flags().StringVar(&cmd.ChartRepo, "chart-repo", LoftChartRepo, "The virtual cluster chart repo to use")
	cobraCmd.Flags().StringVar(&cmd.Distro, "distro", "k8s", fmt.Sprintf("Kubernetes distro to use for the virtual cluster. Allowed distros: %s", strings.Join(AllowedDistros, ", ")))
	cobraCmd.Flags().StringVar(&cmd.KubernetesVersion, "kubernetes-version", "", "The kubernetes version to use (e.g. v1.20). Patch versions are not supported")
	cobraCmd.Flags().StringArrayVarP(&cmd.Values, "values", "f", []string{}, "Path where to load extra helm values from")
	cobraCmd.Flags().StringArrayVar(&cmd.SetValues, "set", []string{}, "Set values for helm. E.g. --set 'persistence.enabled=true'")

	cobraCmd.Flags().BoolVar(&cmd.CreateNamespace, "create-namespace", true, "If true the namespace will be created if it does not exist")
	cobraCmd.Flags().BoolVar(&cmd.UpdateCurrent, "update-current", true, "If true updates the current kube config")
	cobraCmd.Flags().BoolVar(&cmd.Expose, "expose", false, "If true will create a load balancer service to expose the vcluster endpoint")

	cobraCmd.Flags().BoolVar(&cmd.Connect, "connect", true, "If true will run vcluster connect directly after the vcluster was created")
	cobraCmd.Flags().BoolVar(&cmd.Upgrade, "upgrade", false, "If true will try to upgrade the vcluster instead of failing if it already exists")

	// hidden / deprecated
	cobraCmd.Flags().StringVar(&cmd.LocalChartDir, "local-chart-dir", "", "The virtual cluster local chart dir to use")
	cobraCmd.Flags().BoolVar(&cmd.ExposeLocal, "expose-local", true, "If true and a local Kubernetes distro is detected, will deploy vcluster with a NodePort service. Will be set to false and the passed value will be ignored if --expose is set to true.")

	_ = cobraCmd.Flags().MarkHidden("local-chart-dir")
	_ = cobraCmd.Flags().MarkHidden("expose-local")
	return cobraCmd
}

// Run executes the functionality
func (cmd *CreateCmd) Run(ctx context.Context, args []string) error {
	// make sure we deploy the correct version
	if cmd.ChartVersion == upgrade.DevelopmentVersion {
		cmd.ChartVersion = ""
	}

	// check helm binary
	helmBinaryPath, err := GetHelmBinaryPath(ctx, cmd.log)
	if err != nil {
		return err
	}

	output, err := exec.Command(helmBinaryPath, "version", "--client").CombinedOutput()
	if errHelm := clihelper.CheckHelmVersion(string(output)); errHelm != nil {
		return errHelm
	} else if err != nil {
		return err
	}

	err = cmd.prepare(ctx, args[0])
	if err != nil {
		return err
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
			return errors.Wrap(err, "create temp values file")
		}
		defer func(name string) {
			_ = os.Remove(name)
		}(tempValuesFile)

		_, err = tempFile.Write([]byte(decodedString))
		if err != nil {
			return errors.Wrap(err, "write values to temp values file")
		}

		err = tempFile.Close()
		if err != nil {
			return errors.Wrap(err, "close temp values file")
		}
		// setting new file to extraValues slice to process it further.
		newExtraValues = append(newExtraValues, tempValuesFile)
	}

	// resetting this as the base64 encoded strings should be removed and only valid file names should be kept.
	cmd.Values = newExtraValues

	// check if vcluster already exists
	if !cmd.Upgrade {
		release, err := helm.NewSecrets(cmd.kubeClient).Get(ctx, args[0], cmd.Namespace)
		if err != nil && !kerrors.IsNotFound(err) {
			return errors.Wrap(err, "get helm releases")
		} else if release != nil && release.Chart != nil && release.Chart.Metadata != nil && (release.Chart.Metadata.Name == "vcluster" || release.Chart.Metadata.Name == "vcluster-k0s" || release.Chart.Metadata.Name == "vcluster-k8s") && release.Secret != nil && release.Secret.Labels != nil && release.Secret.Labels["status"] == "deployed" {
			if cmd.Connect {
				connectCmd := &ConnectCmd{
					GlobalFlags:           cmd.GlobalFlags,
					UpdateCurrent:         cmd.UpdateCurrent,
					KubeConfigContextName: cmd.KubeConfigContextName,
					KubeConfig:            "./kubeconfig.yaml",
					Log:                   cmd.log,
				}

				return connectCmd.Connect(ctx, args[0], nil)
			}

			return fmt.Errorf("vcluster %s already exists in namespace %s\n- Use `vcluster create %s -n %s --upgrade` to upgrade the vcluster\n- Use `vcluster connect %s -n %s` to access the vcluster", args[0], cmd.Namespace, args[0], cmd.Namespace, args[0], cmd.Namespace)
		}
	}

	// we have to upgrade / install the chart
	err = cmd.deployChart(ctx, args[0], chartValues, helmBinaryPath)
	if err != nil {
		return err
	}

	// check if we should connect to the vcluster
	if cmd.Connect {
		cmd.log.Donef("Successfully created virtual cluster %s in namespace %s", args[0], cmd.Namespace)
		connectCmd := &ConnectCmd{
			GlobalFlags:           cmd.GlobalFlags,
			UpdateCurrent:         cmd.UpdateCurrent,
			KubeConfigContextName: cmd.KubeConfigContextName,
			KubeConfig:            "./kubeconfig.yaml",
			Log:                   cmd.log,
		}

		return connectCmd.Connect(ctx, args[0], nil)
	}

	if cmd.localCluster {
		cmd.log.Donef("Successfully created virtual cluster %s in namespace %s. \n- Use 'vcluster connect %s --namespace %s' to access the virtual cluster", args[0], cmd.Namespace, args[0], cmd.Namespace)
	} else {
		cmd.log.Donef("Successfully created virtual cluster %s in namespace %s. \n- Use 'vcluster connect %s --namespace %s' to access the virtual cluster\n- Use `vcluster connect %s --namespace %s -- kubectl get ns` to run a command directly within the vcluster", args[0], cmd.Namespace, args[0], cmd.Namespace, args[0], cmd.Namespace)
	}

	return nil
}

func getBase64DecodedString(values string) (string, error) {
	strDecoded, err := base64.StdEncoding.DecodeString(values)
	if err != nil {
		return "", err
	}
	return string(strDecoded), nil
}

func (cmd *CreateCmd) deployChart(ctx context.Context, vClusterName, chartValues, helmExecutablePath string) error {
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
			embeddedChartPath := fmt.Sprintf("charts/%s", embeddedChartName)
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
		if !chartEmbedded && cmd.ChartRepo == LoftChartRepo && cmd.ChartVersion != "" { // specify versioned path to repo url
			cmd.LocalChartDir = LoftChartRepo + "/charts/" + cmd.ChartName + "-" + strings.TrimPrefix(cmd.ChartVersion, "v") + ".tgz"
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

func (cmd *CreateCmd) ToChartOptions(kubernetesVersion *version.Info, log log.Logger) (*config.ExtraValuesOptions, error) {
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

func (cmd *CreateCmd) prepare(ctx context.Context, vClusterName string) error {
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
					return errors.Wrap(err, "switch context")
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

func (cmd *CreateCmd) ensureNamespace(ctx context.Context, vClusterName string) error {
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

func (cmd *CreateCmd) createNamespace(ctx context.Context) error {
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
		return errors.Wrap(err, "create namespace")
	}
	return nil
}

func (cmd *CreateCmd) getKubernetesVersion() (*version.Info, error) {
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
