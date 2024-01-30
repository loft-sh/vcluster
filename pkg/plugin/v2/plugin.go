package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"

	"github.com/ghodss/yaml"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	plugintypes "github.com/loft-sh/vcluster/pkg/plugin/types"
	"github.com/loft-sh/vcluster/pkg/plugin/v2/pluginv2"
	"github.com/loft-sh/vcluster/pkg/setup/options"
	"github.com/loft-sh/vcluster/pkg/util/kubeconfig"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

const PluginConfigEnv = "PLUGIN_CONFIG"

func NewManager() *Manager {
	pluginFolder := os.Getenv("PLUGIN_FOLDER")
	if pluginFolder == "" {
		pluginFolder = "/plugins"
	}

	return &Manager{
		PluginFolder: pluginFolder,
		ClientHooks:  map[plugintypes.VersionKindType][]*vClusterPlugin{},
	}
}

type Manager struct {
	// PluginFolder where to load plugins from
	PluginFolder string

	// Plugins that were loaded
	Plugins []*vClusterPlugin

	// ClientHooks that were loaded
	ClientHooks map[plugintypes.VersionKindType][]*vClusterPlugin
}

type vClusterPlugin struct {
	// Path is the path where the plugin was loaded from
	Path string

	// Client is the plugin client
	Client *plugin.Client

	// GRPCClient is the direct grpc client
	GRPCClient pluginv2.PluginClient
}

func (m *Manager) Start(
	ctx context.Context,
	currentNamespace string,
	physicalKubeConfig *rest.Config,
	syncerConfig *clientcmdapi.Config,
	options *options.VirtualClusterOptions,
) error {
	// try to search for plugins
	plugins, err := m.findPlugins(ctx)
	if err != nil {
		return fmt.Errorf("find plugins: %w", err)
	} else if len(plugins) == 0 {
		return nil
	}

	// loop over plugins and load them
	for _, pluginPath := range plugins {
		err = m.loadPlugin(pluginPath)
		if err != nil {
			return fmt.Errorf("start plugin %s: %w", pluginPath, err)
		}
	}

	// after loading all plugins we start them
	for _, vClusterPlugin := range m.Plugins {
		// build the start request
		initRequest, err := m.buildInitRequest(filepath.Dir(vClusterPlugin.Path), currentNamespace, physicalKubeConfig, syncerConfig, options)
		if err != nil {
			return fmt.Errorf("build start request: %w", err)
		}

		// start the plugin
		_, err = vClusterPlugin.GRPCClient.Initialize(ctx, initRequest)
		if err != nil {
			return fmt.Errorf("error starting plugin %s: %w", vClusterPlugin.Path, err)
		}

		// get plugin config
		pluginConfigResponse, err := vClusterPlugin.GRPCClient.GetPluginConfig(ctx, &pluginv2.GetPluginConfig_Request{})
		if err != nil {
			return fmt.Errorf("error retrieving client hooks for plugin %s: %w", vClusterPlugin.Path, err)
		}

		// parse plugin config
		pluginConfig, err := parsePluginConfig(pluginConfigResponse.Config)
		if err != nil {
			return fmt.Errorf("error parsing plugin config: %w", err)
		}

		// register client hooks
		err = m.registerClientHooks(vClusterPlugin, pluginConfig.ClientHooks)
		if err != nil {
			return fmt.Errorf("error adding client hook for plugin %s: %w", vClusterPlugin.Path, err)
		}

		klog.FromContext(ctx).Info("Successfully loaded plugin", "plugin", vClusterPlugin.Path)
	}

	return nil
}

func (m *Manager) MutateObject(ctx context.Context, obj client.Object, hookType string, scheme *runtime.Scheme) error {
	gvk, err := apiutil.GVKForObject(obj, scheme)
	if err != nil {
		return err
	}

	apiVersion, kind := gvk.ToAPIVersionAndKind()
	versionKindType := plugintypes.VersionKindType{
		APIVersion: apiVersion,
		Kind:       kind,
		Type:       hookType,
	}
	clientHooks := m.ClientHooks[versionKindType]
	if len(clientHooks) == 0 {
		return nil
	}

	encodedObj, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("encode object: %w", err)
	}

	for _, clientHook := range clientHooks {
		encodedObj, err = m.mutateObject(ctx, versionKindType, encodedObj, clientHook)
		if err != nil {
			return err
		}
	}

	err = json.Unmarshal(encodedObj, obj)
	if err != nil {
		return fmt.Errorf("decode object: %w", err)
	}

	return nil
}

