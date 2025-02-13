package configmaps

import (
	"fmt"

	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func NewFromHost(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	gvk, err := apiutil.GVKForObject(&corev1.ConfigMap{}, scheme.Scheme)
	if err != nil {
		return nil, fmt.Errorf("retrieve GVK for object failed: %w", err)
	}
	syncCtx := ctx.ToSyncContext("from-host-configmap-syncer")
	fromHostSyncer := &syncToHostConfigMapSyncer{}
	fromConfigTranslator, err := translator.NewFromHostTranslatorForGVK(ctx, gvk, fromHostSyncer.GetMappings(syncCtx), skipKubeRootCaConfigMap)
	if err != nil {
		return nil, err
	}
	return syncer.NewFromHost(ctx, fromHostSyncer, fromConfigTranslator, skipKubeRootCaConfigMap)
}

func skipKubeRootCaConfigMap(hostName, _ string) bool {
	return hostName == "kube-root-ca.crt"
}

type syncToHostConfigMapSyncer struct{}

func (s *syncToHostConfigMapSyncer) SyncToHost(vObj, pObj client.Object) {
	vCm := vObj.(*corev1.ConfigMap)
	hostCopy := pObj.(*corev1.ConfigMap).DeepCopy()
	vCm.SetAnnotations(hostCopy.GetAnnotations())
	vCm.SetLabels(hostCopy.Labels)
	vCm.Data = hostCopy.Data
}

func (s *syncToHostConfigMapSyncer) GetProPatches(ctx *synccontext.SyncContext) []config.TranslatePatch {
	return ctx.Config.Sync.FromHost.ConfigMaps.Patches
}

func (s *syncToHostConfigMapSyncer) GetMappings(ctx *synccontext.SyncContext) map[string]string {
	return ctx.Config.Sync.FromHost.ConfigMaps.Selector.Mappings
}
