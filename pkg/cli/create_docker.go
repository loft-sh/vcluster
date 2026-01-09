package cli

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/oci"
	vclusterconfig "github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/random"
	"github.com/loft-sh/vcluster/pkg/telemetry"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/loft-sh/vcluster/pkg/util/clihelper"
	"github.com/samber/lo"
	"golang.org/x/mod/semver"
)

var containerVolumes = map[string]string{
	"var":     "/var",
	"etc":     "/etc",
	"bin":     "/usr/local/bin",
	"cni-bin": "/opt/cni/bin",
}

func CreateDocker(ctx context.Context, options *CreateOptions, globalFlags *flags.GlobalFlags, vClusterName string, log log.Logger) error {
	// make sure we deploy the correct version
	vClusterVersion := strings.TrimPrefix(options.ChartVersion, "v")
	if vClusterVersion == upgrade.DevelopmentVersion {
		return fmt.Errorf("please specify a vCluster version via --chart-version")
	}

	// if the vCluster version is older than 0.31.0-alpha.0 return an error
	if semver.Compare("v"+vClusterVersion, "v0.31.0-alpha.0") == -1 {
		return fmt.Errorf("please use a newer version of vCluster, the minimum version is v0.31.0-alpha.0")
	}

	// check if container exists
	exists, err := containerExists(ctx, getControlPlaneContainerName(vClusterName))
	if err != nil {
		return fmt.Errorf("failed to check if container exists: %w", err)
	} else if exists {
		if !options.Upgrade {
			return fmt.Errorf("vCluster %s already exists, use --upgrade to upgrade it", vClusterName)
		}

		log.Infof("vCluster container %s already exists, recreating it...", vClusterName)

		// stop and delete the container
		err = stopContainer(ctx, getControlPlaneContainerName(vClusterName))
		if err != nil {
			return fmt.Errorf("failed to stop container: %w", err)
		}
	}

	// build extra values
	filesToRemove, err := buildExtraValues(ctx, options, log)
	if err != nil {
		return err
	}
	defer func() {
		for _, file := range filesToRemove {
			os.Remove(file)
		}
	}()

	dockerOptions, err := toDockerOptions(globalFlags, log)
	if err != nil {
		return err
	}
	extraValues, err := config.GetExtraValues(dockerOptions)
	if err != nil {
		return err
	}

	// parse vCluster config
	finalValues, err := mergeAllValues(options.SetValues, options.Values, extraValues)
	if err != nil {
		return fmt.Errorf("merge values: %w", err)
	}

	// parse config
	vConfig := &config.Config{}
	if err := vConfig.UnmarshalYAMLStrict([]byte(finalValues)); err != nil {
		return fmt.Errorf("unmarshal vcluster config: %w", err)
	}

	// pull the kubernetes image and folder structure looks roughly like this:
	// - etcd
	// - etcdctl
	// - helm
	// - kine
	// - konnectivity-server
	// - kube-apiserver
	// - kube-controller-manager
	// - kube-scheduler
	// - kubernetes-v1.34.0-amd64.tar.gz
	// - kubernetes-v1.34.0-arm64.tar.gz
	kubernetesDir, kubernetesVersion, err := pullKubernetesImage(ctx, vConfig, globalFlags, log)
	if err != nil {
		return err
	}

	// pull the vcluster image and folder structure looks roughly like this:
	// - vcluster
	vClusterDir, err := pullVClusterImage(ctx, vClusterVersion, globalFlags, log)
	if err != nil {
		return err
	}

	// add the platform credentials to the docker container
	extraArgs := []string{
		"--vcluster-name", vClusterName,
	}
	if options.Add && !exists {
		err := vclusterconfig.ValidatePlatformProject(ctx, vConfig, globalFlags.LoadedConfig(log))
		if err != nil {
			return err
		}

		platformArgs, err := addVClusterDocker(ctx, vClusterName, vConfig, options, globalFlags, log)
		if err != nil {
			return err
		}

		extraArgs = append(extraArgs, platformArgs...)
	}

	// write the vcluster.yaml
	vClusterYAMLPath, err := writeVClusterYAML(globalFlags, vClusterName, finalValues)
	if err != nil {
		return err
	}

	// now remove the container if it exists
	if exists {
		err = removeContainer(ctx, getControlPlaneContainerName(vClusterName))
		if err != nil {
			return fmt.Errorf("failed to remove container: %w", err)
		}
	}

	// create the docker network
	if !exists {
		err = createNetwork(ctx, vClusterName, log)
		if err != nil {
			return fmt.Errorf("failed to create network: %w", err)
		}
	}

	// ensure the join token
	vClusterJoinToken, err := ensureVClusterJoinToken(globalFlags, vClusterName)
	if err != nil {
		return fmt.Errorf("failed to ensure join token: %w", err)
	}
	extraArgs = append(extraArgs, "--join-token", vClusterJoinToken)

	// run the docker container
	err = runControlPlaneContainer(ctx, kubernetesDir, vClusterDir, vClusterYAMLPath, vClusterName, vConfig, log)
	if err != nil {
		return err
	}

	// install vCluster standalone
	if !exists {
		err = installVClusterStandalone(ctx, vClusterName, vClusterVersion, extraArgs, log)
		if err != nil {
			return err
		}
	}

	// ensure the nodes
	err = ensureVClusterNodes(ctx, kubernetesDir, vClusterName, vClusterJoinToken, kubernetesVersion, vConfig, log)
	if err != nil {
		return fmt.Errorf("failed to ensure vCluster nodes: %w", err)
	}

	// check if we should connect to the vcluster or print the kubeconfig
	if options.Connect || options.Print {
		log.Donef("Successfully created virtual cluster %s", vClusterName)
		return ConnectDocker(ctx, &ConnectOptions{
			UpdateCurrent: true,
			Print:         options.Print,
		}, globalFlags, vClusterName, nil, log)
	}

	if !exists {
		log.Donef(
			"Successfully created virtual cluster %s. \n"+
				"- Use 'vcluster connect %s' to access the virtual cluster\n"+
				"- Use `vcluster connect %s -- kubectl get ns` to run a command directly within the vcluster",
			vClusterName, vClusterName, vClusterName,
		)
	} else {
		log.Donef(
			"Successfully upgraded virtual cluster %s. \n"+
				"- Use 'vcluster connect %s' to access the virtual cluster\n"+
				"- Use `vcluster connect %s -- kubectl get ns` to run a command directly within the vcluster",
			vClusterName, vClusterName, vClusterName,
		)
	}
	return nil
}

