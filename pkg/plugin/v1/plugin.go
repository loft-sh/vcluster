package v1

import (
	context "context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/loft-sh/vcluster/config/legacyconfig"
	plugintypes "github.com/loft-sh/vcluster/pkg/plugin/types"
	"github.com/loft-sh/vcluster/pkg/util/kubeconfig"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/random"
	"go.uber.org/atomic"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	remote "github.com/loft-sh/vcluster/pkg/plugin/v1/remote"
	grpc "google.golang.org/grpc"
	rest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/pkg/errors"
)

var runID = random.String(12)

var _ remote.VClusterServer = &Manager{}

func NewManager() *Manager {
	return &Manager{
		clientHooks:    map[plugintypes.VersionKindType][]*Plugin{},
		pluginVersions: map[string]*remote.RegisterPluginRequest{},
	}
}

type Manager struct {
	remote.UnimplementedVClusterServer

	physicalKubeConfig string
	virtualKubeConfig  string
	syncerKubeConfig   string

	targetNamespace  string
	currentNamespace string

	options string

	isLeader   atomic.Bool
	hasPlugins atomic.Bool

	clientHooksMutex sync.Mutex
	clientHooks      map[plugintypes.VersionKindType][]*Plugin

	pluginMutex    sync.Mutex
	pluginVersions map[string]*remote.RegisterPluginRequest
}

type Plugin struct {
	Name    string
	Address string
}

func (m *Manager) HasClientHooks() bool {
	m.clientHooksMutex.Lock()
	defer m.clientHooksMutex.Unlock()

	return len(m.clientHooks) > 0
}

func (m *Manager) ClientHooksFor(versionKindType plugintypes.VersionKindType) []*Plugin {
	m.clientHooksMutex.Lock()
	defer m.clientHooksMutex.Unlock()

	return m.clientHooks[versionKindType]
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
	clientHooks := m.ClientHooksFor(versionKindType)
	if len(clientHooks) == 0 {
		return nil
	}

	encodedObj, err := json.Marshal(obj)
	if err != nil {
		return errors.Wrap(err, "encode obj")
	}

	for _, clientHook := range clientHooks {
		encodedObj, err = m.mutateObject(ctx, versionKindType, encodedObj, clientHook)
		if err != nil {
			return err
		}
	}

	err = json.Unmarshal(encodedObj, obj)
	if err != nil {
		return errors.Wrap(err, "unmarshal obj")
	}

	return nil
}

func (m *Manager) mutateObject(ctx context.Context, versionKindType plugintypes.VersionKindType, obj []byte, plugin *Plugin) ([]byte, error) {
	conn, err := grpc.NewClient(plugin.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("error dialing plugin %s: %w", plugin.Name, err)
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)

	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	loghelper.New("mutate").Debugf("calling plugin %s to mutate object %s %s", plugin.Name, versionKindType.APIVersion, versionKindType.Kind)
	mutateResult, err := remote.NewPluginClient(conn).Mutate(ctx, &remote.MutateRequest{
		ApiVersion: versionKindType.APIVersion,
		Kind:       versionKindType.Kind,
		Object:     string(obj),
		Type:       versionKindType.Type,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "call plugin %s", plugin.Name)
	}

	if mutateResult.Mutated {
		return []byte(mutateResult.Object), nil
	}
	return obj, nil
}

func (m *Manager) HasClientHooksForType(versionKindType plugintypes.VersionKindType) bool {
	return len(m.ClientHooksFor(versionKindType)) > 0
}

func (m *Manager) HasPlugins() bool {
	return m.hasPlugins.Load()
}

func (m *Manager) SetLeader(isLeader bool) {
	m.isLeader.Store(isLeader)
}

func (m *Manager) Start(
	ctx context.Context,
	currentNamespace, targetNamespace string,
	virtualKubeConfig *rest.Config,
	physicalKubeConfig *rest.Config,
	syncerConfig *clientcmdapi.Config,
	options *legacyconfig.LegacyVirtualClusterOptions,
) error {
	// set if we have plugins
	m.hasPlugins.Store(len(options.Plugins) > 0)

	// base options
	m.currentNamespace = currentNamespace
	m.targetNamespace = targetNamespace

	// Context options
	out, err := json.Marshal(options)
	if err != nil {
		return errors.Wrap(err, "marshal options")
	}
	m.options = string(out)

	// Virtual client config
	convertedVirtualConfig, err := kubeconfig.ConvertRestConfigToClientConfig(virtualKubeConfig)
	if err != nil {
		return errors.Wrap(err, "convert virtual client config")
	}
	rawVirtualConfig, err := convertedVirtualConfig.RawConfig()
	if err != nil {
		return errors.Wrap(err, "convert virtual client config")
	}
	virtualConfigBytes, err := clientcmd.Write(rawVirtualConfig)
	if err != nil {
		return errors.Wrap(err, "marshal virtual client config")
	}
	m.virtualKubeConfig = string(virtualConfigBytes)

	// Physical client config
	convertedPhysicalConfig, err := kubeconfig.ConvertRestConfigToClientConfig(physicalKubeConfig)
	if err != nil {
		return errors.Wrap(err, "convert physical client config")
	}
	rawPhysicalConfig, err := convertedPhysicalConfig.RawConfig()
	if err != nil {
		return errors.Wrap(err, "convert physical client config")
	}
	phyisicalConfigBytes, err := clientcmd.Write(rawPhysicalConfig)
	if err != nil {
		return errors.Wrap(err, "marshal physical client config")
	}
	m.physicalKubeConfig = string(phyisicalConfigBytes)

	// Syncer client config
	syncerConfigBytes, err := clientcmd.Write(*syncerConfig)
	if err != nil {
		return errors.Wrap(err, "marshal syncer client config")
	}
	m.syncerKubeConfig = string(syncerConfigBytes)

	// start the grpc server
	loghelper.Infof("Plugin server listening on %s", options.PluginListenAddress)
	lis, err := net.Listen("tcp", options.PluginListenAddress)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	remote.RegisterVClusterServer(grpcServer, m)
	go func() {
		err := grpcServer.Serve(lis)
		if err != nil {
			panic(err)
		}
	}()

	return m.waitForPlugins(ctx, options)
}

