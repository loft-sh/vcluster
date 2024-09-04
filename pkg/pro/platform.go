package pro

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var ConnectToPlatform = func(context.Context, *config.VirtualClusterConfig) (func(mgr manager.Manager) error, error) {
	return func(_ manager.Manager) error { return nil }, nil
}
