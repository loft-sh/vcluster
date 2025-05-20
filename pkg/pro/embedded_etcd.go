package pro

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
)

var StartEmbeddedEtcd = func(_ context.Context, _, _, _ string, _, _ int, _ string, _, _ bool, _ []string) error {
	return NewFeatureError("embedded etcd")
}

var StartAutoHealingController = func(*synccontext.ControllerContext, string) error {
	return NewFeatureError("embedded etcd")
}
