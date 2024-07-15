package setup

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/controllers"
	"github.com/loft-sh/vcluster/pkg/mappings/registermappings"
	"github.com/loft-sh/vcluster/pkg/server"
	syncertypes "github.com/loft-sh/vcluster/pkg/types"
	util "github.com/loft-sh/vcluster/pkg/util/context"
	"k8s.io/klog/v2"
)

func StartManagers(controllerContext *config.ControllerContext) ([]syncertypes.Object, error) {
	// register resource mappings
	err := registermappings.RegisterMappings(util.ToRegisterContext(controllerContext))
	if err != nil {
		return nil, fmt.Errorf("register resource mappings: %w", err)
	}

	// index fields for server
	err = server.RegisterIndices(controllerContext)
	if err != nil {
		return nil, fmt.Errorf("register server indices: %w", err)
	}

	// init syncers
	syncers, err := controllers.CreateSyncers(controllerContext)
	if err != nil {
		return nil, fmt.Errorf("create syncers: %w", err)
	}

	// execute controller initializers to setup prereqs, etc.
	err = controllers.ExecuteInitializers(controllerContext, syncers)
	if err != nil {
		return nil, fmt.Errorf("execute initializers: %w", err)
	}

	// register indices
	err = controllers.RegisterIndices(controllerContext, syncers)
	if err != nil {
		return nil, fmt.Errorf("register indices: %w", err)
	}

	// start the local manager
	go func() {
		err := controllerContext.LocalManager.Start(controllerContext.Context)
		if err != nil {
			panic(err)
		}
	}()

	// start the virtual cluster manager
	go func() {
		err := controllerContext.VirtualManager.Start(controllerContext.Context)
		if err != nil {
			panic(err)
		}
	}()

	// Wait for caches to be synced
	klog.Infof("Starting local & virtual managers...")
	controllerContext.LocalManager.GetCache().WaitForCacheSync(controllerContext.Context)
	controllerContext.VirtualManager.GetCache().WaitForCacheSync(controllerContext.Context)
	klog.Infof("Successfully started local & virtual manager")

	return syncers, nil
}
