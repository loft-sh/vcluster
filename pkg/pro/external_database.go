package pro

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/etcd"
)

var ConfigureExternalDatabase = func(_ context.Context, _ string, _ *config.VirtualClusterConfig, _ bool) (string, *etcd.Certificates, error) {
	return "", nil, NewFeatureError("external database")
}
