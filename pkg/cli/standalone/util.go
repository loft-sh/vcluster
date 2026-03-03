package standalone

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/loft-sh/vcluster/config"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

// findConfig finds the first config file that exists
func findConfig(configs ...string) (string, error) {
	for _, c := range configs {
		if c == "" {
			continue
		}

		path, err := filepath.Abs(c)
		if err != nil {
			return "", fmt.Errorf("failed to get absolute path for c: %w", err)
		}

		if _, err := os.Stat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return "", fmt.Errorf("failed to stat config file: %w", err)
		}

		return path, nil
	}

	return "", nil
}

func lookupDataDir(configPath string) (string, error) {
	dataDir := "/var/lib/vcluster"

	if configPath != "" {
		configBytes, err := os.ReadFile(configPath)
		if err != nil {
			return "", fmt.Errorf("failed to read config file: %w", err)
		}
		partialConfig, err := configPartialUnmarshal(configBytes)
		if err != nil {
			return "", fmt.Errorf("failed to unmarshal config: %w", err)
		}

		if partialConfig.ControlPlane.Standalone.DataDir != "" {
			dataDir = partialConfig.ControlPlane.Standalone.DataDir
		}
	}

	return dataDir, nil
}

func configPartialUnmarshal(configBytes []byte) (*config.Config, error) {
	var partialConfig struct {
		ControlPlane struct {
			Standalone config.Standalone `json:"standalone,omitempty"`
		} `json:"controlPlane,omitempty"`
	}

	if err := yaml.Unmarshal(configBytes, &partialConfig); err != nil {
		return nil, err
	}

	return &config.Config{
		ControlPlane: config.ControlPlane{
			Standalone: partialConfig.ControlPlane.Standalone,
		},
	}, nil
}

func downloadFile(ctx context.Context, c *http.Client, url string, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	res, err := c.Do(req)
	if err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		_ = f.Close()
		_ = os.Remove(path)
		return fmt.Errorf("unexpected status code: %s", res.Status)
	}

	_, err = io.Copy(f, res.Body)
	if err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return fmt.Errorf("failed to write bundle to file: %w", err)
	}

	return nil
}

func restartService(ctx context.Context) error {
	log := klog.FromContext(ctx)

	log.Info("Restarting vcluster.service")
	if err := exec.CommandContext(ctx, "systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to systemctl daemon-reload: %w", err)
	}

	if err := exec.CommandContext(ctx, "systemctl", "restart", "vcluster").Run(); err != nil {
		return fmt.Errorf("failed to start vcluster: %w", err)
	}

	return nil
}

func logLatestServiceLogs(ctx context.Context, lines int) error {
	log := klog.FromContext(ctx)

	log.Info("Getting latest service logs", "lines", lines)
	cmd := exec.CommandContext(ctx, "journalctl", "-u", "vcluster.service", "--no-pager", "-n", strconv.Itoa(lines), "-e")
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute output: %w", err)
	}

	return nil
}

func checkServiceIsRunning(ctx context.Context) error {
	log := klog.FromContext(ctx)
	cmd := exec.CommandContext(ctx, "systemctl", "show", "vcluster.service", "--property=MainPID", "--value")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute systemctl show vcluster.service: %w", err)
	}

	parts := strings.Split(string(out), "\n")
	if len(parts) != 2 {
		return fmt.Errorf("unexpected output format: %s", string(out))
	}

	if parts[0] == "0" {
		log.Info("vcluster.service is not running")
		if err := logLatestServiceLogs(ctx, 100); err != nil {
			return fmt.Errorf("failed to get logs: %w", err)
		}
		return fmt.Errorf("main process has exited")
	}

	return nil
}
