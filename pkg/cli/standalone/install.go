package standalone

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"text/template"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/archive"
	corev1 "k8s.io/api/core/v1"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/cmd/kubeadm/app/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// InstallOptions defines the configuration for installing a vCluster Standalone node.
type InstallOptions struct {
	Name               string
	Version            string
	SkipWait           bool
	SkipDownload       bool
	Env                map[string]string
	Fips               bool
	Config             string
	KubernetesBundle   string
	Binary             string
	DownloadURL        string
	JoinURL            string
	InsecureSkipVerify bool
	HTTPClient         *http.Client
}

// Install installs a vCluster Standalone node.
func Install(ctx context.Context, options *InstallOptions) error {
	if err := preflightChecks(); err != nil {
		return err
	}

	if err := checkMinTmpDiskSpace(); err != nil {
		return err
	}

	workspace, err := os.MkdirTemp("", "vcluster-cli-standalone-install-")
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}
	defer cleanupWorkspace(ctx, workspace)

	ic, err := newInstallContext(ctx, options, workspace)
	if err != nil {
		return err
	}

	if err := stopAndDisableService(ctx); err != nil {
		return err
	}

	if err := installBinaries(ctx, ic.binaries, ic.dataDir); err != nil {
		return err
	}

	if err := installUsrLocalBinVClusterLink(ctx, ic.dataDir); err != nil {
		return err
	}

	if err := installPKIs(ctx, ic.pkis, ic.dataDir); err != nil {
		return err
	}

	if err := installEtcdPeersTxt(ctx, ic.joinEtcdEndpoint, ic.dataDir); err != nil {
		return err
	}

	if err := installConfig(ctx, ic.config, ic.confDir); err != nil {
		return err
	}

	if err := setupPersistentLogging(ctx); err != nil {
		return err
	}

	if err := createService(ic); err != nil {
		return err
	}

	if err := startService(ctx); err != nil {
		return err
	}

	if !ic.skipWait {
		if err := waitForServiceToBeReady(ctx, ic.dataDir); err != nil {
			return err
		}
	}

	return logInstallCompleteMessage(ctx)
}

// preflightChecks ensures the system meets the requirements for a vCluster Standalone installation.
func preflightChecks() error {
	// validate supported OS and ARCH
	if runtime.GOOS != "linux" {
		return fmt.Errorf("only Linux OS is supported")
	}

	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		return fmt.Errorf("only amd64 and arm64 architectures are supported")
	}

	// Check if systemctl is installed
	_, err := exec.LookPath("systemctl")
	if err != nil {
		return fmt.Errorf("systemctl is not installed. This installer only works on systems that use systemd: %w", err)
	}

	// Ensure we're running as root
	if os.Getuid() != 0 {
		return fmt.Errorf("this installer needs the ability to run commands as root")
	}

	return nil
}

// checkMinTmpDiskSpace checks if there is enough free space on the temporary directory.
func checkMinTmpDiskSpace() error {
	tmpRequiredBytes := uint64(1024 * 1024 * 1024)
	tmpDir := os.TempDir()
	var st syscall.Statfs_t
	if err := syscall.Statfs(tmpDir, &st); err != nil {
		return fmt.Errorf("failed to get filesystem stats: %w", err)
	}

	tmpAvailableBytes := st.Bavail * uint64(st.Bsize)
	if tmpAvailableBytes < tmpRequiredBytes {
		return fmt.Errorf("not enough free space on %s, availableBytes:%d, requiredBytes:%d", tmpDir, tmpAvailableBytes, tmpRequiredBytes)
	}

	return nil
}

// cleanupWorkspace removes the temporary workspace directory created during installation.
func cleanupWorkspace(ctx context.Context, workspace string) {
	if err := os.RemoveAll(workspace); err != nil {
		klog.FromContext(ctx).Error(err, "Failed to remove temporary workspace")
	}
}

