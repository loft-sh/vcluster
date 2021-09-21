package context

import (
	"context"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	"github.com/loft-sh/vcluster/pkg/util/locks"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
)

// VirtualClusterOptions holds the cmd flags
type VirtualClusterOptions struct {
	ServiceAccountKey   string
	ServerCaCert        string
	ServerCaKey         string
	TlsSANs             []string
	RequestHeaderCaCert string
	ClientCaCert        string
	KubeConfig          string

	KubeConfigSecret          string
	KubeConfigSecretNamespace string
	KubeConfigServer          string

	BindAddress string
	Port        int

	Suffix               string
	DisableSyncResources string
	TargetNamespace      string
	ServiceName          string
	ServiceNamespace     string
	OwningStatefulSet    string

	SyncAllNodes             bool
	SyncNodeChanges          bool
	UseFakeKubelets          bool
	UseFakeNodes             bool
	UseFakePersistentVolumes bool
	EnableStorageClasses     bool
	EnablePriorityClasses    bool

	TranslateImages []string

	NodeSelector        string
	ServiceAccount      string
	EnforceNodeSelector bool

	OverrideHosts               bool
	OverrideHostsContainerImage string

	ClusterDomain string

	LeaseDuration int64
	RenewDeadline int64
	RetryPeriod   int64
}

type ControllerContext struct {
	Context context.Context

	LocalManager   ctrl.Manager
	VirtualManager ctrl.Manager

	NodeServiceProvider nodeservice.NodeServiceProvider

	CacheSynced func()
	LockFactory locks.LockFactory
	Options     *VirtualClusterOptions
	StopChan    <-chan struct{}
}

func NewControllerContext(localManager ctrl.Manager, virtualManager ctrl.Manager, options *VirtualClusterOptions) (*ControllerContext, error) {
	stopChan := make(<-chan struct{})
	cacheSynced := sync.Once{}
	ctx := context.Background()
	uncachedVirtualClient, err := client.New(virtualManager.GetConfig(), client.Options{
		Scheme: virtualManager.GetScheme(),
		Mapper: virtualManager.GetRESTMapper(),
	})
	if err != nil {
		return nil, err
	}
	return &ControllerContext{
		Context:             ctx,
		LocalManager:        localManager,
		VirtualManager:      virtualManager,
		NodeServiceProvider: nodeservice.NewNodeServiceProvider(localManager.GetClient(), virtualManager.GetClient(), uncachedVirtualClient, options.TargetNamespace),
		LockFactory:         locks.NewDefaultLockFactory(),
		CacheSynced: func() {
			cacheSynced.Do(func() {
				localManager.GetCache().WaitForCacheSync(ctx)
				virtualManager.GetCache().WaitForCacheSync(ctx)
			})
		},
		StopChan: stopChan,
		Options:  options,
	}, nil
}
