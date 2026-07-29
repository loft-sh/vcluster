package pro

import (
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var NewImporter = func(_ synccontext.Mapper) syncertypes.Importer {
	return &noopImporter{}
}

type noopImporter struct{}

func (n *noopImporter) Import(_ *synccontext.SyncContext, _ client.Object) (bool, error) {
	return false, nil
}

func (n *noopImporter) IgnoreHostObject(_ *synccontext.SyncContext, _ client.Object) bool {
	return false
}
