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
	fromHostSyncer := &syncToHostSecretSyncer{}
	fromConfigTranslator, err := translator.NewFromHostTranslatorForGVK(
		ctx, gvk, fromHostSyncer.GetMappings(ctx.Config.Config),
	)
	if err != nil {
		return nil, err
	}
	return syncer.NewFromHost(ctx, fromHostSyncer, fromConfigTranslator)
}

type syncToHostSecretSyncer struct{}

func (s *syncToHostSecretSyncer) CopyHostObjectToVirtual(vObj, pObj client.Object) {
	vCm := vObj.(*corev1.Secret)
	hostCopy := pObj.(*corev1.Secret).DeepCopy()
	vCm.SetAnnotations(hostCopy.GetAnnotations())
	vCm.SetLabels(hostCopy.Labels)
	vCm.Data = hostCopy.Data
}

func (s *syncToHostSecretSyncer) GetProPatches(cfg config.Config) []config.TranslatePatch {
	return cfg.Sync.FromHost.Secrets.Patches
}

func (s *syncToHostSecretSyncer) GetMappings(cfg config.Config) map[string]string {
	return cfg.Sync.FromHost.Secrets.Mappings.ByName
}

func (s *syncToHostSecretSyncer) ExcludeVirtual(_ client.Object) bool {
	return false
}

func (s *syncToHostSecretSyncer) ExcludePhysical(_ client.Object) bool {
	return false
}
