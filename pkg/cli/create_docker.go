package cli

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/log/hash"
	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/oci"
	vclusterconfig "github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/random"
	"github.com/loft-sh/vcluster/pkg/strvals"
	"github.com/loft-sh/vcluster/pkg/telemetry"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/loft-sh/vcluster/pkg/util/clihelper"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"golang.org/x/mod/semver"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

const joinTokenLabel = "vcluster.loft.sh/join-token"

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
	log.Infof("Ensuring environment for vCluster %s...", vClusterName)
	userValuesRaw, err := loadUserValues(options, globalFlags, log)
	if err != nil {
		return fmt.Errorf("failed to load docker config: %w", err)
	}

	// On Linux, load kernel modules required for node join (bridge, br_netfilter, overlay).
	// Only run modprobe for modules not already loaded (check via /proc/modules, no sudo).
	if runtime.GOOS == "linux" {
		required := []string{"overlay", "bridge", "br_netfilter"}
		loaded := make(map[string]bool)
		if data, err := os.ReadFile("/proc/modules"); err == nil {
			scanner := bufio.NewScanner(strings.NewReader(string(data)))
			for scanner.Scan() {
				fields := strings.Fields(scanner.Text())
				if len(fields) > 0 {
					loaded[fields[0]] = true
				}
			}
		}
		for _, mod := range required {
			if loaded[mod] {
				continue
			}
			if err := exec.CommandContext(ctx, "modprobe", mod).Run(); err != nil {
				log.Warnf("Could not load kernel module %s: %v. If node join fails, run: sudo modprobe overlay && sudo modprobe bridge && sudo modprobe br_netfilter", mod, err)
			}
		}
	}

	// configure the network and update user values if needed
	networkName, extraDockerArgs, err := configureNetwork(ctx, userValuesRaw, vClusterName, log)
	if err != nil {
		return fmt.Errorf("failed to configure network: %w", err)
	}

	// convert the config to a config object
	vConfig, userValues, err := convertConfig(userValuesRaw)
	if err != nil {
		return fmt.Errorf("convert config: %w", err)
	}

	// validate the config
	err = validateConfig(vConfig, vClusterName)
	if err != nil {
		return fmt.Errorf("validate config: %w", err)
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

	// write the vcluster.yaml
	extraVClusterArgs := []string{
		"--vcluster-name", vClusterName,
	}
	vClusterConfigDir, err := writeVClusterYAML(globalFlags, vClusterName, userValues)
	if err != nil {
		return err
	}

	// ensure the k8s resolv conf file
	err = ensureK8sResolvConf(ctx, globalFlags, vClusterName, log)
	if err != nil {
		return fmt.Errorf("failed to ensure k8s resolv conf file: %w", err)
	}

	// ensure the join token
	vClusterJoinToken, err := ensureVClusterJoinToken(globalFlags, vClusterName, true)
	if err != nil {
		return fmt.Errorf("failed to ensure join token: %w", err)
	}
	extraVClusterArgs = append(extraVClusterArgs, "--join-token", vClusterJoinToken)

	// add the platform credentials to the docker container
	if options.Add && !exists {
		err := vclusterconfig.ValidatePlatformProject(ctx, vConfig, globalFlags.LoadedConfig(log))
		if err != nil {
			return err
		}

		platformArgs, err := addVClusterDocker(ctx, vClusterName, vConfig, options, globalFlags, vClusterJoinToken, log)
		if err != nil {
			return err
		}

		if len(platformArgs) > 0 {
			log.Infof("Will connect vCluster %s to platform...", vClusterName)
			extraVClusterArgs = append(extraVClusterArgs, platformArgs...)
		}
	}

	// now remove the container if it exists
	if exists {
		err = removeContainer(ctx, getControlPlaneContainerName(vClusterName))
		if err != nil {
			return fmt.Errorf("failed to remove container: %w", err)
		}
	}

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

func addVClusterDocker(ctx context.Context, name string, vClusterConfig *config.Config, options *CreateOptions, globalFlags *flags.GlobalFlags, joinToken string, log log.Logger) ([]string, error) {
	platformConfig := vClusterConfig.GetPlatformConfig()
	if platformConfig.APIKey.SecretName != "" || platformConfig.APIKey.Namespace != "" {
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

	// add hashed token to the extra labels
	extraLabels := map[string]string{
		joinTokenLabel: hash.String(joinToken)[:32],
	}

	// try with the regular name first
	created, accessKey, createdName, err := platform.CreateWithName(ctx, managementClient, project, name, extraLabels)
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

// runDockerCommand runs a docker command and captures its combined output.
// If the command runs longer than streamDelay, all buffered output is flushed
// to the logger and subsequent lines are streamed in real time.
func runDockerCommand(ctx context.Context, args []string, streamDelay time.Duration, logger log.Logger) (string, error) {
	logger.Debugf("Running command: docker %s", strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, "docker", args...)

	pr, pw, err := os.Pipe()
	if err != nil {
		return "", fmt.Errorf("create output pipe: %w", err)
	}
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err := cmd.Start(); err != nil {
		pr.Close()
		pw.Close()
		return "", fmt.Errorf("start command: %w", err)
	}
	pw.Close()

	var (
		lines     []string
		streaming bool
		mu        sync.Mutex
	)

	if logger.GetLevel() >= logrus.DebugLevel {
		streaming = true
	}

	timer := time.AfterFunc(streamDelay, func() {
		mu.Lock()
		defer mu.Unlock()
		if streaming {
			return
		}
		streaming = true
		logger.Infof("Command is still running, showing output...")
		for _, line := range lines {
			logger.Infof("  %s", line)
		}
	})
	defer timer.Stop()

	scanner := bufio.NewScanner(pr)
	for scanner.Scan() {
		line := scanner.Text()
		mu.Lock()
		lines = append(lines, line)
		if streaming {
			logger.Infof("  %s", line)
		}
		mu.Unlock()
	}
	pr.Close()

	err = cmd.Wait()

	mu.Lock()
	allOutput := strings.Join(lines, "\n")
	mu.Unlock()

	if err != nil {
		return allOutput, err
	}

	return allOutput, nil
}

func installVClusterStandalone(ctx context.Context, vClusterName, vClusterVersion string, extraArgs []string, log log.Logger) error {
	log.Infof("Starting vCluster standalone %s", vClusterName)
	containerName := getControlPlaneContainerName(vClusterName)
	joinedArgs := strings.Join(extraArgs, " ")
	args := []string{
		"exec", containerName,
		"bash", "-c", fmt.Sprintf(`set -e -o pipefail; curl -sfLk "https://github.com/loft-sh/vcluster/releases/download/v%s/install-standalone.sh" | sh -s -- --skip-download --skip-wait %s`, vClusterVersion, joinedArgs),
	}

	out, err := runDockerCommand(ctx, args, 2*time.Minute, log)
	if err != nil {
		return fmt.Errorf("failed to start vCluster standalone: %w: %s", err, out)
	}

	return nil
}

func ensureK8sResolvConf(ctx context.Context, globalFlags *flags.GlobalFlags, vClusterName string, log log.Logger) error {
	// this is needed for dns to work correctly. Docker sets the dns to 127.0.0.11 but inside coredns or any other container
	// with dnsPolicy: Default it will not work. So we need to set our own /etc/k8s-resolv.conf for this on the kubelet which makes it work for containers as well as nodes.
	resolvConf := filepath.Join(filepath.Dir(globalFlags.Config), "docker", "vclusters", vClusterName, "k8s-resolv.conf")
	_, err := os.Stat(resolvConf)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("ensure k8s resolv conf file: %w", err)
		}

		hostDNSServer, err := getHostDNSServer(ctx, log)
		if err != nil {
			return fmt.Errorf("failed to get dns docker internal ip: %w", err)
		}

		// write the resolv.conf file
		resolvConfContent := fmt.Sprintf("# Custom Kubelet DNS resolver\nnameserver %s\noptions ndots:0\n", hostDNSServer)
		err = os.WriteFile(resolvConf, []byte(resolvConfContent), 0644)
		if err != nil {
			return fmt.Errorf("write k8s resolv conf file: %w", err)
		}

		// write the kubelet config file
		kubeletConfig := filepath.Join(filepath.Dir(resolvConf), "kubelet.env")
		err = os.WriteFile(kubeletConfig, []byte("KUBELET_EXTRA_ARGS=--resolv-conf=/etc/k8s-resolv.conf"), 0644)
		if err != nil {
			return fmt.Errorf("write kubelet config file: %w", err)
		}

		return nil
	}

	return nil
}

func ensureVClusterJoinToken(globalFlags *flags.GlobalFlags, vClusterName string, create bool) (string, error) {
	tokenPath := filepath.Join(filepath.Dir(globalFlags.Config), "docker", "vclusters", vClusterName, "token.txt")
	_, err := os.Stat(tokenPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("ensure token file: %w", err)
		}

		if !create {
			return "", err
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

func canMountPrivilegedPort(ctx context.Context, log log.Logger) bool {
	args := []string{"run", "-q", "--rm", "-p", "127.0.0.1:879:80", "alpine", "echo", "1"}
	log.Debugf("Running command: docker %s", strings.Join(args, " "))
	out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return false
	}

	log.Debugf("Output: %s", string(out))
	return true
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
		"--network-alias", vClusterName,
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
	args = append(args, "--mount", fmt.Sprintf("type=bind,src=%s/kubelet.env,dst=/etc/vcluster/vcluster-flags.env,ro", vClusterConfigDir))
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
		log.Warnf("Failed to run command docker %s", strings.Join(args, " "))
		return fmt.Errorf("failed to start docker container: %w: %s", err, string(out))
	}
	return nil
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
		"--network-alias", workerConfig.Name,
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
		if !strings.HasPrefix(entry.Name(), "kubernetes-") {
			continue
		}

		args = append(args, "--mount", fmt.Sprintf("type=bind,src=%s/%s,dst=/var/lib/vcluster/bin/%s,ro", kubernetesDir, entry.Name(), entry.Name()))
	}
	args = append(args, "--mount", fmt.Sprintf("type=bind,src=%s/k8s-resolv.conf,dst=/etc/k8s-resolv.conf,ro", vClusterConfigDir))
	args = append(args, "--mount", fmt.Sprintf("type=bind,src=%s/kubelet.env,dst=/etc/vcluster/vcluster-flags.env,ro", vClusterConfigDir))

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

func joinDockerContainer(ctx context.Context, vClusterName, containerName, vClusterJoinToken, kubernetesVersion string, log log.Logger) error {
	log.Infof("Joining node %s to vCluster %s...", containerName, vClusterName)

	joinScript := fmt.Sprintf(`
sleep 2
until curl -fsSLk -o /tmp/join.sh "https://%s:8443/node/join?token=%s&type=worker"; do
  echo "Waiting for vCluster API to be ready..."
  sleep 2
done
sh /tmp/join.sh --bundle-path /var/lib/vcluster/bin/kubernetes-%s-%s.tar.gz --force-join
`, vClusterName, url.QueryEscape(vClusterJoinToken), kubernetesVersion, runtime.GOARCH)
	args := []string{"exec", containerName, "bash", "-c", joinScript}

	out, err := runDockerCommand(ctx, args, 2*time.Minute, log)
	if err != nil {
		return fmt.Errorf("failed to join vCluster node: %w: %s", err, out)
	}

	return nil
}

func ensureVClusterNodes(ctx context.Context, kubernetesDir, vClusterConfigDir, vClusterName, networkName, vClusterJoinToken, kubernetesVersion string, vClusterConfig *config.Config, log log.Logger) error {
	nodes, err := findDockerContainer(ctx, "vcluster.node."+vClusterName+".")
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
			err = joinDockerContainer(ctx, vClusterName, getWorkerContainerName(vClusterName, node.Name), vClusterJoinToken, kubernetesVersion, log)
			if err != nil {
				return fmt.Errorf("failed to join vCluster node: %w", err)
			}
		}
	}

	return nil
}

