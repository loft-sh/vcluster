package platform

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/loft-sh/vcluster/pkg/constants"
	homedir "github.com/mitchellh/go-homedir"
)

const (
	VClusterProFolder = "pro"
)

// ConfigFilePath returns the path to the loft config file
func ConfigFilePath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", fmt.Errorf("failed to open vCluster platform configuration file, unable to detect $HOME directory, falling back to default configuration, following error occurred: %w", err)
	}

	return filepath.Join(home, constants.VClusterFolder, constants.ConfigFileName), nil
}

func managerFilePath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", fmt.Errorf("failed to open vCluster platform manager file, unable to detect $HOME directory, falling back to default configuration, following error occurred: %w", err)
	}

	return filepath.Join(home, constants.VClusterFolder, constants.ManagerFileName), nil
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
