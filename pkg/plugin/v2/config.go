package v2

import "encoding/json"

// InitConfig is the config the syncer sends to the plugin
type InitConfig struct {
	Pro                   InitConfigPro `json:"pro,omitempty"`
	PhysicalClusterConfig []byte        `json:"physicalClusterConfig,omitempty"`
	SyncerConfig          []byte        `json:"syncerConfig,omitempty"`
	CurrentNamespace      string        `json:"currentNamespace,omitempty"`

	Config  []byte `json:"config,omitempty"`
	Options []byte `json:"options,omitempty"`

	WorkingDir string `json:"workingDir,omitempty"`

	Port int `json:"port,omitempty"`
}

// InitConfigPro is used to signal the plugin if vCluster.Pro is enabled and what features are allowed
type InitConfigPro struct {
	Enabled  bool            `json:"enabled,omitempty"`
	Features map[string]bool `json:"features,omitempty"`
}

// PluginConfig is the config the plugin sends back to the syncer
type PluginConfig struct {
	ClientHooks  []*ClientHook     `json:"clientHooks,omitempty"`
	Interceptors InterceptorConfig `json:"InterceptorConfig,omitempty"`
}

type InterceptorConfig struct {
	Port         int           `json:"port"`
	Interceptors []Interceptor `json:"interceptors"`
}

type ClientHook struct {
	APIVersion string   `json:"apiVersion,omitempty"`
	Kind       string   `json:"kind,omitempty"`
	Types      []string `json:"types,omitempty"`
}

type Interceptor struct {
	APIGroups       []string `json:"apiGroups,omitempty"`
	HandlerName     string   `json:"name,omitempty"`
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

/*
 // PolicyRule holds information that describes a policy rule, but does not contain information
// about who the rule applies to or which namespace the rule applies to.
type PolicyRule struct {
	// Verbs is a list of Verbs that apply to ALL the ResourceKinds contained in this rule. '*' represents all verbs.
	Verbs []string `json:"verbs" protobuf:"bytes,1,rep,name=verbs"`

	// APIGroups is the name of the APIGroup that contains the resources.  If multiple API groups are specified, any action requested against one of
	// the enumerated resources in any API group will be allowed. "" represents the core API group and "*" represents all API groups.
	// +optional
	APIGroups []string `json:"apiGroups,omitempty" protobuf:"bytes,2,rep,name=apiGroups"`
	// Resources is a list of resources this rule applies to. '*' represents all resources.
	// +optional
	Resources []string `json:"resources,omitempty" protobuf:"bytes,3,rep,name=resources"`
	// ResourceNames is an optional white list of names that the rule applies to.  An empty set means that everything is allowed.
	// +optional
	ResourceNames []string `json:"resourceNames,omitempty" protobuf:"bytes,4,rep,name=resourceNames"`

	// NonResourceURLs is a set of partial urls that a user should have access to.  *s are allowed, but only as the full, final step in the path
	// Since non-resource URLs are not namespaced, this field is only applicable for ClusterRoles referenced from a ClusterRoleBinding.
	// Rules can either apply to API resources (such as "pods" or "secrets") or non-resource URL paths (such as "/api"),  but not both.
	// +optional
	NonResourceURLs []string `json:"nonResourceURLs,omitempty" protobuf:"bytes,5,rep,name=nonResourceURLs"`
}
*/