func pullVClusterImage(ctx context.Context, vClusterVersion string, globalFlags *flags.GlobalFlags, log log.Logger) (string, error) {
	fullImage := "ghcr.io/loft-sh/vcluster-pro:" + vClusterVersion

	targetDir := filepath.Join(filepath.Dir(globalFlags.Config), "docker", "vcluster", vClusterVersion)
	targetDirBinary := filepath.Join(targetDir, "vcluster")
	_, err := os.Stat(targetDirBinary)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("stat target directory: %w", err)
	} else if err == nil {
		return targetDir, nil
	}

	// Use a staging directory so partial downloads don't leave a broken targetDir.
	// On success we atomically rename; on failure the staging dir is cleaned up
	// and the next run will retry.
	stagingDir := targetDir + ".downloading"
	_ = os.RemoveAll(stagingDir)
	err = os.MkdirAll(stagingDir, 0755)
	if err != nil {
		return "", fmt.Errorf("create staging directory: %w", err)
	}
	defer func() {
		// If stagingDir still exists at this point, the operation failed.
		_ = os.RemoveAll(stagingDir)
	}()

	tempDir, err := os.MkdirTemp("", "vcluster-upgrade-")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	log.Infof("Pulling image %s to %s...", fullImage, tempDir)
	err = oci.PullImage(ctx, fullImage, tempDir, nil)
	if err != nil {
		return "", fmt.Errorf("pull image: %w", err)
	}

	log.Infof("Extracting vcluster binary from %s to %s...", fullImage, stagingDir)
	err = oci.ExtractFile(tempDir, "/vcluster", filepath.Join(stagingDir, "vcluster"))
	if err != nil {
		return "", fmt.Errorf("extract image: %w", err)
	}

	// Ensure parent of targetDir exists, then atomically move staging into place.
	// Remove any stale targetDir (e.g. from an interrupted previous run) so
	// that os.Rename does not fail with "file exists".
	err = os.MkdirAll(filepath.Dir(targetDir), 0755)
	if err != nil {
		return "", fmt.Errorf("create target parent directory: %w", err)
	}
	_ = os.RemoveAll(targetDir)
	if err := os.Rename(stagingDir, targetDir); err != nil {
		return "", fmt.Errorf("finalize vcluster download: %w", err)
	}

	return targetDir, nil
}

