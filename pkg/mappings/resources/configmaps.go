package resources

import (
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateConfigMapsMapper(ctx *synccontext.RegisterContext) (synccontext.Mapper, error) {
	mapper, err := generic.NewMapper(ctx, &corev1.ConfigMap{}, translate.Default.PhysicalName, generic.SkipIndex())
	if err != nil {
		return nil, err
	}

	err = ctx.VirtualManager.GetFieldIndexer().IndexField(ctx, &corev1.ConfigMap{}, constants.IndexByPhysicalName, func(rawObj client.Object) []string {
		if !translate.Default.SingleNamespaceTarget() && rawObj.GetName() == "kube-root-ca.crt" {
			return []string{translate.Default.PhysicalNamespace(rawObj.GetNamespace()) + "/" + translate.SafeConcatName("vcluster", "kube-root-ca.crt", "x", translate.VClusterName)}
		}

		return []string{translate.Default.PhysicalNamespace(rawObj.GetNamespace()) + "/" + translate.Default.PhysicalName(rawObj.GetName(), rawObj.GetNamespace())}
	})
	if err != nil {
		return nil, err
	}

	return &configMapsMapper{
		Mapper: mapper,
	}, nil
}

type configMapsMapper struct {
	synccontext.Mapper
}

func (s *configMapsMapper) VirtualToHost(ctx *synccontext.SyncContext, req types.NamespacedName, vObj client.Object) types.NamespacedName {
	if !translate.Default.SingleNamespaceTarget() && req.Name == "kube-root-ca.crt" {
		return types.NamespacedName{
			Name:      translate.SafeConcatName("vcluster", "kube-root-ca.crt", "x", translate.VClusterName),
			Namespace: s.Mapper.VirtualToHost(ctx, req, vObj).Namespace,
		}
	}

	return s.Mapper.VirtualToHost(ctx, req, vObj)
}

func (s *configMapsMapper) HostToVirtual(ctx *synccontext.SyncContext, req types.NamespacedName, pObj client.Object) types.NamespacedName {
	if !translate.Default.SingleNamespaceTarget() && req.Name == translate.SafeConcatName("vcluster", "kube-root-ca.crt", "x", translate.VClusterName) {
		return types.NamespacedName{
			Name:      "kube-root-ca.crt",
			Namespace: s.Mapper.HostToVirtual(ctx, req, pObj).Namespace,
		}
	} else if !translate.Default.SingleNamespaceTarget() && req.Name == "kube-root-ca.crt" {
		// ignore kube-root-ca.crt from host
		return types.NamespacedName{}
	}

	return s.Mapper.HostToVirtual(ctx, req, pObj)
}