func writeVClusterYAML(globalFlags *flags.GlobalFlags, vClusterName string, finalValues string) (string, error) {
	vClusterYAMLPath := filepath.Join(filepath.Dir(globalFlags.Config), "docker", "vclusters", vClusterName, "vcluster.yaml")
	err := os.MkdirAll(filepath.Dir(vClusterYAMLPath), 0755)
	if err != nil {
		return "", fmt.Errorf("create directory: %w", err)
	}

	err = os.WriteFile(vClusterYAMLPath, []byte(finalValues), 0644)
	if err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return vClusterYAMLPath, nil
}

func addVClusterDocker(ctx context.Context, name string, vClusterConfig *config.Config, options *CreateOptions, globalFlags *flags.GlobalFlags, log log.Logger) ([]string, error) {
	platformConfig, err := vClusterConfig.GetPlatformConfig()
	if err != nil {
		return nil, fmt.Errorf("get platform config: %w", err)
	} else if platformConfig.APIKey.SecretName != "" || platformConfig.APIKey.Namespace != "" {
		return nil, nil
	}

	platformClient, err := platform.InitClientFromConfig(ctx, globalFlags.LoadedConfig(log))
	if err != nil {
		if vClusterConfig.IsProFeatureEnabled() {
			return nil, fmt.Errorf("you have vCluster pro features enabled, but seems like you are not logged in (%w). Please make sure to log into vCluster Platform to use vCluster pro features or run this command with --add=false", err)
		}

		log.Debugf("create platform client: %v", err)
		return nil, nil
	}

	// set host
	host := strings.TrimPrefix(platformClient.Config().Platform.Host, "https://")
	project := options.Project
	if project == "" {
		project = platformConfig.Project
	}
	if project == "" {
		project = "default"
	}

	// get management client
	managementClient, err := platformClient.Management()
	if err != nil {
		return nil, fmt.Errorf("error getting management client: %w", err)
	}

	// try with the regular name first
	created, accessKey, createdName, err := platform.CreateWithName(ctx, managementClient, project, name)
	if err != nil {
		return nil, fmt.Errorf("error creating platform secret: %w", err)
	} else if !created {
		return nil, fmt.Errorf("couldn't create virtual cluster instance, name %s already exists", name)
	}

	// build the extra args
	retArgs := []string{
		"--platform-access-key", accessKey,
		"--platform-host", host,
		"--platform-insecure",
		"--platform-project", project,
		"--platform-instance-name", createdName,
	}

	return retArgs, nil
}