func pullKubernetesImage(ctx context.Context, vConfig *config.Config, globalFlags *flags.GlobalFlags, log log.Logger) (string, string, error) {
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

	targetDir := filepath.Join(filepath.Dir(globalFlags.Config), "docker", "kubernetes", kubernetesVersion)
	_, err := os.Stat(targetDir)
	if err != nil && !os.IsNotExist(err) {
		return "", "", fmt.Errorf("stat target directory: %w", err)
	} else if err == nil {
		return targetDir, kubernetesVersion, nil
	}

	// Use a staging directory so partial downloads don't leave a broken targetDir.
	// On success we atomically rename; on failure the staging dir is cleaned up
	// and the next run will retry.
	stagingDir := targetDir + ".downloading"
	_ = os.RemoveAll(stagingDir)
	err = os.MkdirAll(stagingDir, 0755)
	if err != nil {
		return "", "", fmt.Errorf("create staging directory: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(stagingDir)
	}()

	tempDir, err := os.MkdirTemp("", "vcluster-docker-kubernetes-")
	if err != nil {
		return "", "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	log.Infof("Pulling image %s to %s...", fullImage, tempDir)
	err = oci.PullImage(ctx, fullImage, tempDir, nil)
	if err != nil {
		return "", "", fmt.Errorf("pull image: %w", err)
	}

	log.Infof("Extracting image %s to %s...", fullImage, stagingDir)
	err = oci.Extract(tempDir, "/kubernetes", stagingDir)
	if err != nil {
		return "", "", fmt.Errorf("extract image: %w", err)
	}

	// Ensure parent of targetDir exists, then atomically move staging into place.
	// Remove any stale targetDir so that os.Rename does not fail with "file exists".
	err = os.MkdirAll(filepath.Dir(targetDir), 0755)
	if err != nil {
		return "", "", fmt.Errorf("create target parent directory: %w", err)
	}
	_ = os.RemoveAll(targetDir)
	if err := os.Rename(stagingDir, targetDir); err != nil {
		return "", "", fmt.Errorf("finalize kubernetes download: %w", err)
	}

	return targetDir, kubernetesVersion, nil
}

func loadUserValues(options *CreateOptions, globalFlags *flags.GlobalFlags, log log.Logger) (map[string]interface{}, error) {
	defaultConfig, err := config.NewDefaultConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create default config: %w", err)
	}

	// get extra user values
	cfg := globalFlags.LoadedConfig(log)
	extraUserValues, err := config.GetExtraValuesNoDiff(&config.ExtraValuesOptions{
		DisableTelemetry:    cfg.TelemetryDisabled,
		InstanceCreatorType: "vclusterctl",
		MachineID:           telemetry.GetMachineID(cfg),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get extra values: %w", err)
	}

	// enable docker
	extraUserValues.Experimental.Docker.Enabled = true
	extraUserValues.ControlPlane.Standalone.Enabled = true
	extraUserValues.PrivateNodes.Enabled = true

	// disable konnectivity by default, user can still enable it via a values.yaml file
	extraUserValues.ControlPlane.Advanced.Konnectivity.Server.Enabled = false
	extraUserValues.ControlPlane.Advanced.Konnectivity.Agent.Enabled = false

	// calculate the diff between the default config and the extra user values
	extraValuesString, err := config.Diff(defaultConfig, extraUserValues)
	if err != nil {
		return nil, fmt.Errorf("diff config: %w", err)
	}

	// merge all user values together
	userValues, err := mergeAllValues(options.SetValues, options.Values, extraValuesString)
	if err != nil {
		return nil, fmt.Errorf("merge values: %w", err)
	}

	userValuesMap := map[string]interface{}{}
	err = yaml.Unmarshal([]byte(userValues), &userValuesMap)
	if err != nil {
		return nil, fmt.Errorf("unmarshal user values: %w", err)
	}

	// merge the configs
	return userValuesMap, nil
}

func configureNetwork(ctx context.Context, fullConfigRaw map[string]interface{}, vClusterName string, log log.Logger) (string, []string, error) {
	// convert the config to a config object
	fullConfig, _, err := convertConfig(fullConfigRaw)
	if err != nil {
		return "", nil, fmt.Errorf("convert config: %w", err)
	}

	// get the network name
	networkName := getNetworkName(vClusterName)
	if fullConfig.Experimental.Docker.Network != "" {
		networkName = fullConfig.Experimental.Docker.Network
	}

	// create the docker network
	err = createNetwork(ctx, networkName, log)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create network: %w", err)
	}

	// if the registry proxy is disabled, we don't need to mount the containerd socket
	extraArgs := []string{}
	if fullConfig.Experimental.Docker.RegistryProxy.Enabled {
		if !isContainerdImageStore(ctx) {
			log.Infof("Docker is using the non-containerd image store, please use containerd image store to use the docker daemon registry proxy. For more information, see https://docs.docker.com/engine/storage/containerd/")
			err = unstructured.SetNestedField(fullConfigRaw, false, "experimental", "docker", "registryProxy", "enabled")
			if err != nil {
				return "", nil, fmt.Errorf("failed to set nested field: %w", err)
			}
		} else {
			containerdSocketPath, err := getContainerdSocketPath(ctx)
			if err != nil {
				log.Infof("Containerd socket couldn't be found, disabling docker daemon registry proxy (%s)", err.Error())
				err = unstructured.SetNestedField(fullConfigRaw, false, "experimental", "docker", "registryProxy", "enabled")
				if err != nil {
					return "", nil, fmt.Errorf("failed to set nested field: %w", err)
				}
			} else {
				extraArgs = append(extraArgs, "--mount", fmt.Sprintf("type=bind,src=%s,dst=%s,ro", containerdSocketPath, constants.DockerContainerdSocketPath))
			}
		}
	}

	// if the load balancer is disabled, we don't need to mount the docker socket
	if fullConfig.Experimental.Docker.LoadBalancer.Enabled {
		loadBalancerArgs, err := configureLoadBalancer(ctx, fullConfigRaw, networkName, log)
		if err != nil {
			return "", nil, fmt.Errorf("failed to configure load balancer: %w", err)
		}
		extraArgs = append(extraArgs, loadBalancerArgs...)
	}

	return networkName, extraArgs, nil
}

func configureLoadBalancer(ctx context.Context, fullConfigRaw map[string]interface{}, networkName string, log log.Logger) ([]string, error) {
	extraArgs := []string{}
	reachable, err := isDockerNetworkReachable(ctx, networkName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if docker network is reachable: %w", err)
	}

	// if the docker network is reachable, we don't need to forward the ports
	if reachable {
		err = unstructured.SetNestedField(fullConfigRaw, false, "experimental", "docker", "loadBalancer", "forwardPorts")
		if err != nil {
			return nil, fmt.Errorf("failed to set nested field: %w", err)
		}
	} else {
		// this method only works on macos, where we can bind ip addresses to the loopback device
		if runtime.GOOS != "darwin" {
			err = unstructured.SetNestedField(fullConfigRaw, false, "experimental", "docker", "loadBalancer", "enabled")
			if err != nil {
				return nil, fmt.Errorf("failed to set nested field: %w", err)
			}

			log.Warnf("Load balancer type services are not supported inside the vCluster because the docker network is not reachable. Port-forwarding will not work. This is only supported on macOS")
			return extraArgs, nil
		}

		// check if privileged port helper is available
		canMountPrivilegedPort := canMountPrivilegedPort(ctx, log)
		if !canMountPrivilegedPort {
			err = unstructured.SetNestedField(fullConfigRaw, false, "experimental", "docker", "loadBalancer", "enabled")
			if err != nil {
				return nil, fmt.Errorf("failed to set nested field: %w", err)
			}

			log.Warnf("Load balancer type services are not supported inside the vCluster because privileged port mapping is not allowed. If you are using Docker Desktop, please enable it in the Docker Desktop settings")
			return extraArgs, nil
		}

		// check if we can configure the loopback device to forward the ports
		ips, err := findTailIPs(ctx, networkName, 10)
		if err != nil {
			return nil, fmt.Errorf("failed to find tail ips: %w", err)
		}
		for _, ip := range ips {
			out, err := exec.CommandContext(ctx, "ifconfig", "lo0", "alias", ip).CombinedOutput()
			if err != nil {
				if strings.Contains(string(out), "permission denied") {
					err = unstructured.SetNestedField(fullConfigRaw, false, "experimental", "docker", "loadBalancer", "enabled")
					if err != nil {
						return nil, fmt.Errorf("failed to set nested field: %w", err)
					}

					log.Warnf("Load balancer type services are not supported inside the vCluster because this command was executed with insufficient privileges. To enable load balancer type services, run this command with sudo")
					return extraArgs, nil
				}

				return nil, fmt.Errorf("failed to add loopback alias: %s: %w", string(out), err)
			}
		}
	}

	// mount the docker socket
	dockerSocketPath, err := getDockerSocketPath(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get docker socket path: %w", err)
	}
	extraArgs = append(extraArgs, "--mount", fmt.Sprintf("type=bind,src=%s,dst=%s,ro", dockerSocketPath, constants.DockerSocketPath))
	return extraArgs, nil
}

func convertConfig(userConfigRaw map[string]interface{}) (*config.Config, string, error) {
	userConfigBytes, err := yaml.Marshal(userConfigRaw)
	if err != nil {
		return nil, "", fmt.Errorf("marshal config: %w", err)
	}

	// we need to merge the user config with the default config
	defaultConfig, err := config.NewDefaultConfig()
	if err != nil {
		return nil, "", fmt.Errorf("failed to create default config: %w", err)
	}
	defaultConfigRaw, err := convertToMap(defaultConfig)
	if err != nil {
		return nil, "", fmt.Errorf("convert to map: %w", err)
	}

	fullConfigRaw := strvals.MergeMaps(defaultConfigRaw, userConfigRaw)
	fullConfigBytes, err := yaml.Marshal(fullConfigRaw)
	if err != nil {
		return nil, "", fmt.Errorf("marshal config: %w", err)
	}

	fullConfig := &config.Config{}
	err = yaml.Unmarshal(fullConfigBytes, fullConfig)
	if err != nil {
		return nil, "", fmt.Errorf("unmarshal config: %w", err)
	}

	return fullConfig, string(userConfigBytes), nil
}

func validateConfig(fullConfig *config.Config, vClusterName string) error {
	// validate the config
	nodeNames := make(map[string]bool)
	for _, node := range fullConfig.Experimental.Docker.Nodes {
		if node.Name == "" {
			return fmt.Errorf("node name is required")
		}
		if node.Name == vClusterName {
			return fmt.Errorf("node name %s is not allowed to be the same as the vCluster name", node.Name)
		}
		if nodeNames[node.Name] {
			return fmt.Errorf("duplicate node name %s", node.Name)
		}
		nodeNames[node.Name] = true
	}

	return nil
}

func getHostDNSServer(ctx context.Context, log log.Logger) (string, error) {
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
	out, err := exec.CommandContext(ctx, "docker", "run", "-q", "--rm", "alpine", "sh", "-c", cmd).Output()
	if err != nil {
		return "", fmt.Errorf("failed to get real dns name: %s: %w", string(out), err)
	}

	result := strings.TrimSpace(string(out))
	if result == "" {
		return "", fmt.Errorf("could not determine upstream DNS: /etc/resolv.conf contains only 127.0.0.11 and no ExtServers comment")
	}

	log.Debugf("Host DNS server: %s", result)
	return result, nil
}

func isDockerNetworkReachable(ctx context.Context, networkName string) (bool, error) {
	// 1. Start a container listening on port 8080.
	// We use 'nc -l -p 8080' instead of 'tail -f' so we have a target to connect to.
	out, err := exec.CommandContext(ctx, "docker", "run", "-q", "-d", "--rm", "--network", networkName, "alpine", "nc", "-l", "-p", "8080").CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("failed to start test network container: %s: %w", string(out), err)
	}

	containerID := strings.TrimSpace(string(out))

	// 2. Ensure cleanup: Kill the container when function exits.
	// We use a background context here because if 'ctx' is cancelled,
	// we still want the cleanup command to run.
	defer func() {
		_ = exec.Command("docker", "kill", containerID).Run()
	}()

	// 3. Inspect the container to get its IP address.
	// This format string grabs the IP from the first network found.
	inspectCmd := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}", containerID)
	ipOut, err := inspectCmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("failed to inspect test network container: %s: %w", string(ipOut), err)
	}

	ip := strings.TrimSpace(string(ipOut))
	if ip == "" {
		return false, fmt.Errorf("container started but has no IP address")
	}

	// 4. Try to reach the IP directly via TCP.
	// We use a small retry loop because Docker networking or the 'nc' process
	// might take a few milliseconds to be fully ready.
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	// Create a child context with a hard timeout for the connection attempt
	dialCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	for {
		select {
		case <-dialCtx.Done():
			// Timed out or cancelled
			return false, nil
		case <-ticker.C:
			d := net.Dialer{}
			conn, err := d.DialContext(dialCtx, "tcp", net.JoinHostPort(ip, "8080"))
			if err == nil {
				conn.Close()
				return true, nil
			}
		}
	}
}

