package oci

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
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
	return config.Load(dockerConfigDir())
}

func dockerConfigDir() string {
	configDir := os.Getenv("DOCKER_CONFIG")
	if configDir != "" {
		return configDir
	}

	home, _ := os.UserHomeDir()
	// Preserve Docker's account lookup when HOME is unset on Unix.
	if home == "" && runtime.GOOS != "windows" {
		if current, err := user.Current(); err == nil {
			home = current.HomeDir
		}
	}
	return filepath.Join(home, dockerFileFolder)
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
