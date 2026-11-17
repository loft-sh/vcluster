package resources

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateConfigMapsMapper(ctx *synccontext.RegisterContext) (synccontext.Mapper, error) {
	mapper, err := generic.NewMapperWithoutRecorder(ctx, &corev1.ConfigMap{}, func(ctx *synccontext.SyncContext, vName, vNamespace string, _ client.Object) types.NamespacedName {
		return translate.Default.HostName(ctx, vName, vNamespace)
	})
	if err != nil {
		return nil, err
	}

	return generic.WithRecorder(&configMapsMapper{
		Mapper: mapper,
	}), nil
}

type configMapsMapper struct {
	synccontext.Mapper
}

func (s *configMapsMapper) VirtualToHost(ctx *synccontext.SyncContext, req types.NamespacedName, vObj client.Object) types.NamespacedName {
	pName := s.Mapper.VirtualToHost(ctx, req, vObj)
	if pName.Name == "kube-root-ca.crt" {
		return types.NamespacedName{
			Name:      translate.SafeConcatName("vcluster", "kube-root-ca.crt", "x", translate.VClusterName),
			Namespace: pName.Namespace,
		}
	}

	return pName
}

func (s *configMapsMapper) HostToVirtual(ctx *synccontext.SyncContext, req types.NamespacedName, pObj client.Object) types.NamespacedName {
	// ignore kube-root-ca.crt from host
	if req.Name == "kube-root-ca.crt" {
		return types.NamespacedName{}
	}

	// translate the special kube-root-ca.crt back
	if req.Name == translate.SafeConcatName("vcluster", "kube-root-ca.crt", "x", translate.VClusterName) {
		return types.NamespacedName{
			Name:      "kube-root-ca.crt",
			Namespace: s.Mapper.HostToVirtual(ctx, req, pObj).Namespace,
		}
	}

	return s.Mapper.HostToVirtual(ctx, req, pObj)
}

func (s *configMapsMapper) Migrate(ctx *synccontext.RegisterContext, mapper synccontext.Mapper) error {
	// make sure we migrate pods first
	podsMapper, err := ctx.Mappings.ByGVK(mappings.Pods())
	if err != nil {
		return err
	}

	// pods mapper
	err = podsMapper.Migrate(ctx, podsMapper)
	if err != nil {
		return fmt.Errorf("migrate pods")
	}

	// list all pods and map them by their used configmaps
	list := &corev1.PodList{}
	err = ctx.VirtualManager.GetClient().List(ctx, list)
	if err != nil {
		return fmt.Errorf("error listing csi storage capacities: %w", err)
	}

	for _, val := range list.Items {
		item := &val

		// this will try to translate and record the mapping
		for _, configMap := range configMapsFromPod(item) {
			pName := mapper.VirtualToHost(ctx.ToSyncContext("migrate-pod"), configMap, nil)
			if pName.Name != "" {
				err = ctx.Mappings.Store().AddReferenceAndSave(ctx, synccontext.NameMapping{
					GroupVersionKind: mappings.ConfigMaps(),
					VirtualName:      configMap,
					HostName:         pName,
				}, synccontext.NameMapping{
					GroupVersionKind: mappings.Pods(),
					VirtualName:      types.NamespacedName{Name: item.Name, Namespace: item.Namespace},
				})
				if err != nil {
					klog.FromContext(ctx).Error(err, "record config map reference on pod")
				}
			}
		}
	}

	return s.Mapper.Migrate(ctx, mapper)
}

func configMapsFromPod(pod *corev1.Pod) []types.NamespacedName {
	configMaps := []types.NamespacedName{}
	for _, c := range pod.Spec.Containers {
		configMaps = append(configMaps, configNamesFromContainer(pod.Namespace, &c)...)
	}
	for _, c := range pod.Spec.InitContainers {
		configMaps = append(configMaps, configNamesFromContainer(pod.Namespace, &c)...)
	}
	for _, c := range pod.Spec.EphemeralContainers {
		configMaps = append(configMaps, configNamesFromEphemeralContainer(pod.Namespace, &c)...)
	}
	for i := range pod.Spec.Volumes {
		if pod.Spec.Volumes[i].ConfigMap != nil {
			configMaps = append(configMaps, types.NamespacedName{Namespace: pod.Namespace, Name: pod.Spec.Volumes[i].ConfigMap.Name})
		}
		if pod.Spec.Volumes[i].Projected != nil {
			for j := range pod.Spec.Volumes[i].Projected.Sources {
				if pod.Spec.Volumes[i].Projected.Sources[j].ConfigMap != nil {
					configMaps = append(configMaps, types.NamespacedName{Namespace: pod.Namespace, Name: pod.Spec.Volumes[i].Projected.Sources[j].ConfigMap.Name})
				}
			}
		}
	}
	return configMaps
}

func configNamesFromContainer(namespace string, container *corev1.Container) []types.NamespacedName {
	configNames := []types.NamespacedName{}
	for _, env := range container.Env {
		if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil && env.ValueFrom.ConfigMapKeyRef.Name != "" {
			configNames = append(configNames, types.NamespacedName{Namespace: namespace, Name: env.ValueFrom.ConfigMapKeyRef.Name})
		}
	}
	for _, from := range container.EnvFrom {
		if from.ConfigMapRef != nil && from.ConfigMapRef.Name != "" {
			configNames = append(configNames, types.NamespacedName{Namespace: namespace, Name: from.ConfigMapRef.Name})
		}
	}
	return configNames
}

func configNamesFromEphemeralContainer(namespace string, container *corev1.EphemeralContainer) []types.NamespacedName {
	configNames := []types.NamespacedName{}
	for _, env := range container.Env {
		if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil && env.ValueFrom.ConfigMapKeyRef.Name != "" {
			configNames = append(configNames, types.NamespacedName{Namespace: namespace, Name: env.ValueFrom.ConfigMapKeyRef.Name})
		}
	}
	for _, from := range container.EnvFrom {
		if from.ConfigMapRef != nil && from.ConfigMapRef.Name != "" {
			configNames = append(configNames, types.NamespacedName{Namespace: namespace, Name: from.ConfigMapRef.Name})
		}
	}
	return configNames
}
