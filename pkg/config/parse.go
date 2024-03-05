package config

import (
	"fmt"
	"os"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/vcluster/config"
)

func ParseConfig(path string) (*VirtualClusterConfig, error) {
	rawFile, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	rawConfig := &config.Config{}
	err = yaml.Unmarshal(rawFile, rawConfig)
	if err != nil {
		return nil, err
	}

	retConfig, err := Convert(rawConfig)
	if err != nil {
		return nil, err
	}

	err = ValidateConfig(retConfig)
	if err != nil {
		return nil, err
	}

	return retConfig, nil
}

func Convert(config *config.Config) (*VirtualClusterConfig, error) {
	vClusterName := os.Getenv("VCLUSTER_NAME")
	if vClusterName == "" {
		return nil, fmt.Errorf("environment variable VCLUSTER_NAME is not defined")
	}

	retConfig := &VirtualClusterConfig{
		Config: *config,
		Name:   vClusterName,
	}

	// convert legacy options

	return retConfig, nil
}