// installContext holds the installation process configuration.
type installContext struct {
	name             string
	version          string
	skipWait         bool
	skipDownload     bool
	env              map[string]string
	fips             bool
	config           string
	kubernetesBundle string
	downloadURL      string

	joinURL          string
	joinToken        string
	joinEndpoint     string
	joinEtcdEndpoint string
	isJoin           bool

	confDir string
	dataDir string

	httpClient         *http.Client
	insecureSkipVerify bool

	workspace string
	binaries  map[string]string
	pkis      map[string]string
}

// newInstallContext creates a new installContext based on the provided InstallOptions.
func newInstallContext(ctx context.Context, options *InstallOptions, workspace string) (*installContext, error) {
	log := klog.FromContext(ctx)

	ic := &installContext{
		name:             options.Name,
		version:          options.Version,
		skipWait:         options.SkipWait,
		skipDownload:     options.SkipDownload,
		env:              options.Env,
		fips:             options.Fips,
		config:           options.Config,
		kubernetesBundle: options.KubernetesBundle,
		downloadURL:      options.DownloadURL,
		joinURL:          options.JoinURL,

		confDir: "/etc/vcluster",

		httpClient:         options.HTTPClient,
		insecureSkipVerify: options.InsecureSkipVerify,
		workspace:          workspace,

		binaries: map[string]string{},
		pkis:     map[string]string{},
	}

	// http client init
	if ic.httpClient == nil {
		ic.httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: ic.insecureSkipVerify,
				},
			},
		}
	}

	// join config init
	err := initJoinConfig(ic)
	if err != nil {
		return nil, err
	}

	// provided binaries init
	if err := initProvidedBinaries(options.Binary, ic); err != nil {
		return nil, err
	}

	if ic.isJoin {
		// download control plane bundle
		if err := downloadControlPlaneBundle(ctx, ic); err != nil {
			return nil, err
		}
	} else {
		if !ic.skipDownload {
			// download vcluster binary
			if err := downloadBinary(ctx, ic); err != nil {
				return nil, err
			}
		}
	}

	// set config absolute path
	ic.config, err = findConfig(ic.config, filepath.Join(ic.confDir, "vcluster.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to find config file: %w", err)
	}

	// dataDir lookup
	ic.dataDir, err = lookupDataDir(ic.config)
	if err != nil {
		return nil, fmt.Errorf("failed to get data directory: %w", err)
	}
	log.Info("Using data directory", "path", ic.dataDir)

	return ic, nil
}

// initJoinConfig sets the join configuration for the installation context.
func initJoinConfig(ic *installContext) error {
	if ic.joinURL == "" {
		ic.isJoin = false
		return nil
	}

	joinURL, err := url.Parse(ic.joinURL)
	if err != nil {
		return fmt.Errorf("failed to parse join URL: %w", err)
	}

	ic.joinEndpoint = fmt.Sprintf("%s:%s", joinURL.Hostname(), joinURL.Port())
	if ic.joinEndpoint == "" {
		return fmt.Errorf("endpoint is missing in join URL")
	}

	if ic.joinEtcdEndpoint == "" {
		ic.joinEtcdEndpoint = joinURL.Hostname()
	}

	ic.joinToken = joinURL.Query().Get("token")
	if ic.joinToken == "" {
		return fmt.Errorf("token is missing in join URL")
	}

	ic.isJoin = true

	return nil
}

// initProvidedBinaries initializes the provided vCluster binaries in the installation context.
func initProvidedBinaries(providedBinary string, ic *installContext) error {
	if providedBinary == "" {
		return nil
	}

	absProvidedBinary, err := filepath.Abs(providedBinary)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for provided binary: %w", err)
	}

	if _, err := os.Stat(absProvidedBinary); err != nil {
		return fmt.Errorf("failed to find provided binary path %s: %w", providedBinary, err)
	}

	for _, binary := range []string{"vcluster", "vcluster-cli"} {
		path := filepath.Join(providedBinary, binary)
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("failed to find provided binary %s: %w", path, err)
		}

		ic.binaries[binary] = path
	}

	ic.skipDownload = true

	return nil
}

