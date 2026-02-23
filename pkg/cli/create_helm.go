package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
	"github.com/loft-sh/log/terminal"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/localkubernetes"
	pkgconfig "github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/embed"
	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/loft-sh/vcluster/pkg/lifecycle"
	"github.com/loft-sh/vcluster/pkg/platform"
	platformclihelper "github.com/loft-sh/vcluster/pkg/platform/clihelper"
	"github.com/loft-sh/vcluster/pkg/platform/sleepmode"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/snapshot/pod"
	"github.com/loft-sh/vcluster/pkg/telemetry"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/loft-sh/vcluster/pkg/util/clihelper"
	"github.com/loft-sh/vcluster/pkg/util/helmdownloader"
	"github.com/loft-sh/vcluster/pkg/util/namespaces"
	"sigs.k8s.io/yaml"
)

// CreateOptions holds the create cmd options
type CreateOptions struct {
	Driver string

	KubeConfigContextName string
	ChartVersion          string
	ChartName             string
	ChartRepo             string
	LocalChartDir         string
	Values                []string
	SetValues             []string
	Print                 bool

	KubernetesVersion string

	CreateNamespace      bool
	UpdateCurrent        bool
	BackgroundProxy      bool
	BackgroundProxyImage string
	Add                  bool
	Expose               bool
	ExposeLocal          bool
	Restore              string
	Connect              bool
	Upgrade              bool

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
	Description     string
	DisplayName     string
	Team            string
	User            string
	UseExisting     bool
	Recreate        bool
	SkipWait        bool
}

