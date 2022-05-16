package plugin

import (
	context "context"
	"encoding/json"
	"fmt"
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"go.uber.org/atomic"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	"net"
	"strings"
	"sync"
	"time"

	remote "github.com/loft-sh/vcluster/pkg/plugin/remote"
	grpc "google.golang.org/grpc"
	rest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/pkg/errors"
)

var DefaultManager Manager = &manager{
	clientHooks:    map[VersionKindType][]*Plugin{},
	pluginVersions: map[string]string{},
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
}

var _ remote.PluginInitializerServer = &manager{}

type manager struct {
	remote.UnimplementedPluginInitializerServer

	physicalKubeConfig string
	virtualKubeConfig  string
	syncerKubeConfig   string

	targetNamespace  string
	currentNamespace string

	options string

	isLeader    atomic.Bool
	clientHooks map[VersionKindType][]*Plugin

	pluginMutex    sync.Mutex
	pluginVersions map[string]string
}

type VersionKindType struct {
	ApiVersion string
	Kind       string
	Type       string
}

type Plugin struct {
	Name    string
	Address string
}

func (m *manager) HasClientHooks() bool {
	return len(m.clientHooks) > 0
}

func (m *manager) ClientHooksFor(versionKindType VersionKindType) []*Plugin {
	return m.clientHooks[versionKindType]
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
	remote.RegisterPluginInitializerServer(grpcServer, m)
	go func() {
		err := grpcServer.Serve(lis)
		if err != nil {
			panic(err)
		}
	}()

	return m.registerClientHooks(options)
}

func (m *manager) registerClientHooks(options *context2.VirtualClusterOptions) error {
	now := time.Now()
	for _, pluginAddress := range options.PluginAddresses {
		splitted := strings.Split(pluginAddress, "=")
		if len(splitted) != 2 {
			return fmt.Errorf("error parsing plugin address '%s': expected plugin=address", pluginAddress)
		}

		err := wait.PollImmediate(time.Second, time.Minute*3, func() (done bool, err error) {
			// check if old plugin version
			m.pluginMutex.Lock()
			version, ok := m.pluginVersions[splitted[0]]
			if ok && version == "" {
				m.pluginMutex.Unlock()
				return true, nil
			}
			m.pluginMutex.Unlock()

			// try to reach plugin grpc server
			conn, err := grpc.Dial(splitted[1], grpc.WithInsecure())
			if err != nil {
				if time.Since(now) > time.Second*20 {
					klog.Infof("Error dialing plugin %s: %v", splitted[0], err)
				}
				return false, nil
			}
			defer conn.Close()

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()

			registerResult, err := remote.NewPluginClient(conn).Register(ctx, &remote.RegisterPluginRequest{})
			if err != nil {
				if time.Since(now) > time.Second*20 {
					klog.Infof("Error registering plugin %s: %v", splitted[0], err)
				}
				return false, nil
			}

			// register new client hooks
			plugin := &Plugin{
				Name:    splitted[0],
				Address: splitted[1],
			}
			for _, clientHookInfo := range registerResult.ClientHooks {
				if clientHookInfo.ApiVersion == "" {
					return false, fmt.Errorf("api version is empty in plugin %s hook", plugin.Name)
				} else if clientHookInfo.Kind == "" {
					return false, fmt.Errorf("kind is empty in plugin %s hook", plugin.Name)
				}

				for _, t := range clientHookInfo.Types {
					if t == "" {
						continue
					}

					versionKindType := VersionKindType{
						ApiVersion: clientHookInfo.ApiVersion,
						Kind:       clientHookInfo.Kind,
						Type:       t,
					}
					m.clientHooks[versionKindType] = append(m.clientHooks[versionKindType], plugin)
				}

				klog.Infof("Register client hook for %s %s in plugin %s", clientHookInfo.ApiVersion, clientHookInfo.Kind, plugin.Name)
			}

			return true, nil
		})
		if err != nil {
			return fmt.Errorf("error waiting for plugin %s: %v", splitted[0], err)
		}
	}

	return nil
}

func (m *manager) IsLeader(ctx context.Context, empty *remote.Empty) (*remote.LeaderInfo, error) {
	return &remote.LeaderInfo{
		Leader: m.isLeader.Load(),
	}, nil
}

func (m *manager) Register(ctx context.Context, info *remote.PluginInfo) (*remote.Context, error) {
	if info != nil && info.Name != "" {
		klog.Infof("Registering plugin %s", info.Name)

		m.pluginMutex.Lock()
		m.pluginVersions[info.Name] = info.Version
		m.pluginMutex.Unlock()
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
		o, err := ioutil.ReadFile(kubeConfig.Clusters[contextName].CertificateAuthority)
		if err != nil {
			return nil, err
		}

		kubeConfig.Clusters[contextName].CertificateAuthority = ""
		kubeConfig.Clusters[contextName].CertificateAuthorityData = o
	}

	// fill in data
	if kubeConfig.AuthInfos[contextName].ClientCertificateData == nil && kubeConfig.AuthInfos[contextName].ClientCertificate != "" {
		o, err := ioutil.ReadFile(kubeConfig.AuthInfos[contextName].ClientCertificate)
		if err != nil {
			return nil, err
		}

		kubeConfig.AuthInfos[contextName].ClientCertificate = ""
		kubeConfig.AuthInfos[contextName].ClientCertificateData = o
	}
	if kubeConfig.AuthInfos[contextName].ClientKeyData == nil && kubeConfig.AuthInfos[contextName].ClientKey != "" {
		o, err := ioutil.ReadFile(kubeConfig.AuthInfos[contextName].ClientKey)
		if err != nil {
			return nil, err
		}

		kubeConfig.AuthInfos[contextName].ClientKey = ""
		kubeConfig.AuthInfos[contextName].ClientKeyData = o
	}

	return clientcmd.NewDefaultClientConfig(*kubeConfig, &clientcmd.ConfigOverrides{}), nil
}
