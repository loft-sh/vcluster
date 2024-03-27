package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/loft-sh/vcluster/pkg/config"
	plugintypes "github.com/loft-sh/vcluster/pkg/plugin/types"
	"github.com/loft-sh/vcluster/pkg/plugin/v2/pluginv2"
	"github.com/loft-sh/vcluster/pkg/util/kubeconfig"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
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

	client := &http.Client{Timeout: time.Second}

	return &Manager{
		PluginFolder:              pluginFolder,
		ClientHooks:               map[plugintypes.VersionKindType][]*vClusterPlugin{},
		ResourceInterceptorsPorts: map[plugintypes.GroupedResource]map[string]portHandlerName{},
		HTTPClient:                client,
	}
}

type Manager struct {
	// PluginFolder where to load plugins from
	PluginFolder string

	// Plugins that were loaded
	Plugins []*vClusterPlugin

	// ClientHooks that were loaded
	ClientHooks map[plugintypes.VersionKindType][]*vClusterPlugin

	// map to track the port that needs to be targeted for the interceptors
	ResourceInterceptorsPorts map[plugintypes.GroupedResource]map[string]portHandlerName
	// map to track the port that needs to be targeted for the non resource interceptors
	NonResourceInterceptorsPorts map[string]map[string]portHandlerName
	// ProFeatures are pro features to hand-over to the plugin
	ProFeatures map[string]bool

	HTTPClient requestDoer
}

type portHandlerName struct {
	handlerName string
	port        int
}

type requestDoer interface {
	Do(r *http.Request) (*http.Response, error)
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
	syncerConfig *clientcmdapi.Config,
	vConfig *config.VirtualClusterConfig,
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
		err = m.loadPlugin(pluginPath, vConfig)
		if err != nil {
			return fmt.Errorf("start plugin %s: %w", pluginPath, err)
		}
	}

	port := 13370
	// after loading all plugins we start them
	for _, p := range m.Plugins {
		// build the start request
		initRequest, err := m.buildInitRequest(filepath.Dir(vClusterPlugin.Path), syncerConfig, vConfig, port)
		port++

		if err != nil {
			return fmt.Errorf("build start request: %w", err)
		}

		// start the plugin
		_, err = p.GRPCClient.Initialize(ctx, initRequest)
		if err != nil {
			return fmt.Errorf("error starting plugin %s: %w", p.Path, err)
		}

		// get plugin config
		pluginConfigResponse, err := p.GRPCClient.GetPluginConfig(ctx, &pluginv2.GetPluginConfig_Request{})
		if err != nil {
			return fmt.Errorf("error retrieving client hooks for plugin %s: %w", p.Path, err)
		}

		// parse plugin config
		pluginConfig, err := parsePluginConfig(pluginConfigResponse.Config)
		if err != nil {
			return fmt.Errorf("error parsing plugin config: %w", err)
		}

		// register client hooks
		err = m.registerClientHooks(p, pluginConfig.ClientHooks)
		if err != nil {
			return fmt.Errorf("error adding client hook for plugin %s: %w", p.Path, err)
		}

		// register Interceptors
		err = m.registerInterceptors(p, pluginConfig.Interceptors)
		if err != nil {
			return fmt.Errorf("error adding interceptor for plugin %s: %w", p.Path, err)
		}

		klog.FromContext(ctx).Info("Successfully loaded plugin", "plugin", p.Path)
	}

	return nil
}

// InterceptorPortForResource returns the port and handler name for the given group, resource and verb
func (m *Manager) InterceptorPortForResource(apigroup, resource, verb string) (ok bool, port int, handlerName string) {
	versionResource := plugintypes.GroupedResource{
		APIGroup: apigroup,
		Resource: resource,
	}
	// resource not registered
	if m.ResourceInterceptorsPorts[versionResource] == nil {
		return false, -1, ""
	}
	// if we have defined the wildcard for verb, return true
	if portAndName, ok := m.ResourceInterceptorsPorts[versionResource]["*"]; ok {
		return true, portAndName.port, portAndName.handlerName
	}
	// return true only if the verb is in the map
	portAndName, ok := m.ResourceInterceptorsPorts[versionResource][verb]
	return ok, portAndName.port, portAndName.handlerName
}