var CreatedByVClusterAnnotation = "vcluster.loft.sh/created"

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

	output, err := exec.Command(helmBinaryPath, "version", "--template", "{{.Version}}").Output()
	if err != nil {
		return err
	}

	err = clihelper.CheckHelmVersion(string(output))
	if err != nil {
		return err
	}

	vClusters, err := find.ListVClusters(ctx, cmd.Context, "", cmd.Namespace, log)
	if err != nil {
		return err
	}

	// from v0.25 onwards, creation of multiple vClusters inside the same ns is not allowed
	for _, v := range vClusters {
		if v.Namespace == cmd.Namespace && v.Name != vClusterName {
			return fmt.Errorf("there is already a virtual cluster in namespace %s; "+
				"creating multiple virtual clusters inside the same namespace is not supported", cmd.Namespace)
		}
	}

	err = cmd.prepare(ctx, vClusterName)
	if err != nil {
		return err
	}

	release, err := helm.NewSecrets(cmd.kubeClient).Get(ctx, vClusterName, cmd.Namespace)
	if err != nil && !kerrors.IsNotFound(err) {
		return fmt.Errorf("get current helm release: %w", err)
	}

	_, err = cmd.kubeClient.CoreV1().Services(globalFlags.Namespace).Get(ctx, platformclihelper.DefaultPlatformServiceName, metav1.GetOptions{})
	if err == nil {
		return fmt.Errorf("a vCluster platform installation exists in the namespace '%s'. Aborting install", globalFlags.Namespace)
	} else if !kerrors.IsNotFound(err) {
		return fmt.Errorf("get platform service: %w", err)
	}

	// check if vcluster already exists
	if !cmd.Upgrade {
		if isVClusterDeployed(release) {
			if cmd.Restore != "" {
				log.Infof("Resuming vCluster %s after it was paused", vClusterName)
				err = lifecycle.ResumeVCluster(ctx, cmd.kubeClient, vClusterName, cmd.Namespace, true, log)
				if err != nil {
					log.Infof("Skipped resuming vCluster %s", vClusterName)
				}

				log.Infof("Restore vCluster %s...", vClusterName)
				err = Restore(ctx, []string{vClusterName, cmd.Restore}, globalFlags, &snapshot.Options{}, &pod.Options{}, false, false, log)
				if err != nil {
					return fmt.Errorf("restore vCluster %s: %w", vClusterName, err)
				}
			}
			if cmd.Connect {
				return ConnectHelm(ctx, &ConnectOptions{
					BackgroundProxy:       cmd.BackgroundProxy,
					UpdateCurrent:         true,
					KubeConfigContextName: cmd.KubeConfigContextName,
					KubeConfig:            "./kubeconfig.yaml",
					BackgroundProxyImage:  cmd.BackgroundProxyImage,
				}, cmd.GlobalFlags, vClusterName, nil, cmd.log)
			}

			return fmt.Errorf("vcluster %s already exists in namespace %s\n- Use `vcluster create %s -n %s --upgrade` to upgrade the vcluster\n- Use `vcluster connect %s -n %s` to access the vcluster", vClusterName, cmd.Namespace, vClusterName, cmd.Namespace, vClusterName, cmd.Namespace)
		}
	}

	currentVClusterConfig := &config.Config{}
	if isVClusterDeployed(release) {
		currentValues, err := helmExtraValuesYAML(release)
		if err != nil {
			return err
		}
		currentVClusterConfig, err = getConfigfileFromSecret(ctx, vClusterName, cmd.Namespace)
		if err != nil {
			return err
		}

		if len(cmd.Values) == 0 {
			if err := confirmConfigIncompatibility(currentVClusterConfig, currentValues, log); err != nil {
				return err
			}
		}
	}

	// build extra values
	filesToRemove, err := buildExtraValues(ctx, cmd.CreateOptions, log)
	if err != nil {
		return err
	}
	defer func() {
		for _, file := range filesToRemove {
			os.Remove(file)
		}
	}()

	// load the default values
	chartOptions, err := cmd.ToChartOptions(cmd.log)
	if err != nil {
		return err
	}
	chartValues, err := config.GetExtraValues(chartOptions)
	if err != nil {
		return err
	}

	// parse vCluster config
	vClusterConfig, err := parseVClusterYAML(chartValues, cmd.CreateOptions)
	if err != nil {
		return err
	}

	err = pkgconfig.ValidateSyncFromHostClasses(vClusterConfig.Sync.FromHost)
	if err != nil {
		return err
	}

	err = pkgconfig.ValidateAllSyncPatches(vClusterConfig.Sync)
	if err != nil {
		return err
	}

	err = pkgconfig.ValidateVolumeSnapshotController(vClusterConfig.Deploy.VolumeSnapshotController, vClusterConfig.PrivateNodes)
	if err != nil {
		return err
	}

	err = pkgconfig.ValidateCustomResourceSyncProxyConflicts(
		vClusterConfig.Sync.ToHost.CustomResources,
		vClusterConfig.Sync.FromHost.CustomResources,
		vClusterConfig.Experimental.Proxy.CustomResources,
	)
	if err != nil {
		return err
	}

	err = pkgconfig.ValidateExperimentalProxyCustomResourcesConfig(vClusterConfig.Experimental.Proxy.CustomResources)
	if err != nil {
		return err
	}

	warnings := pkgconfig.Lint(*vClusterConfig)
	for _, warning := range warnings {
		cmd.log.Warnf(warning)
	}

	if vClusterConfig.Sync.ToHost.Namespaces.Enabled {
		if err := namespaces.ValidateNamespaceSyncConfig(vClusterConfig, vClusterName, cmd.Namespace); err != nil {
			return err
		}
	}

	if vClusterConfig.IsConfiguredForAutoDeletion() {
		if agentDeployed, err := cmd.isLoftAgentDeployed(ctx); err != nil {
			return fmt.Errorf("is agent deployed: %w", err)
		} else if !agentDeployed {
			return fmt.Errorf("auto deletion is configured but requires an agent to be installed on the host cluster. To install the agent using the vCluster CLI, run: vcluster platform add cluster")
		}
	}

	err = validateHABackingStoreCompatibility(vClusterConfig)
	if err != nil {
		return err
	}

	verb := "created"
	if isVClusterDeployed(release) {
		verb = "upgraded"
		// While certain backing store changes are allowed we prohibit changes to another distro.
		if err := config.ValidateChanges(currentVClusterConfig, vClusterConfig); err != nil {
			return err
		}
	}

	// create platform secret
	if cmd.Add {
		err = pkgconfig.ValidatePlatformProject(ctx, vClusterConfig, cmd.LoadedConfig(cmd.log))
		if err != nil {
			return err
		}
		err = cmd.addVCluster(ctx, vClusterName, vClusterConfig)
		if err != nil {
			return err
		}
	}

	// we have to upgrade / install the chart
	err = cmd.deployChart(ctx, vClusterName, chartValues, helmBinaryPath)
	if err != nil {
		return err
	}

	// check if we should connect to the vcluster or print the kubeconfig
	if cmd.Connect || cmd.Print {
		cmd.log.Donef("Successfully %s virtual cluster %s in namespace %s", verb, vClusterName, cmd.Namespace)
		return ConnectHelm(ctx, &ConnectOptions{
			BackgroundProxy:       cmd.BackgroundProxy,
			UpdateCurrent:         true,
			Print:                 cmd.Print,
			KubeConfigContextName: cmd.KubeConfigContextName,
			KubeConfig:            "./kubeconfig.yaml",
			BackgroundProxyImage:  cmd.BackgroundProxyImage,
		}, cmd.GlobalFlags, vClusterName, nil, cmd.log)
	}

	if cmd.localCluster {
		cmd.log.Donef(
			"Successfully %s virtual cluster %s in namespace %s. \n"+
				"- Use 'vcluster connect %s --namespace %s' to access the virtual cluster",
			verb, vClusterName, cmd.Namespace, vClusterName, cmd.Namespace,
		)
	} else {
		cmd.log.Donef(
			"Successfully %s virtual cluster %s in namespace %s. \n"+
				"- Use 'vcluster connect %s --namespace %s' to access the virtual cluster\n"+
				"- Use `vcluster connect %s --namespace %s -- kubectl get ns` to run a command directly within the vcluster",
			verb, vClusterName, cmd.Namespace, vClusterName, cmd.Namespace, vClusterName, cmd.Namespace,
		)
	}

	return nil
}

