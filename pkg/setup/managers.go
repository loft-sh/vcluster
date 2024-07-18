package setup

import (
	"fmt"

	syncerresources "github.com/loft-sh/vcluster/pkg/controllers/resources"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	mapperresources "github.com/loft-sh/vcluster/pkg/mappings/resources"
	"github.com/loft-sh/vcluster/pkg/server"
	syncertypes "github.com/loft-sh/vcluster/pkg/types"
	"k8s.io/klog/v2"
)

func StartManagers(ctx *synccontext.RegisterContext) ([]syncertypes.Object, error) {
	// index fields for server
	err := server.RegisterIndices(ctx)
	if err != nil {
		return nil, fmt.Errorf("register server indices: %w", err)
	}

	// register resource mappings
	err = mapperresources.RegisterMappings(ctx)
	if err != nil {
		return nil, fmt.Errorf("register resource mappings: %w", err)
	}

	// init syncers before starting the managers as they might need to register indices
	syncers, err := syncerresources.BuildSyncers(ctx)
	if err != nil {
		return nil, fmt.Errorf("create syncers: %w", err)
	}

	// start the local manager
	go func() {
		err := ctx.PhysicalManager.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()

	// start the virtual cluster manager
	go func() {
		err := ctx.VirtualManager.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()

	// Wait for caches to be synced
	klog.FromContext(ctx).Info("Starting local & virtual managers...")
	ctx.PhysicalManager.GetCache().WaitForCacheSync(ctx)
	ctx.VirtualManager.GetCache().WaitForCacheSync(ctx)
	klog.FromContext(ctx).Info("Successfully started local & virtual manager")

	return syncers, nil
}
