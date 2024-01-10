package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"time"

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
	currentNamespace, targetNamespace string,
	virtualKubeConfig *rest.Config,
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

	// build the start request
	startRequest, err := m.buildStartRequest(currentNamespace, targetNamespace, virtualKubeConfig, physicalKubeConfig, syncerConfig, options)
	if err != nil {
		return fmt.Errorf("build start request: %w", err)
	}

	// after loading all plugins we start them
	for _, vClusterPlugin := range m.Plugins {
		// start the plugin
		_, err = vClusterPlugin.GRPCClient.Start(ctx, startRequest)
		if err != nil {
			return fmt.Errorf("error starting plugin %s: %w", vClusterPlugin.Path, err)
		}

		// get client hooks
		clientHooks, err := vClusterPlugin.GRPCClient.GetClientHooks(ctx, &pluginv2.GetClientHooks_Request{})
		if err != nil {
			return fmt.Errorf("error retrieving client hooks for plugin %s: %w", vClusterPlugin.Path, err)
		}

		// add client hooks
		err = m.addClientHooks(vClusterPlugin, clientHooks.ClientHooks)
		if err != nil {
			return fmt.Errorf("error adding client hook for plugin %s: %w", vClusterPlugin.Path, err)
		}
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

func (m *Manager) addClientHooks(vClusterPlugin *vClusterPlugin, clientHooks []*pluginv2.GetClientHooks_ClientHook) error {
	for _, clientHookInfo := range clientHooks {
		if clientHookInfo.ApiVersion == "" {
			return fmt.Errorf("api version is empty in plugin %s hook", vClusterPlugin.Path)
		} else if clientHookInfo.Kind == "" {
			return fmt.Errorf("kind is empty in plugin %s hook", vClusterPlugin.Path)
		}

		for _, t := range clientHookInfo.Types {
			if t == "" {
				continue
			}

			versionKindType := plugintypes.VersionKindType{
				APIVersion: clientHookInfo.ApiVersion,
				Kind:       clientHookInfo.Kind,
				Type:       t,
			}

			m.ClientHooks[versionKindType] = append(m.ClientHooks[versionKindType], vClusterPlugin)
		}

		klog.Infof("Register client hook for %s %s in plugin %s", clientHookInfo.ApiVersion, clientHookInfo.Kind, vClusterPlugin.Path)
	}

	return nil
}

func (m *Manager) buildStartRequest(
	currentNamespace, targetNamespace string,
	virtualKubeConfig *rest.Config,
	physicalKubeConfig *rest.Config,
	syncerConfig *clientcmdapi.Config,
	options *options.VirtualClusterOptions,
) (*pluginv2.Start_Request, error) {
	// Context options
	encodedOptions, err := json.Marshal(options)
	if err != nil {
		return nil, fmt.Errorf("marshal options: %w", err)
	}

	// Virtual client config
	convertedVirtualConfig, err := kubeconfig.ConvertRestConfigToClientConfig(virtualKubeConfig)
	if err != nil {
		return nil, fmt.Errorf("convert virtual client config: %w", err)
	}
	rawVirtualConfig, err := convertedVirtualConfig.RawConfig()
	if err != nil {
		return nil, fmt.Errorf("convert virtual client config: %w", err)
	}
	virtualConfigBytes, err := clientcmd.Write(rawVirtualConfig)
	if err != nil {
		return nil, fmt.Errorf("marshal virtual client config: %w", err)
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

	return &pluginv2.Start_Request{
		VirtualClusterConfig:  string(virtualConfigBytes),
		PhysicalClusterConfig: string(phyisicalConfigBytes),
		SyncerConfig:          string(syncerConfigBytes),
		TargetNamespace:       targetNamespace,
		CurrentNamespace:      currentNamespace,
		Options:               string(encodedOptions),
	}, nil
}

func (m *Manager) loadPlugin(pluginPath string) error {
	// connect to plugin
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion: 1,
		},
		Plugins: map[string]plugin.Plugin{
			"plugin": &GRPCProviderPlugin{},
		},
		Cmd:              exec.Command(pluginPath),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
	})

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return err
	}

	// Request the plugin
	raw, err := rpcClient.Dispense("plugin")
	if err != nil {
		client.Kill()
		return err
	}

	// add to loaded plugins
	m.Plugins = append(m.Plugins, &vClusterPlugin{
		Path:       pluginPath,
		Client:     client,
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
		// is dir or executable?
		if pluginName.IsDir() {
			// check if plugin binary is there
			pluginPath := path.Join(m.PluginFolder, pluginName.Name(), "plugin")
			_, err := os.Stat(pluginPath)
			if err != nil {
				klog.FromContext(ctx).Error(fmt.Errorf("error loading plugin %s: %w", pluginPath, err), "error loading plugin")
				continue
			}

			pluginPaths = append(pluginPaths, pluginPath)
		} else {
			pluginPaths = append(pluginPaths, path.Join(m.PluginFolder, pluginName.Name()))
		}
	}

	return pluginPaths, nil
}
