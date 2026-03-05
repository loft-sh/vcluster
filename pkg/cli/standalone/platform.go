package standalone

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"text/template"

	"github.com/loft-sh/log"
)

// AddToPlatformOptions holds the configuration for connecting a standalone vcluster to the vCluster Platform.
type AddToPlatformOptions struct {
	AccessKey    string
	Host         string
	Insecure     bool
	InstanceName string
	ProjectName  string
}

// AddToPlatform configures the standalone vcluster to connect to the vCluster Platform by creating a systemd
// configuration file and restarting the vcluster service.
func AddToPlatform(ctx context.Context, log log.Logger, options *AddToPlatformOptions) error {
	if err := preflightChecks(); err != nil {
		return err
	}

	log.Info("Creating systemd vcluster service platform conf drop-in file")
	if err := createPlatformConf(options); err != nil {
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

// createPlatformConf writes the vCluster Platform configuration to a systemd drop-in file.
func createPlatformConf(options *AddToPlatformOptions) error {
	// check if vcluster service exists
	if _, err := os.Stat("/etc/systemd/system/vcluster.service"); err != nil {
		return fmt.Errorf("vcluster service not found: %w", err)
	}

	// create systemd platform conf file
	platformConfFileBytes, err := renderSystemdPlatformConfFile(options)
	if err != nil {
		return fmt.Errorf("failed to render systemd vcluster platform conf file: %w", err)
	}

	if err := os.MkdirAll("/etc/systemd/system/vcluster.service.d", 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile("/etc/systemd/system/vcluster.service.d/platform.conf", platformConfFileBytes, 0600); err != nil {
		return fmt.Errorf("failed to write systemd service file: %w", err)
	}

	return nil
}

// renderSystemdPlatformConfFile renders the systemd environment variables for the vCluster Platform connection.
func renderSystemdPlatformConfFile(options *AddToPlatformOptions) ([]byte, error) {
	const platformConfTemplateText = `
[Service]
Environment=LOFT_PLATFORM_ACCESS_KEY="{{.options.AccessKey}}"
Environment=LOFT_PLATFORM_HOST="{{.options.Host}}"
Environment=LOFT_PLATFORM_INSECURE="{{.options.Insecure}}"
Environment=LOFT_PLATFORM_INSTANCE_NAME="{{.options.InstanceName}}"
Environment=LOFT_PLATFORM_PROJECT_NAME="{{.options.ProjectName}}"
`

	serviceTemplate, err := template.New("platformConf").Parse(platformConfTemplateText)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	if err := serviceTemplate.Execute(buf, map[string]any{"options": options}); err != nil {
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