func (m *Manager) mutateObject(ctx context.Context, versionKindType plugintypes.VersionKindType, obj []byte, plugin *vClusterPlugin) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	klog.FromContext(ctx).V(1).Info("calling plugin to mutate object", "plugin", plugin.Path, "apiVersion", versionKindType.APIVersion, "kind", versionKindType.Kind)
	mutateResult, err := plugin.GRPCClient.Mutate(ctx, &pluginv2.Mutate_Request{
		ApiVersion: versionKindType.APIVersion,
		Kind:       versionKindType.Kind,
		Object:     string(obj),
		Type:       versionKindType.Type,
	})
	if err != nil {
		return nil, fmt.Errorf("call plugin mutate %s: %w", plugin.Path, err)
	}

	if mutateResult.Mutated {
		return []byte(mutateResult.Object), nil
	}
	return obj, nil
}

func (m *Manager) SetLeader(ctx context.Context) error {
	for _, vClusterPlugin := range m.Plugins {
		_, err := vClusterPlugin.GRPCClient.SetLeader(ctx, &pluginv2.SetLeader_Request{})
		if err != nil {
			return fmt.Errorf("error setting leader in plugin %s: %w", vClusterPlugin.Path, err)
		}
	}

	return nil
}

func (m *Manager) HasClientHooksForType(versionKindType plugintypes.VersionKindType) bool {
	return len(m.ClientHooks[versionKindType]) > 0
}

func (m *Manager) HasClientHooks() bool {
	return len(m.ClientHooks) > 0
}

func (m *Manager) HasPlugins() bool {
	return len(m.Plugins) > 0
}

func (m *Manager) registerClientHooks(vClusterPlugin *vClusterPlugin, clientHooks []*ClientHook) error {
	for _, clientHookInfo := range clientHooks {
		if clientHookInfo.APIVersion == "" {
			return fmt.Errorf("api version is empty in plugin %s hook", vClusterPlugin.Path)
		} else if clientHookInfo.Kind == "" {
			return fmt.Errorf("kind is empty in plugin %s hook", vClusterPlugin.Path)
		}

		for _, t := range clientHookInfo.Types {
			if t == "" {
				continue
			}

			versionKindType := plugintypes.VersionKindType{
				APIVersion: clientHookInfo.APIVersion,
				Kind:       clientHookInfo.Kind,
				Type:       t,
			}

			m.ClientHooks[versionKindType] = append(m.ClientHooks[versionKindType], vClusterPlugin)
		}

		klog.Infof("Register client hook for %s %s in plugin %s", clientHookInfo.APIVersion, clientHookInfo.Kind, vClusterPlugin.Path)
	}

	return nil
}

func (m *Manager) buildInitRequest(
	workingDir,
	currentNamespace string,
	physicalKubeConfig *rest.Config,
	syncerConfig *clientcmdapi.Config,
	options *options.VirtualClusterOptions,
) (*pluginv2.Initialize_Request, error) {
	// Context options
	encodedOptions, err := json.Marshal(options)
	if err != nil {
		return nil, fmt.Errorf("marshal options: %w", err)
	}

	// Physical client config
	convertedPhysicalConfig, err := kubeconfig.ConvertRestConfigToClientConfig(physicalKubeConfig)
	if err != nil {
		return nil, fmt.Errorf("convert physical client config: %w", err)
	}
	rawPhysicalConfig, err := convertedPhysicalConfig.RawConfig()
	if err != nil {
		return nil, fmt.Errorf("convert physical client config: %w", err)
	}
	phyisicalConfigBytes, err := clientcmd.Write(rawPhysicalConfig)
	if err != nil {
		return nil, fmt.Errorf("marshal physical client config: %w", err)
	}

	// Syncer client config
	syncerConfigBytes, err := clientcmd.Write(*syncerConfig)
	if err != nil {
		return nil, fmt.Errorf("marshal syncer client config: %w", err)
	}

	// marshal init config
	initConfig, err := json.Marshal(&InitConfig{
		PhysicalClusterConfig: phyisicalConfigBytes,
		SyncerConfig:          syncerConfigBytes,
		CurrentNamespace:      currentNamespace,
		Options:               encodedOptions,
		WorkingDir:            workingDir,
	})
	if err != nil {
		return nil, fmt.Errorf("error encoding init config: %w", err)
	}

	return &pluginv2.Initialize_Request{
		Config: string(initConfig),
	}, nil
}

