package oci

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
	"github.com/docker/docker/pkg/homedir"
	perrors "github.com/pkg/errors"
)

const dockerFileFolder = ".docker"
const AzureContainerRegistryUsername = "00000000-0000-0000-0000-000000000000"

func GetAuthConfig(host string) (*Credentials, error) {
	dockerConfig, err := loadDockerConfig()
	if err != nil {
		return nil, err
	}

	if host == "registry-1.docker.io" {
		host = "https://index.docker.io/v1/"
	}
	ac, err := dockerConfig.GetAuthConfig(host)
	if err != nil {
		return nil, perrors.Wrapf(err, "get auth config for host %s", host)
	}

	return prepareCredentials(host, ac), nil
}

func prepareCredentials(host string, authConfig types.AuthConfig) *Credentials {
	if authConfig.Password == "" && authConfig.IdentityToken != "" {
		authConfig.Password = authConfig.IdentityToken
	}

	if authConfig.Username == "" && isAzureContainerRegistry(authConfig.ServerAddress) {
		authConfig.Username = AzureContainerRegistryUsername
	}

	return &Credentials{
		ServerURL: host,
		Username:  authConfig.Username,
		Secret:    authConfig.Password,
	}
}

func loadDockerConfig() (*configfile.ConfigFile, error) {
	configDir := os.Getenv("DOCKER_CONFIG")
	if configDir == "" {
		configDir = filepath.Join(homedir.Get(), dockerFileFolder)
	}

	return config.Load(configDir)
}

func isAzureContainerRegistry(serverAddress string) bool {
	return strings.HasSuffix(serverAddress, "azurecr.io")
}

// Credentials holds the information shared between docker and the credentials store.
type Credentials struct {
	ServerURL string
	Username  string
	Secret    string
}

func (c *Credentials) AuthToken() string {
	if c.Username != "" {
		return c.Username + ":" + c.Secret
	}
	return c.Secret
}