func (m *Manager) waitForPlugins(ctx context.Context, options *legacyconfig.LegacyVirtualClusterOptions) error {
	for _, plugin := range options.Plugins {
		klog.Infof("Waiting for plugin %s to register...", plugin)
		err := wait.PollUntilContextTimeout(ctx, time.Millisecond*100, time.Minute*10, true, func(context.Context) (done bool, err error) {
			m.pluginMutex.Lock()
			defer m.pluginMutex.Unlock()

			_, ok := m.pluginVersions[plugin]
			return ok, nil
		})
		if err != nil {
			return fmt.Errorf("error waiting for plugin %s: %w", plugin, err)
		}
		klog.Infof("Plugin %s has successfully registered", plugin)
	}

	return nil
}

func (m *Manager) IsLeader(context.Context, *remote.Empty) (*remote.LeaderInfo, error) {
	return &remote.LeaderInfo{
		Leader: m.isLeader.Load(),
		RunID:  runID,
	}, nil
}

func (m *Manager) GetContext(context.Context, *remote.Empty) (*remote.Context, error) {
	return &remote.Context{
		VirtualClusterConfig:  m.virtualKubeConfig,
		PhysicalClusterConfig: m.physicalKubeConfig,
		SyncerConfig:          m.syncerKubeConfig,
		TargetNamespace:       m.targetNamespace,
		CurrentNamespace:      m.currentNamespace,
		Options:               m.options,
	}, nil
}

func (m *Manager) RegisterPlugin(_ context.Context, info *remote.RegisterPluginRequest) (*remote.RegisterPluginResult, error) {
	if info != nil && info.Name != "" {
		klog.Infof("Registering plugin %s", info.Name)

		// copy map
		m.pluginMutex.Lock()
		defer m.pluginMutex.Unlock()

		m.clientHooksMutex.Lock()
		defer m.clientHooksMutex.Unlock()

		newPlugins := map[string]*remote.RegisterPluginRequest{}
		for k, v := range m.pluginVersions {
			newPlugins[k] = v
		}
		newPlugins[info.Name] = info

		// regenerate client hooks
		newClientHooks, err := regenerateClientHooks(newPlugins)
		if err != nil {
			klog.Infof("Error regenerating client hooks for plugin %s: %v", info.Name, err)
			return nil, errors.Wrap(err, "generate client hooks")
		}

		m.clientHooks = newClientHooks
		m.pluginVersions = newPlugins
	}

	return &remote.RegisterPluginResult{}, nil
}

// Register is deprecated and will be removed in future
func (m *Manager) Register(_ context.Context, info *remote.PluginInfo) (*remote.Context, error) {
	if info != nil && info.Name != "" {
		klog.Infof("Registering plugin %s", info.Name)

		m.pluginMutex.Lock()
		defer m.pluginMutex.Unlock()

		m.pluginVersions[info.Name] = &remote.RegisterPluginRequest{
			Name: info.Name,
		}
	}

	return &remote.Context{
		VirtualClusterConfig:  m.virtualKubeConfig,
		PhysicalClusterConfig: m.physicalKubeConfig,
		SyncerConfig:          m.syncerKubeConfig,
		TargetNamespace:       m.targetNamespace,
		CurrentNamespace:      m.currentNamespace,
		Options:               m.options,
	}, nil
}

func regenerateClientHooks(plugins map[string]*remote.RegisterPluginRequest) (map[plugintypes.VersionKindType][]*Plugin, error) {
	retMap := map[plugintypes.VersionKindType][]*Plugin{}
	for _, pluginInfo := range plugins {
		plugin := &Plugin{
			Name:    pluginInfo.Name,
			Address: pluginInfo.Address,
		}
		for _, clientHookInfo := range pluginInfo.ClientHooks {
			if clientHookInfo.ApiVersion == "" {
				return nil, fmt.Errorf("api version is empty in plugin %s hook", plugin.Name)
			} else if clientHookInfo.Kind == "" {
				return nil, fmt.Errorf("kind is empty in plugin %s hook", plugin.Name)
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
				retMap[versionKindType] = append(retMap[versionKindType], plugin)
			}

			klog.Infof("Register client hook for %s %s in plugin %s", clientHookInfo.ApiVersion, clientHookInfo.Kind, plugin.Name)
		}
	}

	return retMap, nil
}
