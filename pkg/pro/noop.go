package pro

import (
	"github.com/loft-sh/vcluster/pkg/options"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var SyncNoopSyncerEndpoints = func(_ *options.ControllerContext, _ types.NamespacedName, _ client.Client, _ types.NamespacedName, _ string) error {
	return NewFeatureError("noop syncer")
}
