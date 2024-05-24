package config

import (
	"fmt"
	"os"

	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/strvals"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

func ParseConfig(path, name string, setValues []string) (*VirtualClusterConfig, error) {
	// check if name is empty
	if name == "" {
		return nil, fmt.Errorf("empty vCluster name")
	}

	// read config file
	rawFile, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// apply set values
	rawFile, err = applySetValues(rawFile, setValues)
	if err != nil {
		return nil, fmt.Errorf("apply set values: %w", err)
	}

	// create a new strict decoder
	rawConfig := &config.Config{}
	err = yaml.UnmarshalStrict(rawFile, rawConfig)
	if err != nil {
		fmt.Printf("%#+v\n", errors.Unwrap(err))
		return nil, err
	}

	// build config
	retConfig := &VirtualClusterConfig{
		Config:              *rawConfig,
		Name:                name,
		ControlPlaneService: name,
	}
	if name == "" {
		return nil, fmt.Errorf("environment variable VCLUSTER_NAME is not defined")
	}

	// validate config
	err = ValidateConfigAndSetDefaults(retConfig)
	if err != nil {
		return nil, err
	}

	return retConfig, nil
}

func applySetValues(rawConfig []byte, setValues []string) ([]byte, error) {
	if len(setValues) == 0 {
		return rawConfig, nil
	}

	// parse raw config
	rawConfigMap := map[string]interface{}{}
	err := yaml.Unmarshal(rawConfig, &rawConfigMap)
	if err != nil {
		return nil, fmt.Errorf("parse raw config: %w", err)
	}

	// merge set
	for _, set := range setValues {
		err = strvals.ParseInto(set, rawConfigMap)
		if err != nil {
			return nil, fmt.Errorf("apply --set %s: %w", set, err)
		}
	}

	// marshal again
	rawConfig, err = yaml.Marshal(rawConfigMap)
	if err != nil {
		return nil, fmt.Errorf("marshal config bytes: %w", err)
	}

	return rawConfig, nil
}
