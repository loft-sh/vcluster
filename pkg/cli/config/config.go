package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/platform"
	homedir "github.com/mitchellh/go-homedir"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DirName         = ".vcluster"
	FileName        = "config.json"
	ManagerFileName = "manager.json"
)

var (
	singleConfig     *Config
	singleConfigOnce sync.Once
)

// New creates a new default config
func New() *Config {
	return &Config{
		TelemetryDisabled: false,
		Platform: &PlatformConfig{
			platform.Config{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Config",
					APIVersion: "storage.loft.sh/v1",
				},
				VirtualClusterAccessPointCertificates: make(map[string]platform.VirtualClusterCertificatesEntry),
			},
		},
	}
}

// Read tries to read the vcluster config file from the default location once.
// It returns a default config if there are any errors.
func Read(log log.Logger) *Config {
	singleConfigOnce.Do(func() {
		home, err := homedir.Dir()
		if err != nil {
			log.Debugf("Failed to open vcluster configuration file from, unable to detect $HOME directory, falling back to default configuration, following error occurred: %v", err)
		}
		// set default if nil
		if singleConfig == nil {
			singleConfig = readOrNewCompat(home, log)
		}
	})

	return singleConfig
}

func Write(c *Config) error {
	home, err := homedir.Dir()
	if err != nil {
		return fmt.Errorf("failed to write vcluster configuration file from, unable to detect $HOME directory, falling back to default configuration, following error occurred: %w", err)
	}
	path := configFilePath(home)

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

// FilePath returns the path to the vcluster config file
func FilePath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", fmt.Errorf("failed to open vCluster configuration file, unable to detect $HOME directory, falling back to default configuration, following error occurred: %w", err)
	}

	return filepath.Join(home, DirName, FileName), nil
}

func GetManager(manager string) (ManagerType, error) {
	if manager != "" {
		if manager != string(ManagerPlatform) && manager != string(ManagerHelm) {
			return "", fmt.Errorf("unknown manager: %s, please choose either helm or platform", manager)
		}

		return ManagerType(manager), nil
	}

	managerFile, err := LoadManagerFile()
	if err != nil {
		return "", err
	} else if managerFile.Manager == "" {
		return ManagerHelm, nil
	}

	return managerFile.Manager, nil
}

func LoadManagerFile() (*ManagerConfig, error) {
	managerFile, err := managerFilePath()
	if err != nil {
		return nil, fmt.Errorf("get manager file path: %w", err)
	}

	_, err = os.Stat(managerFile)
	if err != nil {
		// couldn't find manager file, so just return an empty manager config
		return &ManagerConfig{}, nil
	}

	rawBytes, err := os.ReadFile(managerFile)
	if err != nil {
		return nil, fmt.Errorf("error reading vCluster platform manager file: %w", err)
	}

	managerConfig := &ManagerConfig{}
	err = json.Unmarshal(rawBytes, managerConfig)
	if err != nil {
		return nil, fmt.Errorf("error parsing vCluster platform manager file: %w", err)
	}

	return managerConfig, nil
}

func SaveManagerFile(managerConfig *ManagerConfig) error {
	managerFile, err := managerFilePath()
	if err != nil {
		return fmt.Errorf("get manager file path: %w", err)
	}

	rawBytes, err := json.Marshal(managerConfig)
	if err != nil {
		return fmt.Errorf("marshal manager config: %w", err)
	}

	err = os.MkdirAll(filepath.Dir(managerFile), 0755)
	if err != nil {
		return fmt.Errorf("create manager dir: %w", err)
	}

	err = os.WriteFile(managerFile, rawBytes, 0644)
	if err != nil {
		return fmt.Errorf("error saving manager config: %w", err)
	}

	return nil
}

func PrintManagerInfo(verb string, manager ManagerType, log log.Logger) {
	// only print this to stderr
	log = log.ErrorStreamOnly()

	if manager == ManagerHelm {
		log.Infof("Using vCluster manager 'helm' to %s your virtual clusters, which means the vCluster CLI is running helm commands directly", verb)
		log.Info("If you prefer to use the vCluster platform API instead, use the flag '--manager platform' or run 'vcluster use manager platform' to change the default")
	} else {
		log.Infof("Using vCluster manager 'platform' to %s your virtual clusters, which means the CLI is using the vCluster platform API instead of helm", verb)
		log.Info("If you prefer to use helm instead, use the flag '--manager helm' or run 'vcluster use manager helm' to change the default")
	}
}

func managerFilePath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", fmt.Errorf("failed to open vCluster manager file, unable to detect $HOME directory, falling back to default configuration, following error occurred: %w", err)
	}

	return filepath.Join(home, DirName, ManagerFileName), nil
}

func configFilePath(home string) string {
	return filepath.Join(home, DirName, FileName)
}

// TODO delete after ".vcluster/pro/config.json" is not supported anymore.
func configFilePathDeprecated(home string) string {
	return filepath.Join(home, DirName, "pro", FileName)
}

// TODO end

func readOrNewCompat(home string, log log.Logger) *Config {
	// TODO delete after ".vcluster/pro/config.json" is not supported anymore.
	// Try to read the old config file in the pro sub directory.
	oldPath := configFilePathDeprecated(home)
	oldPlatformConfig := tryReadPlatformConfig(oldPath)
	if oldPlatformConfig != nil {
		log.Infof("There is a config file located in %s, which is deprecated. If you did not configure this deliberately, feel free to remove this directory.", oldPath)
	}
	// TODO end

	cfg, err := readOrNew(configFilePath(home))
	if err != nil {
		// At this point we didn't read an existing file but created a new default one in memory.
		// So let's add our potential config file from the deprecated location.
		log.Debugf("Failed to load local configuration file: %v", err)
		cfg.Platform = oldPlatformConfig
	}

	// TODO delete after ".vcluster/pro/config.json" is not supported anymore.
	// Only add the old config if there is nothing set in the new config.
	if cfg.Platform == nil {
		cfg.Platform = oldPlatformConfig
	}
	// TODO end

	return cfg
}

func readOrNew(path string) (*Config, error) {
	// check if the file exists
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return New(), nil
		}
		return New(), fmt.Errorf("failed to load vcluster configuration file from %s, falling back to default configuration, following error occurred: %w", path, err)
	}

	if fi.IsDir() {
		return New(), fmt.Errorf("failed to load vcluster configuration file %s, falling back to default configuration, this path is a directory", path)
	}

	file, err := os.Open(path)
	if err != nil {
		return New(), fmt.Errorf("failed to open vcluster configuration file from %s, falling back to default configuration, following error occurred: %w", path, err)
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return New(), err
	}

	c := &Config{}
	err = json.Unmarshal(bytes, &c)
	if err != nil {
		return New(), fmt.Errorf("failed to unmarshall vcluster configuration from %s file, falling back to default configuration, following error occurred: %w", path, err)
	}

	return c, nil
}

// TODO delete after ".vcluster/pro/config.json" is not supported anymore.
func tryReadPlatformConfig(path string) *PlatformConfig {
	// Try to read the old config file in the pro sub directory.
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil
	}

	cfg := &PlatformConfig{}
	err = json.Unmarshal(bytes, cfg)
	if err != nil {
		return nil
	}

	return cfg
}

// TODO end
