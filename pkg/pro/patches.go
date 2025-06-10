package pro

import (
	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ApplyPatchesVirtualObject = func(_ *synccontext.SyncContext, _, _, _ client.Object, patches []config.TranslatePatch, _ bool) error {
	if len(patches) == 0 {
		return nil
	}

	return NewFeatureError("translate patches")
}

var ApplyPatchesHostObject = func(_ *synccontext.SyncContext, _, _, _ client.Object, patches []config.TranslatePatch, _ bool) error {
	if len(patches) == 0 {
		return nil
	}

	return NewFeatureError("translate patches")
}
