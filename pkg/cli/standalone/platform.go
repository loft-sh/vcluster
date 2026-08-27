package standalone

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"text/template"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/constants"
)

// AddToPlatformOptions holds the configuration for connecting a standalone vcluster to the vCluster Platform.
type AddToPlatformOptions struct {
	AccessKey      string
	Host           string
	Insecure       bool
	InstanceName   string
	ProjectName    string
	SkipConfigSync bool
}

// AddToPlatform configures the standalone vcluster to connect to the vCluster Platform by creating a systemd
// configuration file and restarting the vcluster service.
func AddToPlatform(ctx context.Context, log log.Logger, options *AddToPlatformOptions) error {
	if err := preflightChecks(); err != nil {
		return err
	}

	log.Info("Creating systemd vcluster service platform conf drop-in file")
	if err := createPlatformConf(log, options); err != nil {
		return err
	}

	log.Info("Restarting vcluster.service")
	if err := restartService(ctx); err != nil {
		return err
	}

	return nil
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

// createPlatformConf writes the access key to a root-only env file and the remaining
// vCluster Platform configuration to a systemd drop-in file.
func createPlatformConf(log log.Logger, options *AddToPlatformOptions) error {
	// check if vcluster service exists
	if _, err := os.Stat(constants.VClusterStandaloneSystemdUnitFile); err != nil {
		return fmt.Errorf("vcluster service not found: %w", err)
	}

	warnOnAccessKeyInUnit(log)

	if err := writePlatformEnvFile(constants.VClusterStandalonePlatformEnvFile, options); err != nil {
		return err
	}

	// create systemd platform conf file
	platformConfFileBytes, err := renderSystemdPlatformConfFile(constants.VClusterStandalonePlatformEnvFile, options)
	if err != nil {
		return fmt.Errorf("failed to render systemd vcluster platform conf file: %w", err)
	}

	if err := os.MkdirAll(constants.VClusterStandaloneSystemdDropInDir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(constants.VClusterStandalonePlatformDropInFile, platformConfFileBytes, 0600); err != nil {
		return fmt.Errorf("failed to write systemd service file: %w", err)
	}

	return nil
}

// writePlatformEnvFile writes the access key to a root-only file. It must not be a
// systemd Environment= directive: systemd serves those to any local user over D-Bus.
func writePlatformEnvFile(envPath string, options *AddToPlatformOptions) error {
	// only root reads this directory; Chmod because MkdirAll leaves an existing one alone
	secretsDir := filepath.Dir(envPath)
	if err := os.MkdirAll(secretsDir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	if err := os.Chmod(secretsDir, 0700); err != nil {
		return fmt.Errorf("failed to chmod %s: %w", secretsDir, err)
	}

	// remove first so the mode below applies even if the file already exists
	if err := os.Remove(envPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove %s: %w", envPath, err)
	}

	if err := os.WriteFile(envPath, renderPlatformEnvFile(options), 0600); err != nil {
		return fmt.Errorf("failed to write %s: %w", envPath, err)
	}

	return nil
}

// renderPlatformEnvFile renders the systemd environment file holding the access key.
func renderPlatformEnvFile(options *AddToPlatformOptions) []byte {
	return fmt.Appendf(nil, "%s=%s\n", constants.PlatformAccessKeyEnv, options.AccessKey)
}

// warnOnAccessKeyInUnit reports a key left inline by an installer predating the env file.
// The drop-in overrides it, but it stays readable until the unit is rewritten.
func warnOnAccessKeyInUnit(log log.Logger) {
	unit, err := os.ReadFile(constants.VClusterStandaloneSystemdUnitFile)
	if err != nil || !bytes.Contains(unit, []byte(constants.PlatformAccessKeyEnv)) {
		return
	}

	log.Warnf("%s is still set inline in %s, where any local user can read it. Re-run install-standalone.sh, or remove that line and run 'systemctl daemon-reload'. Rotate the access key that was exposed.", constants.PlatformAccessKeyEnv, constants.VClusterStandaloneSystemdUnitFile)
}

// renderSystemdPlatformConfFile renders the systemd environment variables for the vCluster Platform connection.
func renderSystemdPlatformConfFile(envPath string, options *AddToPlatformOptions) ([]byte, error) {
	const platformConfTemplateText = `
[Service]
EnvironmentFile=-{{.envPath}}
Environment=LOFT_PLATFORM_HOST="{{.options.Host}}"
Environment=LOFT_PLATFORM_INSECURE="{{.options.Insecure}}"
Environment=LOFT_PLATFORM_INSTANCE_NAME="{{.options.InstanceName}}"
Environment=LOFT_PLATFORM_PROJECT_NAME="{{.options.ProjectName}}"
Environment=LOFT_PLATFORM_SKIP_CONFIG_SYNC="{{.options.SkipConfigSync}}"
`

	serviceTemplate, err := template.New("platformConf").Parse(platformConfTemplateText)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	if err := serviceTemplate.Execute(buf, map[string]any{"options": options, "envPath": envPath}); err != nil {
		return nil, fmt.Errorf("failed to render systemd service file: %w", err)
	}

	return buf.Bytes(), nil
}

// restartService reloads the systemd daemon and restarts the vcluster service.
func restartService(ctx context.Context) error {
	if err := exec.CommandContext(ctx, "systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to systemctl daemon-reload: %w", err)
	}

	if err := exec.CommandContext(ctx, "systemctl", "restart", "vcluster.service").Run(); err != nil {
		return fmt.Errorf("failed to start vcluster: %w", err)
	}

	return nil
}