func installVClusterStandalone(ctx context.Context, vClusterName, vClusterVersion string, extraArgs []string, log log.Logger) error {
	log.Infof("Starting vCluster standalone %s", vClusterName)
	joinedArgs := strings.Join(extraArgs, " ")
	args := []string{
		"exec", getControlPlaneContainerName(vClusterName),
		"bash", "-c", fmt.Sprintf(`set -e -o pipefail; curl -sfLk "https://github.com/loft-sh/vcluster/releases/download/v%s/install-standalone.sh" | sh -s -- --skip-download --skip-wait %s`, vClusterVersion, joinedArgs),
	}

	log.Debugf("Running command: docker %s", strings.Join(args, " "))
	out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start vCluster standalone: %w: %s", err, string(out))
	}

	log.Debugf("Output: %s", string(out))
	return nil
}

func runControlPlaneContainer(ctx context.Context, kubernetesDir, vClusterDir, vClusterYAMLPath, vClusterName string, config *config.Config, log log.Logger) error {
	args := []string{
		"run",
		"-d",
		"-h", vClusterName,
		"--tmpfs", "/run",
		"--tmpfs", "/tmp",
		"--privileged",
		"--network", getNetworkName(vClusterName),
		"-e", "VCLUSTER_NAME=" + vClusterName,
		"-p", fmt.Sprintf("%d:8443", clihelper.RandomPort()),
		"--name", getControlPlaneContainerName(vClusterName),
	}
	for volumeName, volumePath := range containerVolumes {
		args = append(args, "-v", getControlPlaneVolumeName(vClusterName, volumeName)+":"+volumePath)
	}
	args = append(args, config.Experimental.Docker.Args...)

	// add the ports and volumes
	for _, port := range config.Experimental.Docker.Ports {
		args = append(args, "-p", port)
	}
	for _, volume := range config.Experimental.Docker.Volumes {
		args = append(args, "-v", volume)
	}
	for _, env := range config.Experimental.Docker.Env {
		args = append(args, "-e", env)
	}

	// create a bind mount for every file in the kubernetes and vcluster directories
	entries, err := os.ReadDir(kubernetesDir)
	if err != nil {
		return fmt.Errorf("read kubernetes directory: %w", err)
	}
	for _, entry := range entries {
		args = append(args, "--mount", fmt.Sprintf("type=bind,src=%s/%s,dst=/var/lib/vcluster/bin/%s,ro", kubernetesDir, entry.Name(), entry.Name()))
	}
	args = append(args, "--mount", fmt.Sprintf("type=bind,src=%s/vcluster,dst=/var/lib/vcluster/bin/vcluster,ro", vClusterDir))
	args = append(args, "--mount", fmt.Sprintf("type=bind,src=%s,dst=/etc/vcluster/vcluster.yaml,ro", vClusterYAMLPath))

	// add the image to start
	image := "ghcr.io/loft-sh/vm-container"
	if config.Experimental.Docker.Image != "" {
		image = config.Experimental.Docker.Image
	}
	args = append(args, image)

	// start the docker container
	log.Debugf("Running command: docker %s", strings.Join(args, " "))
	out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start docker container: %w: %s", err, string(out))
	}
	return nil
}

