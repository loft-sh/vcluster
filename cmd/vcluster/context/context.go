package context

import (
	"context"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	"github.com/loft-sh/vcluster/pkg/util/blockingcacheclient"
	"github.com/loft-sh/vcluster/pkg/util/locks"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
)

// VirtualClusterOptions holds the cmd flags
type VirtualClusterOptions struct {
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

	Name             string
	DeprecatedSuffix string

	DisableSyncResources string
	TargetNamespace      string
	ServiceName          string

	DeprecatedOwningStatefulSet string
	SetOwner                    bool

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

	LeaderElect   bool
	LeaseDuration int64
	RenewDeadline int64
	RetryPeriod   int64
}

type ControllerContext struct {
	Context context.Context

	LocalManager   ctrl.Manager
	VirtualManager ctrl.Manager

	CurrentNamespace       string
	CurrentNamespaceClient client.Client
	NodeServiceProvider    nodeservice.NodeServiceProvider

	CacheSynced func()
	LockFactory locks.LockFactory
	Options     *VirtualClusterOptions
	StopChan    <-chan struct{}
}

func NewControllerContext(currentNamespace string, localManager ctrl.Manager, virtualManager ctrl.Manager, options *VirtualClusterOptions) (*ControllerContext, error) {
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

	// create a new current namespace client
	currentNamespaceClient, err := newCurrentNamespaceClient(ctx, currentNamespace, localManager, options)
	if err != nil {
		return nil, err
	}

	return &ControllerContext{
		Context:        ctx,
		LocalManager:   localManager,
		VirtualManager: virtualManager,

		CurrentNamespace:       currentNamespace,
		CurrentNamespaceClient: currentNamespaceClient,

		NodeServiceProvider: nodeservice.NewNodeServiceProvider(currentNamespace, currentNamespaceClient, virtualManager.GetClient(), uncachedVirtualClient),
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

func newCurrentNamespaceClient(ctx context.Context, currentNamespace string, localManager ctrl.Manager, options *VirtualClusterOptions) (client.Client, error) {
	var err error

	currentNamespaceCache := localManager.GetCache()
	if currentNamespace != options.TargetNamespace {
		currentNamespaceCache, err = cache.New(localManager.GetConfig(), cache.Options{
			Scheme:    localManager.GetScheme(),
			Mapper:    localManager.GetRESTMapper(),
			Namespace: currentNamespace,
		})
		if err != nil {
			return nil, err
		}
	}

	// index node services by their cluster ip
	err = currentNamespaceCache.IndexField(ctx, &corev1.Service{}, constants.IndexByClusterIP, func(object client.Object) []string {
		svc := object.(*corev1.Service)
		if len(svc.Labels) == 0 || svc.Labels[nodeservice.ServiceClusterLabel] != translate.Suffix {
			return nil
		}

		return []string{svc.Spec.ClusterIP}
	})
	if err != nil {
		return nil, err
	}

	// start cache now if it's not in the same namespace
	if currentNamespace != options.TargetNamespace {
		go func() {
			err := currentNamespaceCache.Start(ctx)
			if err != nil {
				panic(err)
			}
		}()
		currentNamespaceCache.WaitForCacheSync(ctx)
	}

	// create a current namespace client
	currentNamespaceClient, err := blockingcacheclient.NewCacheClient(currentNamespaceCache, localManager.GetConfig(), client.Options{
		Scheme: localManager.GetScheme(),
		Mapper: localManager.GetRESTMapper(),
	})
	if err != nil {
		return nil, err
	}

	return currentNamespaceClient, nil
}
