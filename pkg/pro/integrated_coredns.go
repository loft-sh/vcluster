package pro

import (
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/specialservices"
)

var StartIntegratedCoreDNS = func(_ *config.ControllerContext) error {
	return NewFeatureError("integrated core dns")
}

var InitDNSServiceSyncing = func(_ *config.VirtualClusterConfig) specialservices.Interface {
	return specialservices.NewDefaultServiceSyncer()
}
