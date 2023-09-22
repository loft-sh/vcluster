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
	VclusterProFolder     = "pro"
	LoftctlConfigFileName = "creds.json"
)

var (
	// ErrNoLastVersion is returned if no last version was found in the config
	ErrNoLastVersion = errors.New("no vcluster pro version found, please run 'vcluster pro login' first")
)

// CLIConfig is the config of the CLI
type CLIConfig struct {
	LatestVersion string    `json:"latestVersion,omitempty"`
	LatestCheckAt time.Time `json:"latestCheck,omitempty"`
}

// defaultCLIConfig returns the default config
func defaultCLIConfig() *CLIConfig {
	return &CLIConfig{}
}

// LoftctlConfigFilePath returns the path to the loft config file
func LoftctlConfigFilePath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", fmt.Errorf("failed to open vcluster pro configuration file from, unable to detect $HOME directory, falling back to default configuration, following error occurred: %w", err)
	}

	return filepath.Join(home, cliconfig.VclusterFolder, VclusterProFolder, LoftctlConfigFileName), nil
}

// configFilePath returns the path to the config file
func configFilePath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", fmt.Errorf("failed to open vcluster pro configuration file from, unable to detect $HOME directory, falling back to default configuration, following error occurred: %w", err)
	}

	return filepath.Join(home, cliconfig.VclusterFolder, VclusterProFolder, cliconfig.ConfigFileName), nil
}

// GetConfig returns the config from the config file
func GetConfig() (*CLIConfig, error) {
	path, err := configFilePath()
	if err != nil {
		return defaultCLIConfig(), fmt.Errorf("failed to get vcluster pro configuration file path: %w", err)
	}

	// check if the file exists
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultCLIConfig(), nil
		}
		return defaultCLIConfig(), fmt.Errorf("failed to load vcluster configuration file from %s, falling back to default configuration, following error occurred: %w", path, err)
	}
	if fi.IsDir() {
		return defaultCLIConfig(), fmt.Errorf("failed to load vcluster configuration file %s, falling back to default configuration, this path is a directory", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return defaultCLIConfig(), fmt.Errorf("failed to open vcluster configuration file from %s, falling back to default configuration, following error occurred: %w", path, err)
	}
	defer file.Close()
	bytes, _ := io.ReadAll(file)
	c := &CLIConfig{}
	err = json.Unmarshal(bytes, &c)
	if err != nil {
		return defaultCLIConfig(), fmt.Errorf("failed to unmarshall vcluster configuration from %s file, falling back to default configuration, following error occurred: %w", path, err)
	}
	return c, nil
}

// WriteConfig writes the given config to the config file
func WriteConfig(c *CLIConfig) error {
	path, err := configFilePath()
	if err != nil {
		return fmt.Errorf("failed to get vcluster configuration file path: %w", err)
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
