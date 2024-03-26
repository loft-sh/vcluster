package pro

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/etcd"
)

func ConfigureExternalDatabase(_ context.Context, _ *config.VirtualClusterConfig) (string, *etcd.Certificates, error) {
	return "", nil, NewFeatureError("external database")
}