func (m *Manager) loadPlugin(pluginPath string) error {
	// Create an hclog.Logger
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "plugin",
		Output: os.Stdout,
		Level:  hclog.Info,
	})

	// build command
	cmd, err := buildCommand(pluginPath)
	if err != nil {
		return err
	}

	// connect to plugin
	pluginClient := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: HandshakeConfig,
		Logger:          logger,
		Plugins: map[string]plugin.Plugin{
			"plugin": &GRPCProviderPlugin{},
		},
		Cmd:              cmd,
		SyncStdout:       os.Stdout,
		SyncStderr:       os.Stderr,
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
	})

	// Connect via RPC
	rpcClient, err := pluginClient.Client()
	if err != nil {
		pluginClient.Kill()
		return err
	}

	// Request the plugin
	raw, err := rpcClient.Dispense("plugin")
	if err != nil {
		pluginClient.Kill()
		return err
	}

	// add to loaded plugins
	m.Plugins = append(m.Plugins, &vClusterPlugin{
		Path:       pluginPath,
		Client:     pluginClient,
		GRPCClient: raw.(pluginv2.PluginClient),
	})

	return nil
}

func (m *Manager) findPlugins(ctx context.Context) ([]string, error) {
	_, err := os.Stat(m.PluginFolder)
	if err != nil {
		// we cannot stat plugin folder, skip reading plugins
		klog.FromContext(ctx).V(1).Info("Error reading plugin folder", "error", err)
		return nil, nil
	}

	plugins, err := os.ReadDir(m.PluginFolder)
	if err != nil {
		return nil, fmt.Errorf("error reading")
	}

	pluginPaths := []string{}
	for _, pluginName := range plugins {
		// is dir?
		if !pluginName.IsDir() {
			continue
		}

		// check if plugin binary is there
		pluginPath := path.Join(m.PluginFolder, pluginName.Name(), "plugin")
		_, err := os.Stat(pluginPath)
		if err != nil {
			klog.FromContext(ctx).Error(fmt.Errorf("error loading plugin %s: %w", pluginPath, err), "error loading plugin")
			continue
		}

		pluginPaths = append(pluginPaths, pluginPath)
	}

	return pluginPaths, nil
}

func buildCommand(pluginPath string) (*exec.Cmd, error) {
	cmd := exec.Command(pluginPath)

	// check for plugin config
	pluginName := filepath.Base(filepath.Dir(pluginPath))
	pluginConfig := os.Getenv(PluginConfigEnv)
	if pluginConfig == "" {
		return cmd, nil
	}

	// try to parse yaml
	parsedConfig := map[string]interface{}{}
	err := yaml.Unmarshal([]byte(pluginConfig), &parsedConfig)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s: %w", PluginConfigEnv, err)
	}

	// check if plugin is there
	pluginConfigEncoded := ""
	if parsedConfig[pluginName] != nil {
		pluginConfigRaw, err := yaml.Marshal(parsedConfig[pluginName])
		if err != nil {
			return nil, fmt.Errorf("error marshalling plugin config: %w", err)
		}

		pluginConfigEncoded = string(pluginConfigRaw)
	}

	// add to plugin environment
	cmd.Env = append(os.Environ(), PluginConfigEnv+"="+pluginConfigEncoded)
	return cmd, nil
}
