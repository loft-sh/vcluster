package context

import (
	"context"
	"github.com/loft-sh/vcluster/pkg/util/locks"
	ctrl "sigs.k8s.io/controller-runtime"
	"sync"
)

// VirtualCluster holds the cmd flags
type VirtualClusterOptions struct {
	ServerCaCert        string
	ServerCaKey         string
	TlsSANs             []string
	RequestHeaderCaCert string
	ClientCaCert        string
	KubeConfig          string
	KubeConfigSecret    string

	BindAddress string
	Port        int

	Suffix               string
	DisableSyncResources string
	TargetNamespace      string
	ServiceName          string
	OwningStatefulSet    string

	SyncAllNodes             bool
	UseFakeNodes             bool
	UseFakePersistentVolumes bool
	EnableStorageClasses     bool

	TranslateImages []string

	NodeSelector        string
	ServiceAccount      string
	EnforceNodeSelector bool

	OverrideHosts               bool
	OverrideHostsContainerImage string

	ClusterDomain string
}

type ControllerContext struct {
	Context context.Context

	LocalManager   ctrl.Manager
	VirtualManager ctrl.Manager

	CacheSynced func()
	LockFactory locks.LockFactory
	Options     *VirtualClusterOptions
	StopChan    <-chan struct{}
}

func NewControllerContext(localManager ctrl.Manager, virtualManager ctrl.Manager, options *VirtualClusterOptions) *ControllerContext {
	stopChan := make(<-chan struct{})
	cacheSynced := sync.Once{}
	ctx := context.Background()
	return &ControllerContext{
		Context:        ctx,
		LocalManager:   localManager,
		VirtualManager: virtualManager,
		LockFactory:    locks.NewDefaultLockFactory(),
		CacheSynced: func() {
			cacheSynced.Do(func() {
				localManager.GetCache().WaitForCacheSync(ctx)
				virtualManager.GetCache().WaitForCacheSync(ctx)
			})
		},
		StopChan: stopChan,
		Options:  options,
	}
}