func ensureVClusterJoinToken(globalFlags *flags.GlobalFlags, vClusterName string) (string, error) {
	tokenPath := filepath.Join(filepath.Dir(globalFlags.Config), "docker", "vclusters", vClusterName, "token.txt")
	_, err := os.Stat(tokenPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("ensure token file: %w", err)
		}

		token := random.String(64)
		err = os.WriteFile(tokenPath, []byte(token), 0644)
		if err != nil {
			return "", fmt.Errorf("write token file: %w", err)
		}
		return token, nil
	}

	token, err := os.ReadFile(tokenPath)
	if err != nil {
		return "", fmt.Errorf("read token file: %w", err)
	}

	return string(token), nil
}

func runWorkerContainer(ctx context.Context, kubernetesDir, vClusterName string, workerConfig *config.ExperimentalDockerNode, log log.Logger) error {
	args := []string{
		"run",
		"-d",
		"-h", workerConfig.Name,
		"--tmpfs", "/run",
		"--tmpfs", "/tmp",
		"--privileged",
		"--network", getNetworkName(vClusterName),
		"--name", getWorkerContainerName(vClusterName, workerConfig.Name),
	}
	for volumeName, volumePath := range containerVolumes {
		args = append(args, "-v", getWorkerVolumeName(vClusterName, workerConfig.Name, volumeName)+":"+volumePath)
	}
	args = append(args, workerConfig.Args...)

	// add the ports and volumes
	for _, port := range workerConfig.Ports {
		args = append(args, "-p", port)
	}
	for _, volume := range workerConfig.Volumes {
		args = append(args, "-v", volume)
	}
	for _, env := range workerConfig.Env {
		args = append(args, "-e", env)
	}

	// create a bind mount for every file in the kubernetes and vcluster directories
	entries, err := os.ReadDir(kubernetesDir)
	if err != nil {
		return fmt.Errorf("read kubernetes directory: %w", err)
	}
	for _, entry := range entries {
		args = append(args, "--mount", fmt.Sprintf("type=bind,src=%s/%s,dst=/var/lib/vcluster/bin/%s,ro", kubernetesDir, entry.Name(), entry.Name()))
	}

	// add the image to start
	image := "ghcr.io/loft-sh/vm-container"
	if workerConfig.Image != "" {
		image = workerConfig.Image
	}
	args = append(args, image)

	// start the docker container
	log.Debugf("Running command: docker %s", strings.Join(args, " "))
	out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start docker container: %w: %s", err, string(out))
	}
	return nil
}

func joinVClusterNodeContainer(ctx context.Context, vClusterName, workerName, vClusterJoinToken, kubernetesVersion string, log log.Logger) error {
	log.Infof("Joining node %s to vCluster %s...", workerName, vClusterName)

	// Define retry logic: try every 2 seconds until a 200 OK is received.
	// --retry-all-errors is used for curl versions that support it,
	// but a manual 'until' loop is more portable across container OS versions.
	joinScript := fmt.Sprintf(`
until curl -fsSLk -o /tmp/join.sh "https://%s:8443/node/join?token=%s&type=worker"; do
  echo "Waiting for vCluster API to be ready..."
  sleep 2
done
sh /tmp/join.sh --bundle-path /var/lib/vcluster/bin/kubernetes-%s-%s.tar.gz --force-join
`, vClusterName, url.QueryEscape(vClusterJoinToken), kubernetesVersion, runtime.GOARCH)
	args := []string{"exec", getWorkerContainerName(vClusterName, workerName), "bash", "-c", joinScript}

	log.Debugf("Running command: docker %s", strings.Join(args, " "))
	out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start vCluster standalone: %w: %s", err, string(out))
	}

	log.Debugf("Output: %s", string(out))
	return nil
}

