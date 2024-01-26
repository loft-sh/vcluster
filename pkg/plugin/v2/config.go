package v2

import "encoding/json"

// InitConfig is the config the syncer sends to the plugin
type InitConfig struct {
	PhysicalClusterConfig []byte `json:"physicalClusterConfig,omitempty"`
	SyncerConfig          []byte `json:"syncerConfig,omitempty"`
	CurrentNamespace      string `json:"currentNamespace,omitempty"`
	Options               []byte `json:"options,omitempty"`
	WorkingDir            string `json:"workingDir,omitempty"`
}

// PluginConfig is the config the plugin sends back to the syncer
type PluginConfig struct {
	ClientHooks []*ClientHook `json:"clientHooks,omitempty"`
}

type ClientHook struct {
	APIVersion string   `json:"apiVersion,omitempty"`
	Kind       string   `json:"kind,omitempty"`
	Types      []string `json:"types,omitempty"`
}

func parsePluginConfig(config string) (*PluginConfig, error) {
	pluginConfig := &PluginConfig{}
	err := json.Unmarshal([]byte(config), pluginConfig)
	if err != nil {
		return nil, err
	}

	return pluginConfig, nil
}
