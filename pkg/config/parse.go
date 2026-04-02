package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/strvals"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/stringutil"
	"github.com/pkg/errors"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/yaml"
)

func ParseConfig(path, name string, setValues []string) (*VirtualClusterConfig, error) {
	// check if name is empty
	if name == "" {
		return nil, fmt.Errorf("empty vCluster name")
	}

	// figure out the config path
	if path == constants.DefaultVClusterConfigLocation {
		_, err := os.Stat(constants.StandaloneDefaultConfigPath)
		if err == nil {
			path = constants.StandaloneDefaultConfigPath
		}
	}

	// For standalone, we load config from:
	// 1. User provided path (via --config flag)
	// 2. If not set, then config is expected in the /etc/vcluster/vcluster.yaml
	// 3. If it does not exist, then fallback to /var/lib/vcluster/config.yaml
	klog.Info("Reading vcluster.yaml config from", "path", path)
	rawFile, err := os.ReadFile(path)
	if err != nil {
		if !(path == constants.DefaultVClusterConfigLocation || path == constants.StandaloneDefaultConfigPath) && os.IsNotExist(err) {
			// if config does not exist in the path specified by user, do not fallback to default locations, just fail
			return nil, err
		}
	}

	cfg, err := ParseConfigBytes(rawFile, name, setValues)
	if err != nil {
		return nil, fmt.Errorf("parsing config bytes: %w", err)
	}

	cfg.Path = path
	if path != constants.DefaultVClusterConfigLocation {
		// config.ParseConfig does not apply Helm value defaults, so Standalone.Enabled,
		// PrivateNodes.Enabled, and DataDir may be unset even though we are clearly in
		// standalone mode. Mirror what vcluster-pro/pkg/standalone/config does.
		cfg.ControlPlane.Standalone.Enabled = true
		cfg.PrivateNodes.Enabled = true
		if cfg.ControlPlane.Standalone.DataDir == "" {
			cfg.ControlPlane.Standalone.DataDir = "/var/lib/vcluster"
		}
	}

	return cfg, nil
}

func ParseConfigBytes(data []byte, name string, setValues []string) (*VirtualClusterConfig, error) {
	// apply set values
	rawFile, err := applySetValues(data, setValues)
	if err != nil {
		return nil, fmt.Errorf("apply set values: %w", err)
	}

	// create a new strict decoder
	rawConfig := map[string]interface{}{}
	if len(rawFile) > 0 {
		err = yaml.UnmarshalStrict(rawFile, rawConfig)
		if err != nil {
			fmt.Printf("%#+v\n", errors.Unwrap(err))
			return nil, err
		}
	}

	// merge with default config
	defaultConfig, err := vclusterconfig.NewDefaultConfig()
	if err != nil {
		return nil, err
	}
	defaultConfigMap, err := convertToMap(defaultConfig)
	if err != nil {
		return nil, err
	}

	// merge the configs
	outConfigMap := strvals.MergeMaps(defaultConfigMap, rawConfig)
	raw, err := json.Marshal(outConfigMap)
	if err != nil {
		return nil, err
	}
	outConfig := &vclusterconfig.Config{}
	err = json.Unmarshal(raw, outConfig)
	if err != nil {
		return nil, err
	}

	// build config
	retConfig := &VirtualClusterConfig{
		Config: *outConfig,
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

func convertToMap(config *vclusterconfig.Config) (map[string]interface{}, error) {
	raw, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	out := map[string]interface{}{}
	err = json.Unmarshal(raw, &out)
	if err != nil {
		return nil, err
	}

	return out, nil
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
