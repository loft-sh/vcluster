package pro

import (
	"github.com/loft-sh/admin-apis/pkg/licenseapi"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/specialservices"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
)

var StartIntegratedCoreDNS = func(_ *synccontext.ControllerContext) error {
	return NewFeatureError(string(licenseapi.VirtualClusterProDistroBuiltInCoreDNS))
}

var InitDNSServiceSyncing = func(_ *config.VirtualClusterConfig) specialservices.Interface {
	return specialservices.NewDefaultServiceSyncer()
}
