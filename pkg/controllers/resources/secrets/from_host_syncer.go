package secrets

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/syncer"

	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func NewFromHost(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	gvk, err := apiutil.GVKForObject(&corev1.Secret{}, scheme.Scheme)
	if err != nil {
		return nil, fmt.Errorf("retrieve GVK for object failed: %w", err)
	}
	syncCtx := ctx.ToSyncContext("from-host-secret-syncer")
	fromHostSyncer := &syncToHostSecretSyncer{}
	fromConfigTranslator, err := translator.NewFromHostTranslatorForGVK(
		ctx, gvk, fromHostSyncer.GetMappings(syncCtx),
	)
	if err != nil {
		return nil, err
	}
	return syncer.NewFromHost(ctx, fromHostSyncer, fromConfigTranslator)
}

type syncToHostSecretSyncer struct{}

func (s *syncToHostSecretSyncer) SyncToHost(vObj, pObj client.Object) {
	vCm := vObj.(*corev1.Secret)
	hostCopy := pObj.(*corev1.Secret).DeepCopy()
	vCm.SetAnnotations(hostCopy.GetAnnotations())
	vCm.SetLabels(hostCopy.Labels)
	vCm.Data = hostCopy.Data
}

func (s *syncToHostSecretSyncer) GetProPatches(ctx *synccontext.SyncContext) []config.TranslatePatch {
	return ctx.Config.Sync.FromHost.Secrets.Patches
}

func (s *syncToHostSecretSyncer) GetMappings(ctx *synccontext.SyncContext) map[string]string {
	return ctx.Config.Sync.FromHost.Secrets.Selector.Mappings
}

func (s *syncToHostSecretSyncer) ExcludeVirtual(_ client.Object) bool {
	return false
}

func (s *syncToHostSecretSyncer) ExcludePhysical(_ client.Object) bool {
	return false
}
