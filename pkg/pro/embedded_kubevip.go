package pro

import (
	"github.com/loft-sh/admin-apis/pkg/licenseapi"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
)

var StartEmbeddedKubeVip = func(_ *synccontext.ControllerContext, _ *config.VirtualClusterConfig) error {
	return NewFeatureError(string(licenseapi.KubeVip))
}
