package pro

import (
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
)

var StartCustomResourceProxy = func(_ *synccontext.ControllerContext, _ *config.VirtualClusterConfig) error {
	return NewFeatureError("resource proxy")
}
