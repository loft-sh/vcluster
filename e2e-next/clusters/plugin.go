package clusters

import _ "embed"

// PluginVCluster runs legacy v1/v2 plugin tests (bootstrap-with-deployment, hooks, import-secrets).
// Plugin example images must be multi-arch (amd64 + arm64) for local testing on macOS ARM.
// If a plugin image is amd64-only, Kind on Apple Silicon will fail with "exec format error".

//go:embed vcluster-plugin.yaml
var pluginVClusterYAML string

var (
	PluginVClusterName = "plugin-vcluster"
	PluginVCluster     = register(PluginVClusterName, pluginVClusterYAML)
)