// downloadControlPlaneBundle downloads the control plane bundle from the join endpoint.
func downloadControlPlaneBundle(ctx context.Context, ic *installContext) error {
	log := klog.FromContext(ctx)

	// wait for control plane
	checkURL := fmt.Sprintf("https://%s/node/join?token=%s", ic.joinEndpoint, ic.joinToken)
	log.Info("Waiting for vCluster control plane to be ready", "url", checkURL)
	err := wait.PollUntilContextTimeout(ctx, time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, checkURL, nil)
		if err != nil {
			return false, fmt.Errorf("failed to create request: %w", err)
		}
		res, err := ic.httpClient.Do(req)
		if err != nil {
			log.Info("Failed to check control-plane readiness", "err", err)
			return false, nil
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			log.Info("Failed to check control-plane readiness: unexpected status code", "statusCode", res.StatusCode)
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return err
	}

	// download control plane bundle
	controlPlaneBundleURL := fmt.Sprintf("https://%s/node/control-plane-download/%s", ic.joinEndpoint, ic.joinToken)
	log.Info("Downloading vCluster control plane bundle", "url", controlPlaneBundleURL)
	if err := downloadFile(ctx, ic.httpClient, controlPlaneBundleURL, filepath.Join(ic.workspace, "control-plane.tar.gz")); err != nil {
		return fmt.Errorf("failed to download control plane bundle: %w", err)
	}

	if err := os.MkdirAll(filepath.Join(ic.workspace, "control-plane-bundle"), 0755); err != nil {
		return fmt.Errorf("failed to create control plane bundle directory: %w", err)
	}

	// extract control plane bundle
	if err := archive.ExtractTarGz(filepath.Join(ic.workspace, "control-plane.tar.gz"), filepath.Join(ic.workspace, "control-plane-bundle")); err != nil {
		return fmt.Errorf("extract bundle: %w", err)
	}

	// override config option with the control plane bundle config
	ic.config = filepath.Join(ic.workspace, "control-plane-bundle", "vcluster.yaml")
	if _, err := os.Stat(ic.config); err != nil {
		return fmt.Errorf("failed to find control plane bundle config: %w", err)
	}

	// register binaries to install
	binaries := map[string]string{
		"vcluster":     filepath.Join(ic.workspace, "control-plane-bundle", "vcluster"),
		"vcluster-cli": filepath.Join(ic.workspace, "control-plane-bundle", "release", "vcluster-cli"),
	}

	for binary, path := range binaries {
		ic.binaries[binary] = path
		if _, err := os.Stat(ic.binaries[binary]); err != nil {
			return fmt.Errorf("failed to find control plane bundle %s: %w", binary, err)
		}
	}

	// register PKIs to install
	pkisSrcPath := filepath.Join(ic.workspace, "control-plane-bundle", "pki")
	ignoredPkis := map[string]bool{
		"etcd/peer.crt":   true,
		"etcd/peer.key":   true,
		"etcd/server.crt": true,
		"etcd/server.key": true,
	}

	err = filepath.WalkDir(pkisSrcPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(pkisSrcPath, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		if _, ok := ignoredPkis[relPath]; ok {
			return nil
		}

		ic.pkis[relPath] = path

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to copy PKI files: %w", err)
	}

	return nil
}

// downloadBinary downloads the vcluster binary for the given version and platform.
func downloadBinary(ctx context.Context, ic *installContext) error {
	log := klog.FromContext(ctx)

	baseDownloadURL := ic.downloadURL
	if baseDownloadURL == "" {
		baseDownloadURL = fmt.Sprintf("https://github.com/loft-sh/vcluster/releases/download/v%s", ic.version)
	}

	var variant string
	if ic.fips {
		variant = "-fips"
	}

	assets := map[string]string{
		"vcluster":     fmt.Sprintf("vcluster-%s-%s-standalone%s", runtime.GOOS, runtime.GOARCH, variant),
		"vcluster-cli": fmt.Sprintf("vcluster-%s-%s", runtime.GOOS, runtime.GOARCH),
	}

	for binary, asset := range assets {
		downloadURL := fmt.Sprintf("%s/%s", baseDownloadURL, asset)
		path := filepath.Join(ic.workspace, binary)
		log.Info("Downloading vCluster binary", "url", downloadURL, "path", path)
		if err := downloadFile(ctx, ic.httpClient, downloadURL, path); err != nil {
			return fmt.Errorf("failed to download %s version assets %s: %w", ic.version, downloadURL, err)
		}

		// register binary to install
		ic.binaries[binary] = path
	}

	return nil
}

// stopAndDisableService attempts to stop and disable 'vcluster' systemd service.
func stopAndDisableService(ctx context.Context) error {
	log := klog.FromContext(ctx)

	log.Info("Checking if vcluster service is active")
	if err := exec.CommandContext(ctx, "systemctl", "is-active", "--quiet", "vcluster.service").Run(); err != nil {
		log.Info("vcluster service is not active", "err", err)
		return nil
	}

	log.Info("Stopping vcluster service")
	if err := exec.CommandContext(ctx, "systemctl", "stop", "vcluster.service").Run(); err != nil {
		log.Info("Failed to stop vcluster service", "err", err)
	}

	log.Info("Disabling vcluster service")
	if err := exec.CommandContext(ctx, "systemctl", "disable", "vcluster.service").Run(); err != nil {
		log.Info("Failed to disable vcluster service", "err", err)
	}

	return nil
}

// installPKIs copies the PKI files from the control-plane-bundle to the data directory.
func installPKIs(ctx context.Context, pkis map[string]string, dataDir string) error {
	log := klog.FromContext(ctx)

	for relPath, srcPath := range pkis {
		dstPath := filepath.Join(dataDir, "pki", relPath)
		log.Info("Installing PKI file", "src", srcPath, "dst", dstPath)

		dirName := filepath.Dir(dstPath)
		if err := os.MkdirAll(dirName, 0700); err != nil {
			return fmt.Errorf("creating directory %s: %w", dirName, err)
		}

		if err := util.CopyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("failed to copy PKI file: %w", err)
		}
	}

	return nil
}

