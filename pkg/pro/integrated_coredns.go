package pro

import (
	"github.com/loft-sh/vcluster/pkg/options"
	"github.com/loft-sh/vcluster/pkg/specialservices"
)

var StartIntegratedCoreDNS = func(_ *options.ControllerContext) error {
	return NewFeatureError("integrated core dns")
}

var InitDNSServiceSyncing = func(_ *options.VirtualClusterOptions) specialservices.Interface {
	return specialservices.NewDefaultServiceSyncer()
}
