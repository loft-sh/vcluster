package config

import (
	"encoding/json"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/strvals"
	"sigs.k8s.io/yaml"
)

func mergeConfigBytesWithDefaults(rawConfig []byte) ([]byte, error) {
	defaultConfig, err := vclusterconfig.NewDefaultConfig()
	if err != nil {
		return nil, err
	}

	defaultConfigMap, err := convertConfigToMap(defaultConfig)
	if err != nil {
		return nil, err
	}

	overrideMap := map[string]interface{}{}
	if len(rawConfig) > 0 {
		if err := yaml.Unmarshal(rawConfig, &overrideMap); err != nil {
			return nil, err
		}
	}

	return json.Marshal(strvals.MergeMaps(defaultConfigMap, overrideMap))
}

func convertConfigToMap(config *vclusterconfig.Config) (map[string]interface{}, error) {
	raw, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	out := map[string]interface{}{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}

	return out, nil
}