// installBinaries copies the vCluster binary in the binary directory and sets executable permissions.
func installBinaries(ctx context.Context, binaries map[string]string, dataDir string) error {
	log := klog.FromContext(ctx)
	binDir := filepath.Join(dataDir, "bin")

	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", binDir, err)
	}

	for relPah, srcPath := range binaries {
		dstPath := filepath.Join(binDir, relPah)
		log.Info("Installing vcluster binary", "src", srcPath, "dst", dstPath)

		newDstPath := dstPath + ".new"
		if err := util.CopyFile(srcPath, newDstPath); err != nil {
			return fmt.Errorf("failed to copy %s to %s: %w", srcPath, dstPath, err)
		}

		if err := os.Rename(newDstPath, dstPath); err != nil {
			return fmt.Errorf("failed to rename %s to %s: %w", newDstPath, dstPath, err)
		}

		if err := os.Chmod(dstPath, 0755); err != nil {
			return fmt.Errorf("failed to set permissions for vclucster binary: %w", err)
		}
	}

	return nil
}

// installUsrLocalBinVClusterLink creates a symlink for the vcluster cli binary in /usr/local/bin.
func installUsrLocalBinVClusterLink(ctx context.Context, dataDir string) error {
	log := klog.FromContext(ctx)

	srcPath := filepath.Join(dataDir, "bin", "vcluster-cli")
	dstPath := "/usr/local/bin/vcluster"

	if _, err := os.Lstat(dstPath); err == nil {
		log.Info("vCluster cli already exists at path", "path", dstPath)
		return nil
	}

	if err := os.Symlink(srcPath, dstPath); err != nil {
		return fmt.Errorf("failed to create symlink for vcluster cli: %w", err)
	}

	log.Info("Created symlink for vcluster cli", "src", srcPath, "dst", dstPath)

	return nil
}

