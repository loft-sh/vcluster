package docker

import (
	"os"
	"path/filepath"

	"github.com/docker/docker/pkg/homedir"

	dockerconfig "github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
	"github.com/pkg/errors"
)

const dockerFileFolder = ".docker"

// Config is the interface to interact with the docker config
type Config interface {
	// Store saves credentials for the given registry into the local config file
	Store(registry string, authConfig types.AuthConfig) error

	// Save persists the locally changed config file to file
	Save() error
}

// NewDockerConfig creates a new docker client
func NewDockerConfig() (Config, error) {
	configFile, err := loadDockerConfig()
	if err != nil {
		return nil, err
	}

	return &config{
		DockerConfig: configFile,
	}, nil
}

type config struct {
	DockerConfig *configfile.ConfigFile
}

func (c *config) Store(registry string, authConfig types.AuthConfig) error {
	if registry == "" {
		return nil
	}

	err := c.DockerConfig.GetCredentialsStore(registry).Store(authConfig)
	if err != nil {
		return errors.Wrapf(err, "store credentials for registry %s", registry)
	}

	return nil
}

func (c *config) Save() error {
	return c.DockerConfig.Save()
}

func loadDockerConfig() (*configfile.ConfigFile, error) {
	configDir := os.Getenv("DOCKER_CONFIG")
	if configDir == "" {
		configDir = filepath.Join(homedir.Get(), dockerFileFolder)
	}

	return dockerconfig.Load(configDir)
}
