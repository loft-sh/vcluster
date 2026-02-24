package standalone

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"text/template"
)

type AddToPlatformOptions struct {
	AccessKey    string
	Host         string
	Insecure     bool
	InstanceName string
	ProjectName  string
}

func AddToPlatform(ctx context.Context, options *AddToPlatformOptions) error {
	if err := createPlatformConf(ctx, options); err != nil {
		return err
	}

	if err := restartService(ctx); err != nil {
		return err
	}

	return nil
}

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

func createPlatformConf(ctx context.Context, options *AddToPlatformOptions) error {
	platformConfFileBytes, err := renderSystemdPlatformConfFile(options)
	if err != nil {
		return fmt.Errorf("failed to render systemd vcluster platform conf file: %w", err)
	}

	if os.MkdirAll("/etc/systemd/system/vcluster.service.d", 0700) != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile("/etc/systemd/system/vcluster.service.d/platform.conf", platformConfFileBytes, 0600); err != nil {
		return fmt.Errorf("failed to write systemd service file: %w", err)
	}

	return nil
}