// installEtcdPeersTxt creates a peers.txt file in the data directory with the join endpoint.
func installEtcdPeersTxt(ctx context.Context, joinEtcdEndpoint string, dataDir string) error {
	if joinEtcdEndpoint == "" {
		return nil
	}

	log := klog.FromContext(ctx)

	peersTxtPath := filepath.Join(dataDir, "peers.txt")

	_, err := os.Stat(peersTxtPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to stat peers.txt: %w", err)
		}
	} else {
		log.Info("peers.txt already exists", "path", peersTxtPath)
		return nil
	}

	log.Info("Adding etcd endpoint to peers.txt file", "endpoint", joinEtcdEndpoint)
	peersTxt := fmt.Sprintf("%s=https://%s:2380\n", joinEtcdEndpoint, joinEtcdEndpoint)
	if err := os.WriteFile(peersTxtPath, []byte(peersTxt), 0644); err != nil {
		return fmt.Errorf("failed to write peers.txt: %w", err)
	}

	return nil
}

// installConfig copies the user-provided vCluster configuration file to the system configuration directory.
func installConfig(ctx context.Context, config string, confDir string) error {
	if config == "" {
		return nil
	}

	log := klog.FromContext(ctx)

	dstPath := filepath.Join(confDir, "vcluster.yaml")
	if config == dstPath {
		return nil
	}

	dstPathExists := true
	if _, err := os.Stat(dstPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			dstPathExists = false
		} else {
			return fmt.Errorf("failed to stat config file: %w", err)
		}
	}

	if dstPathExists {
		srcRealPath, err := filepath.EvalSymlinks(config)
		if err != nil {
			return fmt.Errorf("failed to resolve config symlink: %w", err)
		}

		dstRealPath, err := filepath.EvalSymlinks(dstPath)
		if err != nil {
			return fmt.Errorf("failed to resolve config symlink: %w", err)
		}

		if srcRealPath == dstRealPath {
			log.Info("Config file already exists at path", "path", dstPath)
			return nil
		}
	}

	log.Info("Copying config file", "src", config, "dst", dstPath)
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", filepath.Dir(dstPath), err)
	}

	if err := util.CopyFile(config, dstPath); err != nil {
		return fmt.Errorf("failed to copy config file: %w", err)
	}

	return nil
}

// setupPersistentLogging ensures systemd journald is configured for persistent logging by creating /var/log/journal.
func setupPersistentLogging(ctx context.Context) error {
	log := klog.FromContext(ctx)

	log.Info("Setting up persistent logging")

	if err := os.MkdirAll("/var/log/journal", 0755); err != nil {
		return fmt.Errorf("failed to create journal directory: %w", err)
	}

	if err := exec.CommandContext(ctx, "systemctl", "restart", "systemd-journald").Run(); err != nil {
		return fmt.Errorf("failed to restart systemd-journald: %w", err)
	}

	log.Info("Restarted systemd-journald")

	return nil
}

// createService writes the rendered systemd service file to /etc/systemd/system/vcluster.service.
func createService(ic *installContext) error {
	serviceFileBytes, err := renderSystemdServiceFile(ic)
	if err != nil {
		return fmt.Errorf("failed to render systemd service file: %w", err)
	}

	if err := os.WriteFile("/etc/systemd/system/vcluster.service", serviceFileBytes, 0644); err != nil {
		return fmt.Errorf("failed to write systemd service file: %w", err)
	}

	return nil
}

