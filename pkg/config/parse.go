package config

import (
	"fmt"
	"os"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/vcluster/config"
)

func ParseConfig(path, name string) (*VirtualClusterConfig, error) {
	rawFile, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	rawConfig := &config.Config{}
	err = yaml.Unmarshal(rawFile, rawConfig)
	if err != nil {
		return nil, err
	}

	retConfig := &VirtualClusterConfig{
		Config: *rawConfig,
		Name:   name,
	}
	if name == "" {
		return nil, fmt.Errorf("environment variable VCLUSTER_NAME is not defined")
	}

	err = ValidateConfig(retConfig)
	if err != nil {
		return nil, err
	}

	return retConfig, nil
}
