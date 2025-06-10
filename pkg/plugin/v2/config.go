package v2

import "encoding/json"

// InitConfig is the config the syncer sends to the plugin
type InitConfig struct {
	Pro          InitConfigPro `json:"pro,omitempty"`
	SyncerConfig []byte        `json:"syncerConfig,omitempty"`

	WorkloadConfig     []byte `json:"workloadConfig,omitempty"`
	ControlPlaneConfig []byte `json:"controlPlaneConfig,omitempty"`

	Config []byte `json:"config,omitempty"`

	WorkingDir string `json:"workingDir,omitempty"`

	Port int `json:"port,omitempty"`

	// Legacy fields we still need to support
	Options               []byte `json:"options,omitempty"`
	CurrentNamespace      string `json:"currentNamespace,omitempty"`
	PhysicalClusterConfig []byte `json:"physicalClusterConfig,omitempty"`
}

// InitConfigPro is used to signal the plugin if vCluster.Pro is enabled and what features are allowed
type InitConfigPro struct {
	Enabled  bool            `json:"enabled,omitempty"`
	Features map[string]bool `json:"features,omitempty"`
}

// PluginConfig is the config the plugin sends back to the syncer
type PluginConfig struct {
	ClientHooks  []*ClientHook                `json:"clientHooks,omitempty"`
	Interceptors map[string][]InterceptorRule `json:"interceptors,omitempty"`
}

type ClientHook struct {
	APIVersion string   `json:"apiVersion,omitempty"`
	Kind       string   `json:"kind,omitempty"`
	Types      []string `json:"types,omitempty"`
}

type InterceptorRule struct {
	APIGroups       []string `json:"apiGroups,omitempty"`
	Resources       []string `json:"resources,omitempty"`
	ResourceNames   []string `json:"resourceNames,omitempty"`
	NonResourceURLs []string `json:"nonResourceURLs,omitempty"`
	Verbs           []string `json:"verbs,omitempty"`
}

func parsePluginConfig(config string) (*PluginConfig, error) {
	pluginConfig := &PluginConfig{}
	err := json.Unmarshal([]byte(config), pluginConfig)
	if err != nil {
		return nil, err
	}

	return pluginConfig, nil
}