// renderSystemdServiceFile generates the systemd service unit file for vCluster from a template.
func renderSystemdServiceFile(ic *installContext) ([]byte, error) {
	const serviceTemplateText = `
[Unit]
Description=vcluster
Documentation=https://vcluster.com
Wants=network-online.target
After=network-online.target dbus.service

[Install]
WantedBy=multi-user.target

[Service]
Type=notify
{{- range $key, $value := .Envs }}
Environment={{$key}}="{{$value}}"
{{- end }}
EnvironmentFile=-/etc/default/%N
EnvironmentFile=-/etc/sysconfig/%N
KillMode=process
Delegate=yes
User=root
# Having non-zero Limit*s causes performance problems due to accounting overhead
# in the kernel. We recommend using cgroups to do container-local accounting.
LimitNOFILE=1048576
LimitNPROC=infinity
LimitCORE=infinity
TasksMax=infinity
TimeoutStartSec=0
Restart=always
RestartSec=5s
ExecStart={{.DataDir}}/bin/vcluster start --config {{.ConfDir}}/vcluster.yaml`

	serviceTemplate, err := template.New("service").Parse(serviceTemplateText)
	if err != nil {
		return nil, err
	}

	// Add extra envs
	envs := map[string]string{}
	for k, v := range ic.env {
		envs[k] = v
	}

	envs["VCLUSTER_NAME"] = ic.name
	envs["VCLUSTER_KUBERNETES_BUNDLE"] = ic.kubernetesBundle

	data := map[string]interface{}{
		"Envs":    envs,
		"ConfDir": ic.confDir,
		"DataDir": ic.dataDir,
	}

	buf := new(bytes.Buffer)
	if err := serviceTemplate.Execute(buf, data); err != nil {
		return nil, fmt.Errorf("failed to render systemd service file: %w", err)
	}

	return buf.Bytes(), nil
}

// startService reloads the systemd daemon and starts the vCluster service.
func startService(ctx context.Context) error {
	log := klog.FromContext(ctx)

	log.Info("Starting vcluster.service")
	if err := exec.CommandContext(ctx, "systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to systemctl daemon-reload: %w", err)
	}

	if err := exec.CommandContext(ctx, "systemctl", "enable", "--now", "vcluster.service").Run(); err != nil {
		return fmt.Errorf("failed to start vcluster: %w", err)
	}

	if err := checkServiceIsRunning(ctx); err != nil {
		return err
	}

	log.Info("Successfully started vcluster.service")

	return nil
}

// waitForServiceToBeReady waits for the vCluster service to become fully functional.
// It checks for the existence of the kubeconfig file and then polls the API server until it responds.
func waitForServiceToBeReady(ctx context.Context, dataDir string) error {
	log := klog.FromContext(ctx)

	if err := checkServiceIsRunning(ctx); err != nil {
		return err
	}

	kubeConfigPath := filepath.Join(dataDir, "kubeconfig.yaml")

	// wait for the kubeconfig to be available
	log.Info("Initializing Kubernetes control plane")
	err := wait.PollUntilContextTimeout(ctx, time.Second*5, 10*time.Minute, true, func(ctx context.Context) (bool, error) {
		if _, err := os.Stat(kubeConfigPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return false, checkServiceIsRunning(ctx)
			}

			return false, err
		}

		return true, nil
	})
	if err != nil {
		return err
	}

	// wait for the api server to be ready
	kubeConfigBytes, err := os.ReadFile(kubeConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read kubeconfig: %w", err)
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeConfigBytes)
	if err != nil {
		return fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	c, err := client.New(restConfig, client.Options{})
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// wait for the api server to be ready
	err = wait.PollUntilContextTimeout(ctx, time.Second*5, 10*time.Minute, true, func(ctx context.Context) (bool, error) {
		var nodes corev1.NodeList
		if err := c.List(ctx, &nodes); err != nil {
			if utilnet.IsConnectionRefused(err) {
				return false, checkServiceIsRunning(ctx)
			}

			return false, err
		}

		return true, nil
	})
	if err != nil {
		return err
	}

	log.Info("API Server is ready")

	return nil
}

// logInstallCompleteMessage prints a message indicating a successful installation and how to access the vCluster.
func logInstallCompleteMessage(ctx context.Context) error {
	log := klog.FromContext(ctx)
	if _, err := exec.LookPath("systemctl"); err != nil {
		log.Info("vCluster is ready. Use '/usr/local/bin/kubectl get pods' to access the vCluster")
	} else {
		log.Info("vCluster is ready. Use 'kubectl get pods' to access the vCluster")
	}

	log.Info("To check vCluster logs, use 'journalctl -u vcluster.service --no-pager'")

	return nil
}
