package cliconfig

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
)

const (
	VclusterFolder = ".vcluster"
	ConfigFileName = "config.json"
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
	return filepath.Join(home, VclusterFolder, ConfigFileName)
}

func GetConfig() (*CLIConfig, error) {
	home, err := homedir.Dir()
	if err != nil {
		return getDefaultCLIConfig(), fmt.Errorf("failed to open vcluster configuration file from, unable to detect $HOME directory, falling back to default configuration, following error occurred: %v", err)
	}

	path := getConfigFilePath(home)
	// check if the file exists
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return getDefaultCLIConfig(), nil
		}
		return getDefaultCLIConfig(), fmt.Errorf("failed to load vcluster configuration file from %s, falling back to default configuration, following error occurred: %v", path, err)
	}
	if fi.IsDir() {
		return getDefaultCLIConfig(), fmt.Errorf("failed to load vcluster configuration file %s, falling back to default configuration, this path is a directory", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return getDefaultCLIConfig(), fmt.Errorf("failed to open vcluster configuration file from %s, falling back to default configuration, following error occurred: %v", path, err)
	}
	defer file.Close()
	bytes, _ := io.ReadAll(file)
	c := &CLIConfig{}
	err = json.Unmarshal(bytes, &c)
	if err != nil {
		return getDefaultCLIConfig(), fmt.Errorf("failed to unmarshall vcluster configuration from %s file, falling back to default configuration, following error occurred: %v", path, err)
	}
	return c, nil
}

func WriteConfig(c *CLIConfig) error {
	home, err := homedir.Dir()
	if err != nil {
		return fmt.Errorf("failed to write vcluster configuration file from, unable to detect $HOME directory, falling back to default configuration, following error occurred: %v", err)
	}
	path := getConfigFilePath(home)

	err = os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory for configuration file, following error occurred: %v", err)
	}

	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to transform config into JSON format, following error occurred: %v", err)
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write configuration file, following error occurred: %v", err)
	}

	return nil
}
