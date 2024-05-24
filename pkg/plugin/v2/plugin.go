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
	"slices"
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

	return &Manager{
		PluginFolder:                 pluginFolder,
		ClientHooks:                  map[plugintypes.VersionKindType][]*vClusterPlugin{},
		ResourceInterceptorsPorts:    map[string]map[string]map[string]map[string]portHandlerName{},
		NonResourceInterceptorsPorts: map[string]map[string]portHandlerName{},
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
	// structure is group>resource>verb>resourceName
	ResourceInterceptorsPorts map[string]map[string]map[string]map[string]portHandlerName
	// map to track the port that needs to be targeted for the non resource interceptors
	NonResourceInterceptorsPorts map[string]map[string]portHandlerName
	// ProFeatures are pro features to hand-over to the plugin
	ProFeatures map[string]bool
}

type portHandlerName struct {
	handlerName string
	port        int
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
	for _, vClusterPlugin := range m.Plugins {
		// build the start request
		initRequest, err := m.buildInitRequest(filepath.Dir(vClusterPlugin.Path), syncerConfig, vConfig, port)

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

		// register Interceptors
		err = m.registerInterceptors(pluginConfig.Interceptors, port)
		if err != nil {
			return fmt.Errorf("error adding interceptor for plugin %s: %w", vClusterPlugin.Path, err)
		}

		klog.FromContext(ctx).Info("Successfully loaded plugin", "plugin", vClusterPlugin.Path)

		port++
	}

	return nil
}

// interceptorPortForResource returns the port and handler name for the given group, resource and verb
func (m *Manager) interceptorPortForResource(group, resource, verb, resourceName string) (bool, int, string) {
	groups := m.ResourceInterceptorsPorts
	if resourcesMap, ok := groups[group]; ok {
		portHandlerName, ok := portForResource(resourcesMap, resource, verb, resourceName)
		if ok {
			return true, portHandlerName.port, portHandlerName.handlerName
		}
	}
	if resourcesMap, ok := groups["*"]; ok {
		portHandlerName, ok := portForResource(resourcesMap, resource, verb, resourceName)
		if ok {
			return true, portHandlerName.port, portHandlerName.handlerName
		}
	}

	return false, 0, ""
}

func portForResource(resources map[string]map[string]map[string]portHandlerName, resource, verb, resourceName string) (portHandlerName, bool) {
	if verbsMap, ok := resources[resource]; ok {
		portHandlerName, ok := portForResourceNameAndVerb(verbsMap, verb, resourceName)
		if ok {
			return portHandlerName, true
		}
	}
	if verbsMap, ok := resources["*"]; ok {
		portHandlerName, ok := portForResourceNameAndVerb(verbsMap, verb, resourceName)
		if ok {
			return portHandlerName, true
		}
	}

	return portHandlerName{}, false
}

func portForResourceNameAndVerb(verbs map[string]map[string]portHandlerName, verb, resourceName string) (portHandlerName, bool) {
	if resourcesNamesMap, ok := verbs[verb]; ok {
		portHandlerName, ok := portForResourceName(resourcesNamesMap, resourceName)
		if ok {
			return portHandlerName, true
		}
	}
	if resourcesNamesMap, ok := verbs["*"]; ok {
		portHandlerName, ok := portForResourceName(resourcesNamesMap, "*")
		if ok {
			return portHandlerName, true
		}
	}

	return portHandlerName{}, false
}

func portForResourceName(resourceNames map[string]portHandlerName, resourceName string) (portHandlerName, bool) {
	if portHandler, ok := resourceNames[resourceName]; ok {
		return portHandler, true
	}
	if portHandler, ok := resourceNames["*"]; ok {
		return portHandler, true
	}
	return portHandlerName{}, false
}

