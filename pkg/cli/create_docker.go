package cli

import (
	"bufio"
	"context"
	"encoding/json"
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
	"github.com/loft-sh/vcluster/pkg/strvals"
	"github.com/loft-sh/vcluster/pkg/telemetry"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/loft-sh/vcluster/pkg/util/clihelper"
	"github.com/samber/lo"
	"golang.org/x/mod/semver"
	"sigs.k8s.io/yaml"
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

	// load the docker config
	vConfig, finalValues, extraDockerArgs, err := loadConfig(ctx, options, globalFlags, log)
	if err != nil {
		return fmt.Errorf("failed to load docker config: %w", err)
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
	vClusterBinaryDir, err := pullVClusterImage(ctx, vClusterVersion, globalFlags, log)
	if err != nil {
		return err
	}

	// add the platform credentials to the docker container
	extraVClusterArgs := []string{
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

		extraVClusterArgs = append(extraVClusterArgs, platformArgs...)
	}

	// write the vcluster.yaml
	vClusterConfigDir, err := writeVClusterYAML(globalFlags, vClusterName, finalValues)
	if err != nil {
		return err
	}

	// ensure the k8s resolv conf file
	err = ensureK8sResolvConf(ctx, globalFlags, vClusterName)
	if err != nil {
		return fmt.Errorf("failed to ensure k8s resolv conf file: %w", err)
	}

	// now remove the container if it exists
	if exists {
		err = removeContainer(ctx, getControlPlaneContainerName(vClusterName))
		if err != nil {
			return fmt.Errorf("failed to remove container: %w", err)
		}
	}

	// get the network name
	networkName := getNetworkName(vClusterName)
	if vConfig.Experimental.Docker.Network != "" {
		networkName = vConfig.Experimental.Docker.Network
	}

	// create the docker network
	if !exists {
		err = createNetwork(ctx, networkName, log)
		if err != nil {
			return fmt.Errorf("failed to create network: %w", err)
		}
	}

	// ensure the join token
	vClusterJoinToken, err := ensureVClusterJoinToken(globalFlags, vClusterName)
	if err != nil {
		return fmt.Errorf("failed to ensure join token: %w", err)
	}
	extraVClusterArgs = append(extraVClusterArgs, "--join-token", vClusterJoinToken)

	// run the docker container
	err = runControlPlaneContainer(ctx, kubernetesDir, vClusterBinaryDir, vClusterConfigDir, vClusterName, networkName, vConfig, extraDockerArgs, log)
	if err != nil {
		return err
	}

	// install vCluster standalone
	if !exists {
		err = installVClusterStandalone(ctx, vClusterName, vClusterVersion, extraVClusterArgs, log)
		if err != nil {
			return err
		}
	}

	// ensure the nodes
	err = ensureVClusterNodes(ctx, kubernetesDir, vClusterConfigDir, vClusterName, networkName, vClusterJoinToken, kubernetesVersion, vConfig, log)
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

	return filepath.Dir(vClusterYAMLPath), nil
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
		return nil, fmt.Errorf("error creating platform access key: %w. If you don't want to use the platform, run this command with --add=false or run 'vcluster logout'", err)
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

func runControlPlaneContainer(ctx context.Context, kubernetesDir, vClusterBinaryDir, vClusterConfigDir, vClusterName, networkName string, config *config.Config, extraArgs []string, log log.Logger) error {
	args := []string{
		"run",
		"-d",
		"-h", vClusterName,
		"--tmpfs", "/run",
		"--tmpfs", "/tmp",
		"--privileged",
		"--network", networkName,
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
	args = append(args, "--mount", fmt.Sprintf("type=bind,src=%s/vcluster,dst=/var/lib/vcluster/bin/vcluster,ro", vClusterBinaryDir))
	args = append(args, "--mount", fmt.Sprintf("type=bind,src=%s/vcluster.yaml,dst=/etc/vcluster/vcluster.yaml,ro", vClusterConfigDir))
	args = append(args, "--mount", fmt.Sprintf("type=bind,src=%s/k8s-resolv.conf,dst=/etc/k8s-resolv.conf,ro", vClusterConfigDir))
	args = append(args, extraArgs...)

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

func ensureK8sResolvConf(ctx context.Context, globalFlags *flags.GlobalFlags, vClusterName string) error {
	resolvConf := filepath.Join(filepath.Dir(globalFlags.Config), "docker", "vclusters", vClusterName, "k8s-resolv.conf")
	_, err := os.Stat(resolvConf)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("ensure k8s resolv conf file: %w", err)
		}

		hostDNSServer, err := getHostDNSServer(ctx)
		if err != nil {
			return fmt.Errorf("failed to get dns docker internal ip: %w", err)
		}

		resolvConfContent := fmt.Sprintf("# Custom Kubelet DNS resolver\nnameserver %s\noptions ndots:0\n", hostDNSServer)
		err = os.WriteFile(resolvConf, []byte(resolvConfContent), 0644)
		if err != nil {
			return fmt.Errorf("write k8s resolv conf file: %w", err)
		}
		return nil
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

func runWorkerContainer(ctx context.Context, kubernetesDir, vClusterConfigDir, vClusterName, networkName string, workerConfig *config.ExperimentalDockerNode, log log.Logger) error {
	args := []string{
		"run",
		"-d",
		"-h", workerConfig.Name,
		"--tmpfs", "/run",
		"--tmpfs", "/tmp",
		"--privileged",
		"--network", networkName,
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
	args = append(args, "--mount", fmt.Sprintf("type=bind,src=%s/k8s-resolv.conf,dst=/etc/k8s-resolv.conf,ro", vClusterConfigDir))

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

func ensureVClusterNodes(ctx context.Context, kubernetesDir, vClusterConfigDir, vClusterName, networkName, vClusterJoinToken, kubernetesVersion string, vClusterConfig *config.Config, log log.Logger) error {
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
			err = runWorkerContainer(ctx, kubernetesDir, vClusterConfigDir, vClusterName, networkName, &node, log)
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

func loadConfig(ctx context.Context, options *CreateOptions, globalFlags *flags.GlobalFlags, log log.Logger) (*config.Config, string, []string, error) {
	defaultConfig, err := config.NewDefaultConfig()
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to create default config: %w", err)
	}

	// get extra user values
	cfg := globalFlags.LoadedConfig(log)
	extraUserValues, err := config.GetExtraValuesNoDiff(&config.ExtraValuesOptions{
		DisableTelemetry:    cfg.TelemetryDisabled,
		InstanceCreatorType: "vclusterctl",
		MachineID:           telemetry.GetMachineID(cfg),
	})
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to get extra values: %w", err)
	}

	// enable docker
	extraUserValues.Experimental.Docker.Enabled = true

	// this is needed for dns to work correctly. Docker sets the dns to 127.0.0.11 but inside coredns or any other container
	// with dnsPolicy: Default it will not work. So we need to set our own /etc/k8s-resolv.conf for this on the kubelet which makes it work for containers as well as nodes.
	extraUserValues.PrivateNodes.Kubelet.Config = map[string]interface{}{}
	extraUserValues.PrivateNodes.Kubelet.Config["resolvConf"] = "/etc/k8s-resolv.conf"

	// disable konnectivity by default
	extraUserValues.ControlPlane.Advanced.Konnectivity.Server.Enabled = false
	extraUserValues.ControlPlane.Advanced.Konnectivity.Agent.Enabled = false

	// check if we should use the registry proxy
	extraArgs := []string{}
	if extraUserValues.Experimental.Docker.RegistryProxy.Enabled {
		if !isContainerdImageStore(ctx) {
			log.Infof("Docker is using the non-containerd image store, please use containerd image store to use the docker daemon registry proxy. For more information, see https://docs.docker.com/engine/storage/containerd/")
			extraUserValues.Experimental.Docker.RegistryProxy.Enabled = false
		} else {
			containerdSocketPath, err := getContainerdSocketPath(ctx)
			if err != nil {
				extraUserValues.Experimental.Docker.RegistryProxy.Enabled = false
				log.Infof("Containerd socket couldn't be found, disabling docker daemon registry proxy (%s)", err.Error())
			} else {
				extraArgs = append(extraArgs, "--mount", fmt.Sprintf("type=bind,src=%s,dst=/var/run/docker/containerd/containerd.sock,ro", containerdSocketPath))
			}
		}
	}
	extraValuesString, err := config.Diff(defaultConfig, extraUserValues)
	if err != nil {
		return nil, "", nil, fmt.Errorf("diff config: %w", err)
	}

	// merge all user values together
	userValues, err := mergeAllValues(options.SetValues, options.Values, extraValuesString)
	if err != nil {
		return nil, "", nil, fmt.Errorf("merge values: %w", err)
	}

	// parse config non-strict here to make sure we are compatible with other config formats
	userConfigRaw := map[string]interface{}{}
	if err := yaml.Unmarshal([]byte(userValues), &userConfigRaw); err != nil {
		return nil, "", nil, fmt.Errorf("unmarshal vcluster config: %w", err)
	}

	// merge with default config
	defaultConfigRaw, err := convertToMap(defaultConfig)
	if err != nil {
		return nil, "", nil, fmt.Errorf("convert to map: %w", err)
	}

	// merge the configs
	fullConfigRaw := strvals.MergeMaps(defaultConfigRaw, userConfigRaw)
	fullConfigBytes, err := json.Marshal(fullConfigRaw)
	if err != nil {
		return nil, "", nil, fmt.Errorf("marshal config: %w", err)
	}
	fullConfig := &config.Config{}
	err = json.Unmarshal(fullConfigBytes, fullConfig)
	if err != nil {
		return nil, "", nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return fullConfig, userValues, extraArgs, nil
}

func getHostDNSServer(ctx context.Context) (string, error) {
	// This shell command does two things:
	// 1. Tries to find the IP inside the 'ExtServers' comment (Docker Desktop/Embedded DNS style).
	// 2. If not found, falls back to the first 'nameserver' entry that is NOT 127.0.0.11.
	// 3. If all else fails, it might print nothing (which returns an error downstream).
	cmd := `
        ip=$(grep 'ExtServers' /etc/resolv.conf | grep -oE '[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+' | head -n 1);
        if [ -n "$ip" ]; then
            echo "$ip";
        else
            awk '/^nameserver/ && $2 != "127.0.0.11" {print $2; exit}' /etc/resolv.conf;
        fi
    `

	// We use "sh -c" to run the complex command string.
	out, err := exec.CommandContext(ctx, "docker", "run", "--rm", "alpine", "sh", "-c", cmd).Output()
	if err != nil {
		return "", fmt.Errorf("failed to get real dns name: %s: %w", string(out), err)
	}

	result := strings.TrimSpace(string(out))
	if result == "" {
		return "", fmt.Errorf("could not determine upstream DNS: /etc/resolv.conf contains only 127.0.0.11 and no ExtServers comment")
	}

	return result, nil
}

func isContainerdImageStore(ctx context.Context) bool {
	out, err := exec.CommandContext(ctx, "docker", "info", "-f", "{{ .DriverStatus }}").CombinedOutput()
	if err != nil {
		return false
	}

	return strings.Contains(string(out), "io.containerd.snapshotter")
}

func getContainerdSocketPath(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "docker", "run", "-q", "--rm", "--privileged", "--pid=host", "alpine", "nsenter", "-t", "1", "-m", "-p", "-u", "-i", "-n", "sh", "-c", `netstat -xlp | awk '$NF ~ /\/containerd\.sock$/ {print $NF}'`).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get containerd socket path: %s: %w", string(out), err)
	}

	scanner := bufio.NewScanner(strings.NewReader(strings.TrimSpace(string(out))))
	containerdSocketPaths := []string{}
	for scanner.Scan() {
		containerdSocketPaths = append(containerdSocketPaths, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to scan containerd socket path: %s: %w", string(out), err)
	}
	if len(containerdSocketPaths) == 0 {
		return "", fmt.Errorf("no containerd socket path found")
	}

	return containerdSocketPaths[0], nil
}

func convertToMap(config *config.Config) (map[string]interface{}, error) {
	raw, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	out := map[string]interface{}{}
	err = json.Unmarshal(raw, &out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func createNetwork(ctx context.Context, networkName string, log log.Logger) error {
	log.Infof("Creating network %s...", networkName)
	args := []string{"network", "create", networkName}
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
		if strings.Contains(string(out), "not found") {
			return nil
		}

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