func ensureVClusterNodes(ctx context.Context, kubernetesDir, vClusterName, vClusterJoinToken, kubernetesVersion string, vClusterConfig *config.Config, log log.Logger) error {
	nodes, err := findDockerVClusterNodes(ctx, vClusterName)
	if err != nil {
		return fmt.Errorf("failed to find vCluster nodes: %w", err)
	}

	// remove the nodes that are not in the config
	for _, node := range nodes {
		_, found := lo.Find(vClusterConfig.Experimental.Docker.Nodes, func(n config.ExperimentalDockerNode) bool {
			return node.Name == n.Name
		})
		if !found {
			log.Infof("Removing node %s from vCluster %s", node.Name, vClusterName)
			err = stopContainer(ctx, getWorkerContainerName(vClusterName, node.Name))
			if err != nil {
				return fmt.Errorf("failed to stop vCluster node: %w", err)
			}
			err = removeContainer(ctx, getWorkerContainerName(vClusterName, node.Name))
			if err != nil {
				return fmt.Errorf("failed to remove vCluster node: %w", err)
			}
			for volumeName := range containerVolumes {
				err = removeVolume(ctx, getWorkerVolumeName(vClusterName, node.Name, volumeName))
				if err != nil {
					return fmt.Errorf("failed to remove vCluster node volume: %w", err)
				}
			}
		}
	}

	// if there are no nodes to add, return
	if len(vClusterConfig.Experimental.Docker.Nodes) == 0 {
		return nil
	}

	// add the nodes that are not in the config
	for _, node := range vClusterConfig.Experimental.Docker.Nodes {
		_, found := lo.Find(nodes, func(n dockerVCluster) bool {
			return node.Name == n.Name
		})
		if !found {
			if node.Name == "" {
				return fmt.Errorf("node name is required")
			}

			log.Infof("Adding node %s to vCluster %s", node.Name, vClusterName)
			err = runWorkerContainer(ctx, kubernetesDir, vClusterName, &node, log)
			if err != nil {
				return fmt.Errorf("failed to run vCluster node: %w", err)
			}
			err = joinVClusterNodeContainer(ctx, vClusterName, node.Name, vClusterJoinToken, kubernetesVersion, log)
			if err != nil {
				return fmt.Errorf("failed to join vCluster node: %w", err)
			}
		}
	}

	return nil
}

func pullVClusterImage(ctx context.Context, vClusterVersion string, globalFlags *flags.GlobalFlags, log log.Logger) (string, error) {
	fullImage := "ghcr.io/loft-sh/vcluster-pro:" + vClusterVersion

	// get the target directory
	targetDir := filepath.Join(filepath.Dir(globalFlags.Config), "docker", "vcluster", vClusterVersion)
	_, err := os.Stat(targetDir)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("stat target directory: %w", err)
	} else if err == nil {
		return targetDir, nil
	}

	// create the target directory
	err = os.MkdirAll(targetDir, 0755)
	if err != nil {
		return "", fmt.Errorf("create target directory: %w", err)
	}

	// create a temp directory
	tempDir, err := os.MkdirTemp("", "vcluster-upgrade-")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// pull the image
	log.Infof("Pulling image %s to %s...", fullImage, tempDir)
	err = oci.PullImage(ctx, fullImage, tempDir, nil)
	if err != nil {
		return "", fmt.Errorf("pull image: %w", err)
	}

	// extract the image
	log.Infof("Extracting vcluster binary from %s to %s...", fullImage, targetDir)
	err = oci.ExtractFile(tempDir, "/vcluster", filepath.Join(targetDir, "vcluster"))
	if err != nil {
		_ = os.RemoveAll(targetDir)
		return "", fmt.Errorf("extract image: %w", err)
	}

	return targetDir, nil
}

