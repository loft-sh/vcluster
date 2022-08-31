package plugin

import (
	context "context"
	"encoding/json"
	"fmt"
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/random"
	"go.uber.org/atomic"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	"net"
	"os"
	"sync"
	"time"

	remote "github.com/loft-sh/vcluster/pkg/plugin/remote"
	grpc "google.golang.org/grpc"
	rest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/pkg/errors"
)

var runID = random.RandomString(12)

var DefaultManager Manager = &manager{
	clientHooks:    map[VersionKindType][]*Plugin{},
	pluginVersions: map[string]*remote.RegisterPluginRequest{},
}

type Manager interface {
	Start(
		currentNamespace, targetNamespace string,
		virtualKubeConfig *rest.Config,
		physicalKubeConfig *rest.Config,
		syncerConfig *clientcmdapi.Config,
		options *context2.VirtualClusterOptions,
	) error
	SetLeader(isLeader bool)
	ClientHooksFor(versionKindType VersionKindType) []*Plugin
	HasClientHooks() bool
	HasPlugins() bool
}

var _ remote.VClusterServer = &manager{}

type manager struct {
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
	clientHooks      map[VersionKindType][]*Plugin

	pluginMutex    sync.Mutex
	pluginVersions map[string]*remote.RegisterPluginRequest
}

type VersionKindType struct {
	APIVersion string
	Kind       string
	Type       string
}

type Plugin struct {
	Name    string
	Address string
}

func (m *manager) HasClientHooks() bool {
	m.clientHooksMutex.Lock()
	defer m.clientHooksMutex.Unlock()

	return len(m.clientHooks) > 0
}

func (m *manager) ClientHooksFor(versionKindType VersionKindType) []*Plugin {
	m.clientHooksMutex.Lock()
	defer m.clientHooksMutex.Unlock()

	return m.clientHooks[versionKindType]
}

func (m *manager) HasPlugins() bool {
	return m.hasPlugins.Load()
}

func (m *manager) SetLeader(isLeader bool) {
	m.isLeader.Store(isLeader)
}

func (m *manager) Start(
	currentNamespace, targetNamespace string,
	virtualKubeConfig *rest.Config,
	physicalKubeConfig *rest.Config,
	syncerConfig *clientcmdapi.Config,
	options *context2.VirtualClusterOptions,
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
	convertedVirtualConfig, err := ConvertRestConfigToClientConfig(virtualKubeConfig)
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
	convertedPhysicalConfig, err := ConvertRestConfigToClientConfig(physicalKubeConfig)
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
		return fmt.Errorf("failed to listen: %v", err)
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

	return m.waitForPlugins(options)
}

func (m *manager) waitForPlugins(options *context2.VirtualClusterOptions) error {
	for _, plugin := range options.Plugins {
		klog.Infof("Waiting for plugin %s to register...", plugin)
		err := wait.PollImmediate(time.Millisecond*100, time.Minute*10, func() (done bool, err error) {
			m.pluginMutex.Lock()
			defer m.pluginMutex.Unlock()

			_, ok := m.pluginVersions[plugin]
			return ok, nil
		})
		if err != nil {
			return fmt.Errorf("error waiting for plugin %s: %v", plugin, err)
		}
		klog.Infof("Plugin %s has successfully registered", plugin)
	}

	return nil
}

func (m *manager) IsLeader(ctx context.Context, empty *remote.Empty) (*remote.LeaderInfo, error) {
	return &remote.LeaderInfo{
		Leader: m.isLeader.Load(),
		RunID:  runID,
	}, nil
}

func (m *manager) GetContext(ctx context.Context, empty *remote.Empty) (*remote.Context, error) {
	return &remote.Context{
		VirtualClusterConfig:  m.virtualKubeConfig,
		PhysicalClusterConfig: m.physicalKubeConfig,
		SyncerConfig:          m.syncerKubeConfig,
		TargetNamespace:       m.targetNamespace,
		CurrentNamespace:      m.currentNamespace,
		Options:               m.options,
	}, nil
}

func (m *manager) RegisterPlugin(ctx context.Context, info *remote.RegisterPluginRequest) (*remote.RegisterPluginResult, error) {
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
func (m *manager) Register(ctx context.Context, info *remote.PluginInfo) (*remote.Context, error) {
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

func regenerateClientHooks(plugins map[string]*remote.RegisterPluginRequest) (map[VersionKindType][]*Plugin, error) {
	retMap := map[VersionKindType][]*Plugin{}
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

				versionKindType := VersionKindType{
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

func ConvertRestConfigToClientConfig(config *rest.Config) (clientcmd.ClientConfig, error) {
	contextName := "local"
	kubeConfig := clientcmdapi.NewConfig()
	kubeConfig.Contexts = map[string]*clientcmdapi.Context{
		contextName: {
			Cluster:  contextName,
			AuthInfo: contextName,
		},
	}
	kubeConfig.Clusters = map[string]*clientcmdapi.Cluster{
		contextName: {
			Server:                   config.Host,
			InsecureSkipTLSVerify:    config.Insecure,
			CertificateAuthorityData: config.CAData,
			CertificateAuthority:     config.CAFile,
		},
	}
	kubeConfig.AuthInfos = map[string]*clientcmdapi.AuthInfo{
		contextName: {
			Token:                 config.BearerToken,
			TokenFile:             config.BearerTokenFile,
			Impersonate:           config.Impersonate.UserName,
			ImpersonateGroups:     config.Impersonate.Groups,
			ImpersonateUserExtra:  config.Impersonate.Extra,
			ClientCertificate:     config.CertFile,
			ClientCertificateData: config.CertData,
			ClientKey:             config.KeyFile,
			ClientKeyData:         config.KeyData,
			Username:              config.Username,
			Password:              config.Password,
			AuthProvider:          config.AuthProvider,
			Exec:                  config.ExecProvider,
		},
	}
	kubeConfig.CurrentContext = contextName

	// resolve certificate
	if kubeConfig.Clusters[contextName].CertificateAuthorityData == nil && kubeConfig.Clusters[contextName].CertificateAuthority != "" {
		o, err := os.ReadFile(kubeConfig.Clusters[contextName].CertificateAuthority)
		if err != nil {
			return nil, err
		}

		kubeConfig.Clusters[contextName].CertificateAuthority = ""
		kubeConfig.Clusters[contextName].CertificateAuthorityData = o
	}

	// fill in data
	if kubeConfig.AuthInfos[contextName].ClientCertificateData == nil && kubeConfig.AuthInfos[contextName].ClientCertificate != "" {
		o, err := os.ReadFile(kubeConfig.AuthInfos[contextName].ClientCertificate)
		if err != nil {
			return nil, err
		}

		kubeConfig.AuthInfos[contextName].ClientCertificate = ""
		kubeConfig.AuthInfos[contextName].ClientCertificateData = o
	}
	if kubeConfig.AuthInfos[contextName].ClientKeyData == nil && kubeConfig.AuthInfos[contextName].ClientKey != "" {
		o, err := os.ReadFile(kubeConfig.AuthInfos[contextName].ClientKey)
		if err != nil {
			return nil, err
		}

		kubeConfig.AuthInfos[contextName].ClientKey = ""
		kubeConfig.AuthInfos[contextName].ClientKeyData = o
	}

	return clientcmd.NewDefaultClientConfig(*kubeConfig, &clientcmd.ConfigOverrides{}), nil
}
