package pro

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/cliconfig"
	homedir "github.com/mitchellh/go-homedir"
)

const (
	VclusterProFolder = "pro"
	BinariesFolder    = "bin"
)

var (
	// ErrNoLastVersion is returned if no last version was found in the config
	ErrNoLastVersion = errors.New("no vcluster pro version found, please run 'vcluster pro login' first")
)

// CLIConfig is the config of the CLI
type CLIConfig struct {
	LatestVersion   string    `json:"latestVersion,omitempty"`
	LatestCheckAt   time.Time `json:"latestCheck,omitempty"`
	LastUsedVersion string    `json:"lastUsedVersion,omitempty"`
}

// getDefaultCLIConfig returns the default config
func getDefaultCLIConfig() *CLIConfig {
	return &CLIConfig{}
}

// getConfigFilePath returns the path to the config file
func getConfigFilePath(home string) string {
	return filepath.Join(home, cliconfig.VclusterFolder, VclusterProFolder, cliconfig.ConfigFileName)
}

// GetConfig returns the config from the config file
func GetConfig() (*CLIConfig, error) {
	home, err := homedir.Dir()
	if err != nil {
		return getDefaultCLIConfig(), fmt.Errorf("failed to open vcluster pro configuration file from, unable to detect $HOME directory, falling back to default configuration, following error occurred: %w", err)
	}

	path := getConfigFilePath(home)
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

// WriteConfig writes the given config to the config file
func WriteConfig(c *CLIConfig) error {
	home, err := homedir.Dir()
	if err != nil {
		return fmt.Errorf("failed to write vcluster configuration file from, unable to detect $HOME directory, falling back to default configuration, following error occurred: %w", err)
	}
	path := getConfigFilePath(home)

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

// LastUsedVersion returns the last used version of the loft cli
func LastUsedVersion() (string, error) {
	config, err := GetConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get vcluster pro config: %w", err)
	}

	if config.LastUsedVersion == "" {
		return "", ErrNoLastVersion
	}

	return config.LastUsedVersion, nil
}