var advisors = map[string]func() (warning string){
	"sleepMode":    sleepmode.Warning,
	"platform":     config.WarningPlatform,
	"autoDelete":   config.WarningAutoDelete,
	"autoSleep":    config.WarningAutoSleep,
	"autoSnapshot": config.WarningAutoSnapshot,
}

func confirmConfigIncompatibility(currentVClusterConfig *config.Config, currentValues string, log log.Logger) error {
	if err := currentVClusterConfig.UnmarshalYAMLStrict([]byte(currentValues)); err != nil {
		warning := config.ConfigStructureWarning(log, []byte(currentValues), advisors)
		if warning == "" {
			warning = "The current configuration is not compatible with the version you're upgrading to."
		}

		log.Warn(warning)
		if terminal.IsTerminalIn {
			answer, qErr := log.Question(&survey.QuestionOptions{
				Question:     "The vCluster configuration structure has changed. Features that aren't manually migrated will be lost. Would you like to proceed?",
				DefaultValue: "no",
				Options:      []string{"no", "yes, I'll update my configuration later"},
			})
			if qErr != nil {
				return qErr
			}

			if answer == "no" {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

func buildExtraValues(ctx context.Context, cmd *CreateOptions, log log.Logger) ([]string, error) {
	// build extra values
	var newExtraValues []string
	var filesToRemove []string

	// get config from snapshot
	if len(cmd.Values) == 0 && len(cmd.SetValues) == 0 {
		restoreValuesFile, err := getVClusterConfigFromSnapshot(ctx, cmd)
		if err != nil {
			log.Warnf("get vCluster config from snapshot: %w", err)
		} else if restoreValuesFile != "" {
			filesToRemove = append(filesToRemove, restoreValuesFile)
			log.Info("Using vCluster config from snapshot")
			newExtraValues = append(newExtraValues, restoreValuesFile)
		}
	} else if cmd.Restore != "" {
		log.Warnf("Skipping config from snapshot because --values or --set flag is used")
	}

	// get config from values files
	for _, value := range cmd.Values {
		// ignore decoding errors and treat it as non-base64 string
		decodedString, err := getBase64DecodedString(value)
		if err != nil {
			newExtraValues = append(newExtraValues, value)
			continue
		}

		// write the decoded string to a temp file
		tempValuesFile, err := writeTempFile([]byte(decodedString))
		if err != nil {
			return nil, fmt.Errorf("write temp values file: %w", err)
		}
		filesToRemove = append(filesToRemove, tempValuesFile)

		// setting new file to extraValues slice to process it further.
		newExtraValues = append(newExtraValues, tempValuesFile)
	}

	// resetting this as the base64 encoded strings should be removed and only valid file names should be kept.
	cmd.Values = newExtraValues
	return filesToRemove, nil
}

func parseVClusterYAML(extraValues string, cmd *CreateOptions) (*config.Config, error) {
	finalValues, err := mergeAllValues(cmd.SetValues, cmd.Values, extraValues)
	if err != nil {
		return nil, fmt.Errorf("merge values: %w", err)
	}

	// parse config
	vClusterConfig := &config.Config{}
	if err := vClusterConfig.UnmarshalYAMLStrict([]byte(finalValues)); err != nil {
		return nil, fmt.Errorf("merge values: %w", err)
	}

	return vClusterConfig, nil
}

func (cmd *createHelm) addVCluster(ctx context.Context, name string, vClusterConfig *config.Config) error {
	platformConfig := vClusterConfig.GetPlatformConfig()
	if platformConfig.APIKey.SecretName != "" || platformConfig.APIKey.Namespace != "" {
		return nil
	}

	_, err := platform.InitClientFromConfig(ctx, cmd.LoadedConfig(cmd.log))
	if err != nil {
		if vClusterConfig.IsProFeatureEnabled() {
			return fmt.Errorf("you have vCluster pro features enabled, but seems like you are not logged in (%w). Please make sure to log into vCluster Platform to use vCluster pro features or run this command with --add=false", err)
		}

		cmd.log.Debugf("create platform client: %v", err)
		return nil
	}

	err = platform.ApplyPlatformSecret(ctx, cmd.LoadedConfig(cmd.log), cmd.kubeClient, "", name, cmd.Namespace, cmd.Project, "", "", false, cmd.LoadedConfig(cmd.log).Platform.CertificateAuthorityData, cmd.log)
	if err != nil {
		return fmt.Errorf("apply platform secret: %w", err)
	}

	return nil
}

func (cmd *createHelm) isLoftAgentDeployed(ctx context.Context) (bool, error) {
	podList, err := cmd.kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		LabelSelector: "app=loft",
	})
	if err != nil && !kerrors.IsNotFound(err) {
		return false, err
	} else if podList == nil {
		return false, errors.New("nil podList")
	}

	return len(podList.Items) > 0, nil
}

func isVClusterDeployed(release *helm.Release) bool {
	return release != nil &&
		release.Chart != nil &&
		release.Chart.Metadata != nil &&
		(release.Chart.Metadata.Name == "vcluster" || release.Chart.Metadata.Name == "vcluster-k8s" ||
			release.Chart.Metadata.Name == "vcluster-eks") &&
		release.Secret != nil &&
		release.Secret.Labels != nil &&
		release.Secret.Labels["status"] == "deployed"
}

func validateHABackingStoreCompatibility(config *config.Config) error {
	if !config.EmbeddedDatabase() {
		return nil
	}
	if !(config.ControlPlane.StatefulSet.HighAvailability.Replicas > 1) {
		return nil
	}
	return fmt.Errorf("cannot use default embedded database (sqlite) in high availability mode. Try embedded etcd backing store instead")
}

// helmValuesYAML returns the extraValues from the helm release in yaml format.
// If the extra values in the chart are nil it returns an empty string.
func helmExtraValuesYAML(release *helm.Release) (string, error) {
	if release == nil || release.Config == nil {
		return "", nil
	}
	extraValues, err := yaml.Marshal(release.Config)
	if err != nil {
		return "", err
	}

	return string(extraValues), nil
}

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

	// if restore we deploy a resource quota to prevent other pods from starting
	if cmd.Restore != "" {
		// this is required or otherwise vCluster pods would start which we don't want when restoring
		_, err = cmd.kubeClient.CoreV1().ResourceQuotas(cmd.Namespace).Create(ctx, &corev1.ResourceQuota{
			ObjectMeta: metav1.ObjectMeta{
				Name:      RestoreResourceQuota,
				Namespace: cmd.Namespace,
			},
			Spec: corev1.ResourceQuotaSpec{
				Hard: corev1.ResourceList{
					corev1.ResourcePods: resource.MustParse("0"),
				},
			},
		}, metav1.CreateOptions{})
		if err != nil && !kerrors.IsAlreadyExists(err) {
			return fmt.Errorf("unable to create vcluster restore resource quota: %w", err)
		}

		// create interrupt channel
		sigint := make(chan os.Signal, 1)
		defer func() {
			// make sure we won't interfere with interrupts anymore
			signal.Stop(sigint)

			// delete the resource quota when we are done
			_ = cmd.kubeClient.CoreV1().ResourceQuotas(cmd.Namespace).Delete(ctx, RestoreResourceQuota, metav1.DeleteOptions{})
		}()

		// also delete on interrupt
		go func() {
			// interrupt signal sent from terminal
			signal.Notify(sigint, os.Interrupt)
			// sigterm signal sent from kubernetes
			signal.Notify(sigint, syscall.SIGTERM)

			// wait until we get killed
			<-sigint

			// cleanup resource quota
			_ = cmd.kubeClient.CoreV1().ResourceQuotas(cmd.Namespace).Delete(ctx, RestoreResourceQuota, metav1.DeleteOptions{})
			os.Exit(1)
		}()
	}

	// we have to upgrade / install the chart
	helmClient := helm.NewClient(&cmd.rawConfig, cmd.log, helmExecutablePath)
	err = helmClient.Upgrade(ctx, vClusterName, cmd.Namespace, helm.UpgradeOptions{
		CreateNamespace: cmd.CreateNamespace,
		Chart:           cmd.ChartName,
		Repo:            cmd.ChartRepo,
		Version:         cmd.ChartVersion,
		Path:            cmd.LocalChartDir,
		Values:          chartValues,
		ValuesFiles:     cmd.Values,
		SetValues:       cmd.SetValues,
		Debug:           cmd.Debug,
	})
	if err != nil {
		return err
	}

	// now restore if wanted
	if cmd.Restore != "" {
		cmd.log.Infof("Restore vCluster %s...", vClusterName)
		err = Restore(ctx, []string{vClusterName, cmd.Restore}, cmd.GlobalFlags, &snapshot.Options{}, &pod.Options{}, true, false, cmd.log)
		if err != nil {
			// delete the vcluster if the restore failed
			deleteErr := helmClient.Delete(vClusterName, cmd.Namespace)
			if deleteErr != nil {
				cmd.log.Errorf("Failed to delete vcluster %s: %v", vClusterName, deleteErr)
			}

			return fmt.Errorf("restore vCluster %s: %w", vClusterName, err)
		}
	}

	return nil
}

func (cmd *createHelm) ToChartOptions(log log.Logger) (*config.ExtraValuesOptions, error) {
	// check if we should create with node port
	clusterType := localkubernetes.DetectClusterType(&cmd.rawConfig)
	if cmd.ExposeLocal && clusterType.LocalKubernetes() && clusterType != localkubernetes.ClusterTypeOrbstack {
		cmd.log.Infof("Detected local kubernetes cluster %s. Will deploy vcluster with a NodePort", clusterType)
		cmd.localCluster = true
	}

	cfg := cmd.LoadedConfig(log)
	return &config.ExtraValuesOptions{
		Expose:              cmd.Expose,
		NodePort:            cmd.localCluster,
		DisableTelemetry:    cfg.TelemetryDisabled,
		InstanceCreatorType: "vclusterctl",
		MachineID:           telemetry.GetMachineID(cfg),
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

func writeTempFile(data []byte) (string, error) {
	// write a temporary values file
	tempFile, err := os.CreateTemp("", "")
	tempValuesFile := tempFile.Name()
	if err != nil {
		return "", fmt.Errorf("create temp values file: %w", err)
	}

	_, err = tempFile.Write(data)
	if err != nil {
		_ = os.Remove(tempValuesFile)
		return "", fmt.Errorf("write values to temp values file: %w", err)
	}

	err = tempFile.Close()
	if err != nil {
		_ = os.Remove(tempValuesFile)
		return "", fmt.Errorf("close temp values file: %w", err)
	}

	return tempValuesFile, nil
}

func getVClusterConfigFromSnapshot(ctx context.Context, cmd *CreateOptions) (string, error) {
	if cmd.Restore == "" {
		return "", nil
	}

	snapshotOptions := &snapshot.Options{}
	err := snapshot.Parse(cmd.Restore, snapshotOptions)
	if err != nil {
		return "", fmt.Errorf("parse snapshot: %w", err)
	}

	objectStore, err := snapshot.CreateStore(ctx, snapshotOptions)
	if err != nil {
		return "", fmt.Errorf("create snapshot store: %w", err)
	}

	reader, err := objectStore.GetObject(ctx)
	if err != nil {
		return "", fmt.Errorf("get snapshot object: %w", err)
	}

	// read the first tar entry
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return "", fmt.Errorf("create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// create a new tar reader
	tarReader := tar.NewReader(gzipReader)

	// read the vCluster config
	header, err := tarReader.Next()
	if err != nil {
		return "", err
	}

	buf := &bytes.Buffer{}
	_, err = io.Copy(buf, tarReader)
	if err != nil {
		return "", err
	}

	// no vCluster config in the snapshot
	if header.Name != snapshot.SnapshotReleaseKey {
		return "", nil
	}

	// unmarshal the release
	release := &snapshot.HelmRelease{}
	err = json.Unmarshal(buf.Bytes(), release)
	if err != nil {
		return "", fmt.Errorf("unmarshal vCluster release: %w", err)
	}

	// set chart version
	if release.ChartVersion != "" && (cmd.ChartVersion == "" || cmd.ChartVersion == upgrade.GetVersion()) {
		cmd.ChartVersion = release.ChartVersion
	}

	// write the values to a temp file
	if len(release.Values) > 0 {
		return writeTempFile(release.Values)
	}

	return "", nil
}

func getConfigfileFromSecret(ctx context.Context, name, namespace string) (*config.Config, error) {
	secretName := "vc-config-" + name

	kConf := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})
	clientConfig, err := kConf.ClientConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return nil, err
	}

	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	configBytes, ok := secret.Data["config.yaml"]
	if !ok {
		return nil, fmt.Errorf("secret %s in namespace %s does not contain the expected 'config.yaml' field", secretName, namespace)
	}

	config := config.Config{}
	err = yaml.Unmarshal(configBytes, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
