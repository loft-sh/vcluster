package pro

import (
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
)

var RegisterProControllers = func(*synccontext.ControllerContext) error {
	return nil
}

var BuildProSyncers = func(_ *synccontext.RegisterContext) ([]syncertypes.Object, error) {
	return []syncertypes.Object{}, nil
}
