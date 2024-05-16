package cliconfig

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/constants"
	homedir "github.com/mitchellh/go-homedir"
)

var (
	singleConfig     *CLIConfig
	singleConfigOnce sync.Once
)

type CLIConfig struct {
	TelemetryDisabled bool `json:"telemetryDisabled,omitempty"`
}

func getDefaultCLIConfig() *CLIConfig {
	return &CLIConfig{
		TelemetryDisabled: false,
	}
}

func getConfigFilePath(home string) string {
	return filepath.Join(home, constants.VClusterFolder, constants.ConfigFileName)
}

// ConfigFilePath returns the path to the vcluster config file
func ConfigFilePath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", fmt.Errorf("failed to open vCluster configuration file, unable to detect $HOME directory, falling back to default configuration, following error occurred: %w", err)
	}

	return getConfigFilePath(home), nil
}

func GetConfig(log log.Logger) *CLIConfig {
	singleConfigOnce.Do(func() {
		var err error
		singleConfig, err = getConfig()
		if err != nil && log != nil {
			log.Debugf("Failed to load local configuration file: %v", err.Error())
		}

		// set default if nil
		if singleConfig == nil {
			singleConfig = getDefaultCLIConfig()
		}
	})

	return singleConfig
}

func getConfig() (*CLIConfig, error) {
	path, err := ConfigFilePath()
	if err != nil {
		return nil, err
	}
	// check if the file exists
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return getDefaultCLIConfig(), nil
		}
		return getDefaultCLIConfig(), fmt.Errorf("failed to load vcluster configuration file from %s, falling back to default configuration, following error occurred: %w", path, err)
	}
	if fi.IsDir() {
		return getDefaultCLIConfig(), fmt.Errorf("failed to load vcluster configuration file %s, falling back to default configuration, this path is a directory", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return getDefaultCLIConfig(), fmt.Errorf("failed to open vcluster configuration file from %s, falling back to default configuration, following error occurred: %w", path, err)
	}
	defer file.Close()
	bytes, _ := io.ReadAll(file)
	c := &CLIConfig{}
	err = json.Unmarshal(bytes, &c)
	if err != nil {
		return getDefaultCLIConfig(), fmt.Errorf("failed to unmarshall vcluster configuration from %s file, falling back to default configuration, following error occurred: %w", path, err)
	}
	return c, nil
}

func WriteConfig(c *CLIConfig) error {
	path, err := ConfigFilePath()
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory for configuration file, following error occurred: %w", err)
	}

	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to transform config into JSON format, following error occurred: %w", err)
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write configuration file, following error occurred: %w", err)
	}

	return nil
}