func isContainerdImageStore(ctx context.Context) bool {
	out, err := exec.CommandContext(ctx, "docker", "info", "-f", "{{ .DriverStatus }}").CombinedOutput()
	if err != nil {
		return false
	}

	return strings.Contains(string(out), "io.containerd.snapshotter")
}

// NetworkResource represents the partial JSON structure returned by "docker network inspect"
// We only define the fields we strictly need.
type NetworkResource struct {
	IPAM struct {
		Config []struct {
			Subnet string `json:"Subnet"`
		} `json:"Config"`
	} `json:"IPAM"`
	// Containers is a map of ContainerID -> ContainerDetails
	Containers map[string]struct {
		IPv4Address string `json:"IPv4Address"`
	} `json:"Containers"`
}

// findTailIPs finds the last tailSize IPs in the network
func findTailIPs(ctx context.Context, networkName string, tailSize int) ([]string, error) {
	// 1. Execute "docker network inspect <networkName>"
	cmd := exec.CommandContext(ctx, "docker", "network", "inspect", networkName)
	output, err := cmd.Output()
	if err != nil {
		// Try to capture stderr for a better error message if possible
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("docker inspect failed: %w, stderr: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to execute docker inspect: %w", err)
	}

	// 2. Unmarshal the JSON output
	// Docker inspect returns a JSON array of networks, even if we request just one.
	var resources []NetworkResource
	if err := json.Unmarshal(output, &resources); err != nil {
		return nil, fmt.Errorf("failed to parse docker inspect json: %w", err)
	}

	if len(resources) == 0 {
		return nil, fmt.Errorf("network %s not found", networkName)
	}
	netResource := resources[0]

	if len(netResource.IPAM.Config) == 0 {
		return nil, fmt.Errorf("no IPAM config found for network %s", networkName)
	}

	// Assume IPv4 and take the first config
	subnetCIDR := netResource.IPAM.Config[0].Subnet
	_, ipNet, err := net.ParseCIDR(subnetCIDR)
	if err != nil {
		return nil, fmt.Errorf("invalid subnet CIDR: %w", err)
	}

	// 4. Calculate the range of IPs
	// Ensure we are working with a 4-byte IPv4 representation
	ip4 := ipNet.IP.To4()
	if ip4 == nil {
		return nil, fmt.Errorf("subnet is not a valid IPv4 address")
	}

	startIP := binary.BigEndian.Uint32(ip4)
	mask := binary.BigEndian.Uint32(ipNet.Mask)

	// Calculate Broadcast address: (Network IP) | (^Mask)
	broadcast := startIP | ^mask
	ips := []string{}

	// We start checking from (Broadcast - 1) downwards
	for i := 1; i <= tailSize; i++ {
		candidate := broadcast - uint32(i)

		// Safety check: ensure we haven't wrapped around to the network address or beyond
		if candidate <= startIP {
			break
		}

		// Convert back to net.IP
		ipBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(ipBytes, candidate)

		// add the ip to the list
		ips = append(ips, net.IP(ipBytes).String())
	}

	return ips, nil
}

