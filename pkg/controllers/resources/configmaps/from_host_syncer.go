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
	fromHostSyncer := &syncToHostConfigMapSyncer{}
	fromConfigTranslator, err := translator.NewFromHostTranslatorForGVK(ctx, gvk, fromHostSyncer.GetMappings(ctx.Config.Config), skipKubeRootCaConfigMap)
	if err != nil {
		return nil, err
	}
	return syncer.NewFromHost(ctx, fromHostSyncer, fromConfigTranslator, skipKubeRootCaConfigMap)
}

func skipKubeRootCaConfigMap(hostName, _ string) bool {
	return hostName == "kube-root-ca.crt"
}

type syncToHostConfigMapSyncer struct{}

func (s *syncToHostConfigMapSyncer) CopyHostObjectToVirtual(vObj, pObj client.Object) {
	vCm := vObj.(*corev1.ConfigMap)
	hostCopy := pObj.(*corev1.ConfigMap).DeepCopy()
	vCm.SetAnnotations(hostCopy.GetAnnotations())
	vCm.SetLabels(hostCopy.Labels)
	vCm.Data = hostCopy.Data
}

func (s *syncToHostConfigMapSyncer) GetProPatches(cfg config.Config) []config.TranslatePatch {
	return cfg.Sync.FromHost.ConfigMaps.Patches
}

func (s *syncToHostConfigMapSyncer) GetMappings(cfg config.Config) map[string]string {
	return cfg.Sync.FromHost.ConfigMaps.Mappings.ByName
}