// InterceptorPortForNonResourceURL returns the port and handler name for the given nonResourceUrl and verb
func (m *Manager) InterceptorPortForNonResourceURL(path, verb string) (bool, int, string) {
	// matchedPath will contain either the original path or the wildcard path that matched
	matchedPath := ""
	ok := false
	if ok, matchedPath = m.urlMatchWithWildcard(path); !ok {
		return false, 0, ""
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
		// just the wildcar isn't valid
		if path == "*" {
			return false, ""
		}
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

func validateInterceptor(interceptor InterceptorRule, name string) error {
	if len(interceptor.Verbs) == 0 {
		return fmt.Errorf("verb is empty in interceptor plugin %s  ", name)
	}
	// check for wildcards and extra names, which should not be allowed
	if slices.Contains(interceptor.Resources, "*") && len(interceptor.Resources) > 1 {
		return fmt.Errorf("error while loading the plugins, interceptor for handler %s defines both * and other resources, or is empty. please either specify * or a list of resource", name)
	}
	if slices.Contains(interceptor.APIGroups, "*") && len(interceptor.APIGroups) > 1 {
		return fmt.Errorf("error while loading the plugins, interceptor for handler %s defines both * and other apigroups, or is empty. please either specify * or a list of apigroups", name)
	}

	// make sure that if we don't have any nonresourceurl we at least have some group + resource
	if len(interceptor.NonResourceURLs) == 0 {
		// check for wildcards and extra names, which should not be allowed
		if len(interceptor.Resources) == 0 {
			return fmt.Errorf("error while loading the plugins, interceptor for handler %s defines both * and other resources, or is empty. please either specify * or a list of resource", name)
		}
		if len(interceptor.APIGroups) == 0 {
			return fmt.Errorf("error while loading the plugins, interceptor for handler %s defines both * and other apigroups, or is empty. please either specify * or a list of apigroups", name)
		}
	}

	if (slices.Contains(interceptor.Verbs, "*") && len(interceptor.Verbs) > 1) ||
		len(interceptor.Verbs) == 0 {
		return fmt.Errorf("error while loading the plugins, interceptor for handler %s defines both * and other verbs, or is empty. please either specify * or a list of verb", name)
	}
	// having a wildcard char not at the end is forbidden
	for _, nonResourceURL := range interceptor.NonResourceURLs {
		firstStar := strings.Index(nonResourceURL, "*")
		if firstStar > -1 && firstStar != len(nonResourceURL)-1 {
			return fmt.Errorf("error while loading the plugins, interceptor for non resource url %s defines a wildcard not at the end of the url, or is only a wildcard", nonResourceURL)
		}
	}

	return nil
}

func (m *Manager) registerInterceptors(interceptors map[string][]InterceptorRule, port int) error {
	// register the interceptors
	for name, interceptorRules := range interceptors {
		// make sure that it is valid
		for _, rule := range interceptorRules {
			if err := validateInterceptor(rule, name); err != nil {
				return err
			}

			// register resource interceptors for each verb
			err := m.registerResourceInterceptor(port, rule, name)
			if err != nil {
				return err
			}

			// register nonresourceurls interceptors for each verb
			err = m.registerNonResourceURL(port, rule, name)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *Manager) registerResourceInterceptor(port int, interceptorsInfos InterceptorRule, interceptorName string) error {
	// add all group/version/verb/resourceName tuples to the map
	// each group
	if m.hasConflictWithExistingWildcard(interceptorsInfos.APIGroups, interceptorsInfos.Resources, interceptorsInfos.Verbs, interceptorsInfos.ResourceNames) {
		return fmt.Errorf("error while loading the plugins, there are conflicts with the wildcards")
	}
	for _, apigroup := range interceptorsInfos.APIGroups {
		// create the map if not existing
		if _, ok := m.ResourceInterceptorsPorts[apigroup]; !ok {
			m.ResourceInterceptorsPorts[apigroup] = make(map[string]map[string]map[string]portHandlerName)
		}

		// for each resource
		for _, resource := range interceptorsInfos.Resources {
			// each verb
			if m.ResourceInterceptorsPorts[apigroup][resource] == nil {
				m.ResourceInterceptorsPorts[apigroup][resource] = make(map[string]map[string]portHandlerName)
			}
			for _, verb := range interceptorsInfos.Verbs {
				if m.ResourceInterceptorsPorts[apigroup][resource][verb] == nil {
					m.ResourceInterceptorsPorts[apigroup][resource][verb] = make(map[string]portHandlerName)
				} else {
					// we can't add empty resources if there's already a map since it is
					// the equivalent of *
					if len(interceptorsInfos.ResourceNames) == 0 {
						return fmt.Errorf("error while loading the plugins, multiple interceptor plugins are registered for the same resource %s/%s verb %s and resource name", apigroup, resource, verb)
					}
					// check for the exact same api group
					for resourceName := range m.ResourceInterceptorsPorts[apigroup][resource][verb] {
						if slices.Contains(interceptorsInfos.ResourceNames, resourceName) {
							return fmt.Errorf("error while loading the plugins, multiple interceptor plugins are registered for the same resource %s/%s , verb %s and resource name %s", apigroup, resource, verb, resourceName)
						}
					}
				}
				// now add the specific resource names
				if len(interceptorsInfos.ResourceNames) == 0 {
					// empty slice means everything is allowed
					m.ResourceInterceptorsPorts[apigroup][resource][verb]["*"] = portHandlerName{handlerName: interceptorName, port: port}
				} else {
					for _, name := range interceptorsInfos.ResourceNames {
						m.ResourceInterceptorsPorts[apigroup][resource][verb][name] = portHandlerName{handlerName: interceptorName, port: port}
					}
				}
			}
		}
	}
	return nil
}

func (m *Manager) hasConflictWithExistingWildcard(apigroups, resources, verbs, resourceNames []string) bool {
	if len(resourceNames) == 0 {
		resourceNames = []string{"*"}
	}
	// check if any existing object has wildcards which would conflict
	for _, resourceName := range resourceNames {
		for _, verb := range verbs {
			for _, resource := range resources {
				for _, apiGroup := range apigroups {
					// check for conflict with existing wildcards
					var apiGroupsMap map[string]map[string]map[string]portHandlerName
					if _, ok := m.ResourceInterceptorsPorts["*"]; ok {
						// match on the wildcard
						apiGroupsMap = m.ResourceInterceptorsPorts["*"]
					} else if _, ok := m.ResourceInterceptorsPorts[apiGroup]; ok {
						// match on the resource
						apiGroupsMap = m.ResourceInterceptorsPorts[apiGroup]
					} else {
						// no potential match
						continue
					}
					var resourcesMap map[string]map[string]portHandlerName
					if _, ok := apiGroupsMap["*"]; ok {
						// match on the wildcard
						resourcesMap = apiGroupsMap["*"]
					} else if _, ok := apiGroupsMap[resource]; ok {
						// match on the resource
						resourcesMap = apiGroupsMap[resource]
					} else {
						// no potential match
						continue
					}
					var verbMap map[string]portHandlerName
					if _, ok := resourcesMap["*"]; ok {
						// match on the wildcard
						verbMap = resourcesMap["*"]
					} else if _, ok := resourcesMap[verb]; ok {
						// match on the resource
						verbMap = resourcesMap[verb]
					} else {
						// no potential match
						continue
					}
					if _, ok := verbMap[resourceName]; ok || resourceName == "*" {
						return true
					}
				}
			}
		}
	}

	// check with the new object being added
	for _, resourceName := range resourceNames {
		for _, verb := range verbs {
			for _, resource := range resources {
				for _, apiGroup := range apigroups {
					if hasGroupConflict(m.ResourceInterceptorsPorts, apiGroup, resource, verb, resourceName) {
						return true
					}
				}
			}
		}
	}

	return false
}

func hasGroupConflict(existing map[string]map[string]map[string]map[string]portHandlerName, group, resource, verb, resourceName string) bool {
	if group == "*" {
		for _, v := range existing {
			hasConflict := hasResourceConflict(v, resource, verb, resourceName)
			if hasConflict {
				return true
			}
		}
	} else if resources, ok := existing[group]; ok {
		return hasResourceConflict(resources, resource, verb, resourceName)
	}

	return false
}

func hasResourceConflict(existing map[string]map[string]map[string]portHandlerName, resource, verb, resourceName string) bool {
	if resource == "*" {
		for _, v := range existing {
			hasConflict := hasVerbConflict(v, verb, resourceName)
			if hasConflict {
				return true
			}
		}
	} else if verbs, ok := existing[resource]; ok {
		return hasVerbConflict(verbs, verb, resourceName)
	}

	return false
}

func hasVerbConflict(existing map[string]map[string]portHandlerName, verb, resourceName string) bool {
	if verb == "*" {
		for _, v := range existing {
			hasConflict := hasResourceNameConflit(v, resourceName)
			if hasConflict {
				return true
			}
		}
	} else if resourcesNames, ok := existing[verb]; ok {
		return hasResourceNameConflit(resourcesNames, resourceName)
	}

	return false
}

func hasResourceNameConflit(existing map[string]portHandlerName, resourceName string) bool {
	if resourceName == "*" {
		return true
	}
	_, ok := existing[resourceName]
	return ok
}

func (m *Manager) registerNonResourceURL(port int, interceptorsInfos InterceptorRule, interceptorName string) error {
	// register nonresourceurls for each verb
	for _, nonResourceURL := range interceptorsInfos.NonResourceURLs {
		// ignore empty resources
		if nonResourceURL == "" {
			continue
		}
		for _, v := range interceptorsInfos.Verbs {
			if _, ok := m.NonResourceInterceptorsPorts[nonResourceURL][v]; ok {
				return fmt.Errorf("error while loading the plugins, multiple interceptor plugins are registered for the same non resource url %s and verb %s", nonResourceURL, v)
			}

			m.NonResourceInterceptorsPorts[nonResourceURL][v] = portHandlerName{port: port, handlerName: interceptorName}
		}
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
	workloadConfigBytes, err := clientcmd.Write(rawPhysicalConfig)
	if err != nil {
		return nil, fmt.Errorf("marshal physical client config: %w", err)
	}
	convertedControlPlaneConfig, err := kubeconfig.ConvertRestConfigToClientConfig(vConfig.ControlPlaneConfig)
	if err != nil {
		return nil, fmt.Errorf("convert control plane client config: %w", err)
	}
	rawControlPlaneConfig, err := convertedControlPlaneConfig.RawConfig()
	if err != nil {
		return nil, fmt.Errorf("convert control plane client config: %w", err)
	}
	controlPlaneConfigBytes, err := clientcmd.Write(rawControlPlaneConfig)
	if err != nil {
		return nil, fmt.Errorf("marshal control plane client config: %w", err)
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

		PhysicalClusterConfig: workloadConfigBytes,

		WorkloadConfig:     workloadConfigBytes,
		ControlPlaneConfig: controlPlaneConfigBytes,

		SyncerConfig:     syncerConfigBytes,
		CurrentNamespace: vConfig.WorkloadNamespace,
		Config:           encodedConfig,
		Options:          encodedLegacyOptions,
		WorkingDir:       workingDir,

		Port: port,
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
			klog.V(1).Infof("could not determine the infos from the request %s, serving next handler", r.URL.Path)
			next.ServeHTTP(w, r)
			return
		}

		port := -1
		handlerName := ""
		// check if this is a match for the non resource url that we registered
		if info.IsResourceRequest {
			ok, port, handlerName = m.interceptorPortForResource(info.APIGroup, info.Resource, info.Verb, info.Name)
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
