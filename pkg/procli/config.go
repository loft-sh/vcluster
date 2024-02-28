package procli

import (
	"fmt"
	"path/filepath"

	"github.com/loft-sh/vcluster/pkg/util/cliconfig"
	homedir "github.com/mitchellh/go-homedir"
)

const (
	VClusterProFolder = "pro"
)

// ConfigFilePath returns the path to the loft config file
func ConfigFilePath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", fmt.Errorf("failed to open vcluster pro configuration file from, unable to detect $HOME directory, falling back to default configuration, following error occurred: %w", err)
	}

	return filepath.Join(home, cliconfig.VClusterFolder, VClusterProFolder, cliconfig.ConfigFileName), nil
}