func pullKubernetesImage(ctx context.Context, vConfig *config.Config, globalFlags *flags.GlobalFlags, log log.Logger) (string, string, error) {
	// get the kubernetes version
	kubernetesVersion := ""
	if vConfig.ControlPlane.Distro.K8S.Version != "" {
		kubernetesVersion = vConfig.ControlPlane.Distro.K8S.Version
	} else {
		kubernetesVersion = vConfig.ControlPlane.Distro.K8S.Image.Tag
	}
	if kubernetesVersion == "" {
		defaultConfig, err := config.NewDefaultConfig()
		if err != nil {
			return "", "", err
		}
		kubernetesVersion = defaultConfig.ControlPlane.Distro.K8S.Image.Tag
	}

	fullImage := "ghcr.io/loft-sh/kubernetes:" + kubernetesVersion + "-full"

	// get the target directory
	targetDir := filepath.Join(filepath.Dir(globalFlags.Config), "docker", "kubernetes", kubernetesVersion)
	_, err := os.Stat(targetDir)
	if err != nil && !os.IsNotExist(err) {
		return "", "", fmt.Errorf("stat target directory: %w", err)
	} else if err == nil {
		return targetDir, kubernetesVersion, nil
	}

	// create a temp directory
	tempDir, err := os.MkdirTemp("", "vcluster-docker-kubernetes-")
	if err != nil {
		return "", "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// create the target directory
	err = os.MkdirAll(targetDir, 0755)
	if err != nil {
		return "", "", fmt.Errorf("create target directory: %w", err)
	}

	// pull the image
	log.Infof("Pulling image %s to %s...", fullImage, tempDir)
	err = oci.PullImage(ctx, fullImage, tempDir, nil)
	if err != nil {
		_ = os.RemoveAll(targetDir)
		return "", "", fmt.Errorf("pull image: %w", err)
	}

	// extract the image
	log.Infof("Extracting image %s to %s...", fullImage, targetDir)
	err = oci.Extract(tempDir, "/kubernetes", targetDir)
	if err != nil {
		_ = os.RemoveAll(targetDir)
		return "", "", fmt.Errorf("extract image: %w", err)
	}

	return targetDir, kubernetesVersion, nil
}

func toDockerOptions(globalFlags *flags.GlobalFlags, log log.Logger) (*config.ExtraValuesOptions, error) {
	cfg := globalFlags.LoadedConfig(log)
	return &config.ExtraValuesOptions{
		DisableTelemetry:    cfg.TelemetryDisabled,
		InstanceCreatorType: "vclusterctl",
		MachineID:           telemetry.GetMachineID(cfg),
	}, nil
}

func createNetwork(ctx context.Context, vClusterName string, log log.Logger) error {
	log.Infof("Creating network %s...", getNetworkName(vClusterName))
	args := []string{"network", "create", getNetworkName(vClusterName)}
	log.Debugf("Running command: docker %s", strings.Join(args, " "))
	out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil && !strings.HasSuffix(strings.TrimSpace(string(out)), "already exists") {
		return fmt.Errorf("failed to create network: %w: %s", err, string(out))
	}
	return nil
}

func deleteNetwork(ctx context.Context, vClusterName string, log log.Logger) error {
	args := []string{"network", "rm", getNetworkName(vClusterName)}
	log.Debugf("Running command: docker %s", strings.Join(args, " "))
	out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete network: %w: %s", err, string(out))
	}
	return nil
}

func getNetworkName(vClusterName string) string {
	return "vcluster-" + vClusterName
}

func getControlPlaneContainerName(vClusterName string) string {
	return "vcluster-docker." + vClusterName
}

func getControlPlaneVolumeName(vClusterName, volumeName string) string {
	return "vcluster-docker." + vClusterName + "." + volumeName
}

func getWorkerContainerName(vClusterName, workerName string) string {
	return "vcluster-docker-worker." + vClusterName + "." + workerName
}

func getWorkerVolumeName(vClusterName, workerName, volumeName string) string {
	return "vcluster-docker-worker." + vClusterName + "." + workerName + "." + volumeName
}
