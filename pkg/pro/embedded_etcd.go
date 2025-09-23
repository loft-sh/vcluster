package pro

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/config"
)

var StartEmbeddedEtcd = func(_ context.Context, _ *config.VirtualClusterConfig, _ string, _ string, _ bool, _ bool) (func(), error) {
	return nil, NewFeatureError("embedded etcd")
}