// InterceptorPortForNonResourceURL returns the port and handler name for the given nonResourceUrl and verb
func (m *Manager) InterceptorPortForNonResourceURL(path, verb string) (ok bool, port int, handlerName string) {
	// matchedPath will contain either the original path or the wildcard path that matched
	matchedPath := ""
	if ok, matchedPath = m.urlMatchWithWildcard(path); !ok {
		return false, -1, ""
	}
	// wildcard for verb so return true
	if portAndName, ok := m.NonResourceInterceptorsPorts[matchedPath]["*"]; ok {
		return true, portAndName.port, portAndName.handlerName
	}
	// return true only if the verb is in the map
	portAndName, ok := m.NonResourceInterceptorsPorts[matchedPath][verb]
	return ok, portAndName.port, portAndName.handlerName
}

func (m *Manager) urlMatchWithWildcard(path string) (bool, string) {
	for key := range m.NonResourceInterceptorsPorts {
		// safe because we don't add the empty string in the registration
		// if we have a wildcard, we should return true if the path starts with what's before *
		if key[len(key)-1] == '*' && strings.HasPrefix(path, key[:len(key)-1]) {
			return true, key
		}
		if path == key {
			return true, key
		}
	}

	return false, ""
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

func (m *Manager) registerInterceptors(vClusterPlugin *vClusterPlugin, interceptors InterceptorConfig) error {
	// register the interceptors
	for _, interceptorsInfos := range interceptors.Interceptors {
		if interceptorsInfos.Resources == nil {
			return fmt.Errorf("resources is empty in plugin %s hook", vClusterPlugin.Path)
		}

		// add all group/version/verb tuples to the map
		// each group
		for _, apigroup := range interceptorsInfos.APIGroups {
			// each resource
			for _, resource := range interceptorsInfos.Resources {
				groupedResource := plugintypes.GroupedResource{
					Resource: resource,
					APIGroup: apigroup,
				}
				// each verb
				for _, v := range interceptorsInfos.Verbs {
					// we shouldn't have * and other verbs
					if v == "*" && len(interceptorsInfos.Verbs) > 1 {
						return fmt.Errorf("error while loading the plugins, interceptor for %s/%s defines both * and other verbs. Please either specify * or a list of verbs", apigroup, resource)
					}
					if _, ok := m.ResourceInterceptorsPorts[groupedResource][v]; ok {
						return fmt.Errorf("error while loading the plugins, multiple interceptor plugins are registered for the same resource %s/%s and verb %s", groupedResource.APIGroup, groupedResource.Resource, v)
					}

					m.ResourceInterceptorsPorts[groupedResource][v] = portHandlerName{port: interceptors.Port, handlerName: interceptorsInfos.HandlerName}
				}
			}
		}

		// register nonresourceurls for each verb
		for _, nonResourceURL := range interceptorsInfos.NonResourceURLs {
			// ignore empty resources
			if nonResourceURL == "" {
				continue
			}
			// having a wildcard char not at the end is forbidden
			firstStar := strings.Index(nonResourceURL, "*")
			if firstStar > -1 && firstStar != len(nonResourceURL)-1 {
				return fmt.Errorf("error while loading the plugins, interceptor for non resource url %s defines a wildcard not at the end of the url", nonResourceURL)
			}

			for _, v := range interceptorsInfos.Verbs {
				// we shouldn't have * and other verbs
				if v == "*" && len(interceptorsInfos.Verbs) > 1 {
					return fmt.Errorf("error while loading the plugins, interceptor for non resource url %s defines both * and other verbs. Please either specify * or a list of verbs", nonResourceURL)
				}
				if _, ok := m.NonResourceInterceptorsPorts[nonResourceURL][v]; ok {
					return fmt.Errorf("error while loading the plugins, multiple interceptor plugins are registered for the same non resource url %s and verb %s", nonResourceURL, v)
				}

				m.NonResourceInterceptorsPorts[nonResourceURL][v] = portHandlerName{port: interceptors.Port, handlerName: interceptorsInfos.HandlerName}
			}
		}

		//klog.Infof("Register interceptor for %s %s in plugin %s", interceptorsInfos.APIVersion, interceptorsInfos.Resource, vClusterPlugin.Path)
	}

	return nil
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
	workingDir string,
	syncerConfig *clientcmdapi.Config,
	vConfig *config.VirtualClusterConfig,
	port int,
) (*pluginv2.Initialize_Request, error) {
	// encode config
	encodedConfig, err := json.Marshal(vConfig)
	if err != nil {
		return nil, fmt.Errorf("encode config: %w", err)
	}

	// convert to legacy options
	legacyOptions, err := vConfig.LegacyOptions()
	if err != nil {
		return nil, fmt.Errorf("get legacy options: %w", err)
	}

	// We need this for downward compatibility
	encodedLegacyOptions, err := json.Marshal(legacyOptions)
	if err != nil {
		return nil, fmt.Errorf("marshal options: %w", err)
	}

	// Physical client config
	convertedPhysicalConfig, err := kubeconfig.ConvertRestConfigToClientConfig(vConfig.WorkloadConfig)
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
		Pro: InitConfigPro{
			Enabled:  len(m.ProFeatures) > 0,
			Features: m.ProFeatures,
		},
		PhysicalClusterConfig: phyisicalConfigBytes,
		SyncerConfig:          syncerConfigBytes,
		CurrentNamespace:      vConfig.WorkloadNamespace,
		Config:                encodedConfig,
		Options:               encodedLegacyOptions,
		WorkingDir:            workingDir,
		Port:                  port,
	})
	if err != nil {
		return nil, fmt.Errorf("error encoding init config: %w", err)
	}

	return &pluginv2.Initialize_Request{
		Config: string(initConfig),
	}, nil
}

