package nodes

import (
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	syncer "github.com/loft-sh/vcluster/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	uncachedVirtualClient, err := client.New(ctx.VirtualManager.GetConfig(), client.Options{
		Scheme: ctx.VirtualManager.GetScheme(),
		Mapper: ctx.VirtualManager.GetRESTMapper(),
	})
	if err != nil {
		return nil, err
	}

	nodeService := nodeservice.NewNodeServiceProvider(ctx.Config.WorkloadService, ctx.CurrentNamespace, ctx.CurrentNamespaceClient, ctx.VirtualManager.GetClient(), uncachedVirtualClient)
	if !ctx.Config.Sync.FromHost.Nodes.Enabled {
		return NewFakeSyncer(ctx, nodeService)
	}

	return NewSyncer(ctx, nodeService)
}