func getDockerSocketPath(ctx context.Context) (string, error) {
	// Updated awk regex: /\/docker\.sock(\.real)?$/
	// This matches paths ending in "/docker.sock" OR "/docker.sock.real"
	cmdStr := `
        if command -v netstat >/dev/null 2>&1; then
            netstat -xlp | awk '$NF ~ /\/docker\.sock(\.real)?$/ {print $NF}'
        else
            for p in /var/run/docker.sock.real /var/run/docker.sock; do
                if [ -S "$p" ]; then
                    echo "$p"
                fi
            done
        fi
    `

	out, err := exec.CommandContext(ctx, "docker", "run", "-q", "--rm", "--privileged", "--pid=host", "alpine", "nsenter", "-t", "1", "-m", "-p", "-u", "-i", "-n", "sh", "-c", cmdStr).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("failed to get docker socket path: %s: %w", string(exitErr.Stderr), err)
		}
		return "", fmt.Errorf("failed to get docker socket path: %w", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(strings.TrimSpace(string(out))))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Only accept the line if it looks like an absolute path.
		if strings.HasPrefix(line, "/") {
			return line, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to scan docker socket path: %w", err)
	}

	return "", fmt.Errorf("no docker socket path found")
}

func getContainerdSocketPath(ctx context.Context) (string, error) {
	cmdStr := `
        if command -v netstat >/dev/null 2>&1; then
            netstat -xlp | awk '$NF ~ /\/containerd\.sock$/ {print $NF}'
        else
            for p in /run/containerd/containerd.sock /var/run/containerd/containerd.sock /var/run/docker/containerd/containerd.sock; do
                if [ -S "$p" ]; then
                    echo "$p"
                fi
            done
        fi
    `

	out, err := exec.CommandContext(ctx, "docker", "run", "-q", "--rm", "--privileged", "--pid=host", "alpine", "nsenter", "-t", "1", "-m", "-p", "-u", "-i", "-n", "sh", "-c", cmdStr).Output()
	if err != nil {
		// Extract stderr for better debugging if the command actually fails
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("failed to get containerd socket path: %s: %w", string(exitErr.Stderr), err)
		}
		return "", fmt.Errorf("failed to get containerd socket path: %w", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(strings.TrimSpace(string(out))))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Only accept the line if it looks like an absolute path.
		// This ignores any stray warning lines that might still slip through.
		if strings.HasPrefix(line, "/") {
			return line, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to scan containerd socket path: %w", err)
	}

	return "", fmt.Errorf("no containerd socket path found")
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
	// 1. Check if the network already exists
	// It is cleaner to check this first rather than relying on the "already exists" error text later.
	if err := exec.CommandContext(ctx, "docker", "network", "inspect", networkName).Run(); err == nil {
		log.Debugf("Network %s already exists, skipping creation", networkName)
		return nil
	}

	// 2. Create a temporary "probe" network
	// We use a unique name to ensure we don't conflict with other operations.
	probeName := fmt.Sprintf("%s-probe-%d", networkName, time.Now().UnixNano())
	log.Debugf("Creating probe network %s to discover valid subnet", probeName)

	if out, err := exec.CommandContext(ctx, "docker", "network", "create", probeName).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create probe network: %w: %s", err, string(out))
	}

	// SAFETY: Ensure the probe is deleted when we are done, even if we crash/return error below.
	defer func() {
		// We suppress errors here because if the happy path works, the probe is already gone.
		_ = exec.CommandContext(ctx, "docker", "network", "rm", probeName).Run()
	}()

	// 3. Inspect the probe to retrieve the Subnet
	// We use a specific Go template to extract only the subnet string.
	cmd := exec.CommandContext(ctx, "docker", "network", "inspect", probeName, "--format", "{{(index .IPAM.Config 0).Subnet}}")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to inspect probe subnet: %w: %s", err, string(out))
	}

	subnet := strings.TrimSpace(string(out))
	if subnet == "" {
		return fmt.Errorf("probe network returned empty subnet")
	}
	log.Debugf("Discovered free subnet: %s", subnet)

	// 4. Delete the probe explicitly
	// We must delete it NOW to free up the subnet so we can reuse it immediately.
	if err := exec.CommandContext(ctx, "docker", "network", "rm", probeName).Run(); err != nil {
		return fmt.Errorf("failed to remove probe network: %w", err)
	}

	// 5. Create the ACTUAL network with the specific Subnet
	// This satisfies the "user configured subnet" requirement.
	args := []string{"network", "create", "--subnet", subnet, networkName}
	log.Debugf("Running command: docker %s", strings.Join(args, " "))

	out, err = exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		// Handle race condition: If someone created it between step 1 and now
		if strings.Contains(string(out), "already exists") {
			log.Donef("Network %s already exists", networkName)
			return nil
		}
		return fmt.Errorf("failed to create network %s with subnet %s: %w: %s", networkName, subnet, err, string(out))
	}

	log.Donef("Created network %s", networkName)
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
	return constants.DockerNetworkPrefix + vClusterName
}

func getControlPlaneContainerName(vClusterName string) string {
	return constants.DockerControlPlanePrefix + vClusterName
}

func getControlPlaneVolumeName(vClusterName, volumeName string) string {
	return constants.DockerControlPlanePrefix + vClusterName + "." + volumeName
}

func getWorkerContainerName(vClusterName, workerName string) string {
	return constants.DockerNodePrefix + vClusterName + "." + workerName
}

func getWorkerVolumeName(vClusterName, workerName, volumeName string) string {
	return constants.DockerNodePrefix + vClusterName + "." + workerName + "." + volumeName
}
