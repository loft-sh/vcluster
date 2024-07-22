package pro

import (
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var SyncNoopSyncerEndpoints = func(_ *synccontext.ControllerContext, _ types.NamespacedName, _ client.Client, _ types.NamespacedName, _ string) error {
	return NewFeatureError("noop syncer")
}
