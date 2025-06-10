package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/loft-sh/log"
	homedir "github.com/mitchellh/go-homedir"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DirName  = ".vcluster"
	FileName = "config.json"

	HelmDriver     DriverType = "helm"
	PlatformDriver DriverType = "platform"
)

var singleConfig *CLI

// New creates a new default config
func New() *CLI {
	return &CLI{
		TelemetryDisabled: false,
		Driver: Driver{
			Type: HelmDriver,
		},
		Platform: Platform{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Config",
				APIVersion: "storage.loft.sh/v1",
			},
		},
	}
}

func (c *CLI) Save() error {
	path := ""
	if c != nil {
		path = c.path
	} else {
		var err error
		path, err = DefaultFilePath()
		if err != nil {
			return err
		}
	}

	return Write(path, c)
}

func (c *CLI) Delete() error {
	if c == nil || c.path == "" {
		return errors.New("nil config path")
	}

	file, err := os.Open(c.path)
	if err != nil {
		return fmt.Errorf("failed to load vcluster configuration file from %q : %w", c.path, err)
	}
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to load vcluster configuration file from %q: %w", c.path, err)
	}
	if stat.IsDir() {
		return fmt.Errorf("failed to load vcluster configuration file %q", c.path)
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("read all: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(fileBytes))
	decoder.DisallowUnknownFields()
	tryRead := &CLI{}
	err = decoder.Decode(tryRead)
	if err != nil {
		return fmt.Errorf("failed to unmarshall vcluster configuration from %q: %w", c.path, err)
	}

	// delete file at path
	err = os.Remove(c.path)
	if err != nil {
		return fmt.Errorf("failed to delete configuration file at %q: %w", c.path, err)
	}
	return nil
}

// Read returns the current config by trying to read it from the given config path.
// It returns a new default config if there have been any errors during the read.
func Read(path string, log log.Logger) *CLI {
	if singleConfig == nil {
		var err error
		singleConfig, err = readOrNewConfig(path)
		if err != nil {
			log.Debugf("Failed to load local configuration file: %v", err)
		}
	}
	if singleConfig == nil {
		singleConfig = New()
	}

	singleConfig.path = path

	return singleConfig
}

// Write writes the config content to the provided path.
func Write(path string, c *CLI) error {
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

func PrintDriverInfo(verb string, driver DriverType, log log.Logger) {
	// only print this to stderr
	log = log.ErrorStreamOnly()

	if driver == HelmDriver {
		log.Infof("Using vCluster driver 'helm' to %s your virtual clusters, which means the vCluster CLI is running helm commands directly", verb)
		log.Info("If you prefer to use the vCluster platform API instead, use the flag '--driver platform' or run 'vcluster use driver platform' to change the default")
	} else {
		log.Infof("Using vCluster driver 'platform' to %s your virtual clusters, which means the CLI is using the vCluster platform API instead of helm", verb)
		log.Info("If you prefer to use helm instead, use the flag '--driver helm' or run 'vcluster use driver helm' to change the default")
	}
}

func ParseDriverType(driver string) (DriverType, error) {
	switch driver {
	case "", "helm":
		return HelmDriver, nil
	case "platform":
		return PlatformDriver, nil
	default:
		return "", fmt.Errorf("invalid driver type: %q, only \"helm\" or \"platform\" are valid", driver)
	}
}

func DefaultFilePath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, DirName, FileName), nil
}

func readOrNewConfig(path string) (*CLI, error) {
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

	c := &CLI{}
	err = json.Unmarshal(bytes, &c)
	if err != nil {
		return New(), fmt.Errorf("failed to unmarshall vcluster configuration from %s file, falling back to default configuration, following error occurred: %w", path, err)
	}

	return c, nil
}
