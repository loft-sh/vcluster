package secrets

import (
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/pkg/k0s"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/controllers/generic"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func NewFromHost(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	gvk, err := apiutil.GVKForObject(&corev1.ConfigMap{}, scheme.Scheme)
	if err != nil {
		return nil, fmt.Errorf("retrieve GVK for object failed: %w", err)
	}
	syncCtx := ctx.ToSyncContext("from-host-configmap-syncer")
	virtualToExclude := checkExperimentalDeployConfig(ctx)
	fromHostSyncer := &syncToHostConfigMapSyncer{
		virtualObjectsToExclude: virtualToExclude,
	}
	fromConfigTranslator, err := translator.NewFromHostTranslatorForGVK(ctx, gvk, fromHostSyncer.GetMappings(syncCtx), skipKubeRootCaConfigMap)
	if err != nil {
		return nil, err
	}
	return generic.NewFromHost(ctx, fromHostSyncer, fromConfigTranslator, skipKubeRootCaConfigMap)
}

func skipKubeRootCaConfigMap(hostName, _ string) bool {
	return hostName == "kube-root-ca.crt"
}

type syncToHostConfigMapSyncer struct {
	virtualObjectsToExclude map[string]bool
}

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

func (s *syncToHostConfigMapSyncer) ExcludeVirtual(vObj client.Object) bool {
	_, found := s.virtualObjectsToExclude[vObj.GetNamespace()+"/"+vObj.GetName()]
	if found {
		klog.Infof("excluding virtual object %s in namespace %s because it is part of experimental.deploy manifests", vObj.GetName(), vObj.GetNamespace())
	}
	return found
}

func (s *syncToHostConfigMapSyncer) ExcludePhysical(_ client.Object) bool {
	return false
}

func checkExperimentalDeployConfig(ctx *synccontext.RegisterContext) map[string]bool {
	deploy := ctx.Config.Experimental.Deploy
	virtualConfigMapsToSkip := make(map[string]bool)
	if strings.Contains(deploy.VCluster.Manifests, "---") {
		for _, manifest := range strings.Split(deploy.VCluster.Manifests, "---") {
			configMapKey, found := processManifest(manifest)
			if found {
				virtualConfigMapsToSkip[configMapKey] = true
			}
		}
	} else {
		configMapKey, found := processManifest(deploy.VCluster.Manifests)
		if found {
			virtualConfigMapsToSkip[configMapKey] = true
		}
	}

	if strings.Contains(deploy.VCluster.ManifestsTemplate, "---") {
		for _, manifest := range strings.Split(deploy.VCluster.ManifestsTemplate, "---") {
			configMapKey, found := processTemplate(manifest, &ctx.Config.Config, ctx.Config.Name, ctx.Config.WorkloadTargetNamespace)
			if found {
				virtualConfigMapsToSkip[configMapKey] = true
			}
		}
	} else {
		configMapKey, found := processTemplate(deploy.VCluster.ManifestsTemplate, &ctx.Config.Config, ctx.Config.Name, ctx.Config.WorkloadTargetNamespace)
		if found {
			virtualConfigMapsToSkip[configMapKey] = true
		}
	}
	return virtualConfigMapsToSkip
}

func processManifest(manifest string) (string, bool) {
	manifest = strings.TrimSpace(manifest)
	if manifest == "" {
		return "", false
	}
	cm := &corev1.ConfigMap{}
	err := yaml.Unmarshal([]byte(manifest), cm)
	if err != nil {
		return "", false
	}
	name, ns := cm.GetName(), cm.GetNamespace()
	if ns == "" {
		ns = "default"
	}
	return ns + "/" + name, true
}

func processTemplate(manifest string, vConfig *config.Config, name, targetNs string) (string, bool) {
	templatedManifests, err := k0s.ExecTemplate(manifest, name, targetNs, vConfig)
	if err != nil {
		return "", false
	}
	return processManifest(string(templatedManifests))
}
