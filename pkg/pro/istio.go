package pro

import (
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ApplyIstioPatches = func(_ *synccontext.SyncContext, _, _, _ client.Object) error {
	return nil
}
