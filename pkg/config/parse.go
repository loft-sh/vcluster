package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/strvals"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/stringutil"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/cache"
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

	cfg, err := ParseConfigBytes(rawFile, name, setValues)
	if err != nil {
		return nil, fmt.Errorf("parsing config bytes: %w", err)
	}

	cfg.Path = path

	return cfg, nil
}

func ParseConfigBytes(data []byte, name string, setValues []string) (*VirtualClusterConfig, error) {
	// apply set values
	rawFile, err := applySetValues(data, setValues)
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
		Config: *rawConfig,
		Name:   name,
	}

	// validate config
	err = ValidateConfigAndSetDefaults(retConfig)
	if err != nil {
		return nil, err
	}

	configLogger := loghelper.New("config")
	warnings := Lint(retConfig.Config)
	for _, warning := range warnings {
		configLogger.Infof("Warning: %s", warning)
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

func GetLocalCacheOptionsFromConfigMappings(mappings map[string]string, vClusterNamespace string) (cache.Options, bool) {
	defaultNamespaces := make(map[string]cache.Config)
	namespaces := parseHostNamespacesFromMappings(mappings, vClusterNamespace)
	if len(namespaces) == 1 {
		for _, k := range namespaces {
			if k == vClusterNamespace {
				// then there is no need to create custom manager
				return cache.Options{}, false
			}
		}
	}
	for _, ns := range namespaces {
		defaultNamespaces[ns] = cache.Config{}
	}
	return cache.Options{DefaultNamespaces: defaultNamespaces}, true
}

func parseHostNamespacesFromMappings(mappings map[string]string, vClusterNs string) []string {
	ret := make([]string, 0)
	for host := range mappings {
		if host == constants.VClusterNamespaceInHostMappingSpecialCharacter {
			ret = append(ret, vClusterNs)
		}
		parts := strings.Split(host, "/")
		if len(parts) != 2 {
			continue
		}

		if parts[0] == "" {
			// this means that the mapping key is e.g. "/my-cm-1",
			// then, we should append virtual cluster namespace
			ret = append(ret, vClusterNs)
			continue
		}
		hostNs := parts[0]
		ret = append(ret, hostNs)
	}
	return stringutil.RemoveDuplicates(ret)
}