func (m *Manager) loadPlugin(pluginPath string, vConfig *config.VirtualClusterConfig) error {
	// Create an hclog.Logger
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "plugin",
		Output: os.Stdout,
		Level:  hclog.Info,
	})

	// build command
	cmd, err := buildCommand(pluginPath, vConfig)
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
		SkipHostEnv:      true,
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

func buildCommand(pluginPath string, vConfig *config.VirtualClusterConfig) (*exec.Cmd, error) {
	pluginName := filepath.Base(filepath.Dir(pluginPath))
	pluginConfig := ""

	// legacy plugin
	if vConfig.Plugin != nil {
		legacyPlugin, ok := vConfig.Plugin[pluginName]
		if ok && legacyPlugin.Version == "v2" {
			pluginConfigEncoded, err := yaml.Marshal(legacyPlugin.Config)
			if err != nil {
				return nil, fmt.Errorf("encode plugin config: %w", err)
			}

			pluginConfig = string(pluginConfigEncoded)
		}
	}

	// new plugin
	if vConfig.Plugins != nil {
		newPlugin, ok := vConfig.Plugins[pluginName]
		if ok {
			pluginConfigEncoded, err := yaml.Marshal(newPlugin.Config)
			if err != nil {
				return nil, fmt.Errorf("encode plugin config: %w", err)
			}

			pluginConfig = string(pluginConfigEncoded)
		}
	}

	// check for plugin config
	cmd := exec.Command(pluginPath)
	if pluginConfig == "" {
		cmd.Env = os.Environ()
		return cmd, nil
	}

	// add to plugin environment
	cmd.Env = append(os.Environ(), PluginConfigEnv+"="+pluginConfig)
	return cmd, nil
}

func (m *Manager) WithInterceptors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info, ok := request.RequestInfoFrom(r.Context())
		if !ok {
			klog.V(1).Info("could not determine the infos from the request, serving next handler")
			next.ServeHTTP(w, r)
			return
		}

		port := -1
		handlerName := ""
		// check if this is a match for the non resource url that we registered
		if info.IsResourceRequest {
			ok, port, handlerName = m.InterceptorPortForResource(info.APIGroup, info.Resource, info.Verb)
			if !ok {
				// no interceptor, business as usual
				next.ServeHTTP(w, r)
				return
			}
		} else {
			ok, port, handlerName = m.InterceptorPortForNonResourceURL(r.URL.Path, info.Verb)
			if !ok {
				// no interceptor, business as usual
				next.ServeHTTP(w, r)
				return
			}
		}
		reverseProxy := httputil.ReverseProxy{
			Rewrite: func(r *httputil.ProxyRequest) {
				// adds an extra header so it is simpler within the plugin sdk to
				// determine which handler matched
				r.Out.Header.Add("Vcluster-Plugin-Handler-Name", handlerName)
				r.Out.URL.Host = "localhost:" + strconv.Itoa(port)
				r.Out.URL.Scheme = "http"
			},
		}
		reverseProxy.ServeHTTP(w, r)
	})
}
