package pro

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var ConnectToPlatform = func(context.Context, *config.VirtualClusterConfig, manager.Manager) error {
	return nil
}
