package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/manager"
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
	singleConfig      *Config
	singleConfigOnce  sync.Once
	singleConfigMutex sync.RWMutex
)

// New creates a new default config
func New() *Config {
	return &Config{
		TelemetryDisabled: false,
		Platform: PlatformConfig{
			platform.Config{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Config",
					APIVersion: "storage.loft.sh/v1",
				},
				VirtualClusterAccessPointCertificates: make(map[string]platform.VirtualClusterCertificatesEntry),
			},
		},
		Manager: ManagerConfig{
			Type: manager.Helm,
		},
	}
}

// Read returns the current config by trying to read it from the given config path.
// It returns a new default config if there have been any errors during the read.
func Read(path string, log log.Logger) *Config {
	singleConfigMutex.RLock()
	defer singleConfigMutex.RUnlock()

	singleConfigOnce.Do(func() {
		if singleConfig == nil {
			cfg, err := readOrNew(path)
			if err != nil {
				log.Debugf("Failed to load local configuration file: %v", err)
			}
			singleConfig = cfg
		}
	})

	return singleConfig
}

// Write updates the current in-memory config and writes its content to the provided path.
func Write(path string, c *Config) error {
	singleConfigMutex.Lock()
	defer singleConfigMutex.Unlock()

	singleConfig = c

	err := os.MkdirAll(filepath.Dir(path), 0755)
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

func PrintManagerInfo(verb string, mngr manager.Type, log log.Logger) {
	// only print this to stderr
	log = log.ErrorStreamOnly()

	if mngr == manager.Helm {
		log.Infof("Using vCluster manager 'helm' to %s your virtual clusters, which means the vCluster CLI is running helm commands directly", verb)
		log.Info("If you prefer to use the vCluster platform API instead, use the flag '--manager platform' or run 'vcluster use manager platform' to change the default")
	} else {
		log.Infof("Using vCluster manager 'platform' to %s your virtual clusters, which means the CLI is using the vCluster platform API instead of helm", verb)
		log.Info("If you prefer to use helm instead, use the flag '--manager helm' or run 'vcluster use manager helm' to change the default")
	}
}

func DefaultConfigFilePath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, DirName, FileName), nil
}

func readOrNew(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return New(), fmt.Errorf("failed to load vcluster configuration file from %s, falling back to default configuration, following error occurred: %w", path, err)
	}
	stat, err := file.Stat()
	if err != nil {
		return New(), fmt.Errorf("failed to load vcluster configuration file from %s, falling back to default configuration, following error occurred: %w", path, err)
	}
	if stat.IsDir() {
		return New(), fmt.Errorf("failed to load vcluster configuration file %s, falling back to default configuration, this path is a directory", path)
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
