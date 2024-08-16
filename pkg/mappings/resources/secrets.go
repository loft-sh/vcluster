package resources

import (
	"fmt"
	"strings"

	podtranslate "github.com/loft-sh/vcluster/pkg/controllers/resources/pods/token"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
)

func CreateSecretsMapper(ctx *synccontext.RegisterContext) (synccontext.Mapper, error) {
	mapper, err := generic.NewMapper(ctx, &corev1.Secret{}, translate.Default.HostName)
	if err != nil {
		return nil, err
	}

	return &secretsMapper{
		Mapper: mapper,
	}, nil
}

type secretsMapper struct {
	synccontext.Mapper
}

func (s *secretsMapper) Migrate(ctx *synccontext.RegisterContext, mapper synccontext.Mapper) error {
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

	// list all pods and map them by their used secrets
	list := &corev1.PodList{}
	err = ctx.VirtualManager.GetClient().List(ctx, list)
	if err != nil {
		return fmt.Errorf("error listing csi storage capacities: %w", err)
	}

	for _, val := range list.Items {
		item := &val

		// this will try to translate and record the mapping
		syncContext := ctx.ToSyncContext("migrate-pod")
		for _, secret := range secretNamesFromPod(syncContext, item) {
			pName := mapper.VirtualToHost(syncContext, secret, nil)
			if pName.Name != "" {
				err = ctx.Mappings.Store().AddReferenceAndSave(ctx, synccontext.NameMapping{
					GroupVersionKind: mappings.Secrets(),
					VirtualName:      secret,
					HostName:         pName,
				}, synccontext.NameMapping{
					GroupVersionKind: mappings.Pods(),
					VirtualName:      types.NamespacedName{Name: item.Name, Namespace: item.Namespace},
				})
				if err != nil {
					klog.FromContext(ctx).Error(err, "record secret reference on pod")
				}
			}
		}
	}

	// check if ingress sync is enabled
	if ctx.Config.Sync.ToHost.Ingresses.Enabled {
		// list all pods and map them by their used secrets
		list := &networkingv1.IngressList{}
		err = ctx.VirtualManager.GetClient().List(ctx, list)
		if err != nil {
			return fmt.Errorf("error listing csi storage capacities: %w", err)
		}

		for _, val := range list.Items {
			item := &val

			// this will try to translate and record the mapping
			syncContext := ctx.ToSyncContext("migrate-ingress")
			for _, secret := range secretNamesFromIngress(syncContext, item) {
				pName := mapper.VirtualToHost(syncContext, secret, nil)
				if pName.Name != "" {
					err = ctx.Mappings.Store().AddReferenceAndSave(ctx, synccontext.NameMapping{
						GroupVersionKind: mappings.Secrets(),
						VirtualName:      secret,
						HostName:         pName,
					}, synccontext.NameMapping{
						GroupVersionKind: mappings.Pods(),
						VirtualName:      types.NamespacedName{Name: item.Name, Namespace: item.Namespace},
					})
					if err != nil {
						klog.FromContext(ctx).Error(err, "record secret reference on pod")
					}
				}
			}
		}
	}

	return s.Mapper.Migrate(ctx, mapper)
}

var TranslateAnnotations = map[string]bool{
	"nginx.ingress.kubernetes.io/auth-secret":      true,
	"nginx.ingress.kubernetes.io/auth-tls-secret":  true,
	"nginx.ingress.kubernetes.io/proxy-ssl-secret": true,
}

func TranslateIngressAnnotations(ctx *synccontext.SyncContext, annotations map[string]string, ingressNamespace string) (map[string]string, []types.NamespacedName) {
	foundSecrets := []types.NamespacedName{}
	newAnnotations := map[string]string{}
	for k, v := range annotations {
		if !TranslateAnnotations[k] {
			newAnnotations[k] = v
			continue
		}

		splitted := strings.Split(annotations[k], "/")
		if len(splitted) == 1 { // If value is only "secret"
			secret := splitted[0]
			foundSecrets = append(foundSecrets, types.NamespacedName{Namespace: ingressNamespace, Name: secret})
			newAnnotations[k] = mappings.VirtualToHostName(ctx, secret, ingressNamespace, mappings.Secrets())
		} else if len(splitted) == 2 { // If value is "namespace/secret"
			namespace := splitted[0]
			secret := splitted[1]
			foundSecrets = append(foundSecrets, types.NamespacedName{Namespace: namespace, Name: secret})
			pName := mappings.VirtualToHost(ctx, secret, namespace, mappings.Secrets())
			newAnnotations[k] = pName.Namespace + "/" + pName.Name
		} else {
			newAnnotations[k] = v
		}
	}

	return newAnnotations, foundSecrets
}

func secretNamesFromIngress(ctx *synccontext.SyncContext, ingress *networkingv1.Ingress) []types.NamespacedName {
	secrets := []types.NamespacedName{}
	_, extraSecrets := TranslateIngressAnnotations(ctx, ingress.Annotations, ingress.Namespace)
	secrets = append(secrets, extraSecrets...)
	for _, tls := range ingress.Spec.TLS {
		if tls.SecretName != "" {
			secrets = append(secrets, types.NamespacedName{Namespace: ingress.Namespace, Name: tls.SecretName})
		}
	}
	return secrets
}

func secretNamesFromPod(ctx *synccontext.SyncContext, pod *corev1.Pod) []types.NamespacedName {
	secrets := []types.NamespacedName{}
	for _, c := range pod.Spec.Containers {
		secrets = append(secrets, secretNamesFromContainer(pod.Namespace, &c)...)
	}
	for _, c := range pod.Spec.InitContainers {
		secrets = append(secrets, secretNamesFromContainer(pod.Namespace, &c)...)
	}
	for _, c := range pod.Spec.EphemeralContainers {
		secrets = append(secrets, secretNamesFromEphemeralContainer(pod.Namespace, &c)...)
	}
	for i := range pod.Spec.ImagePullSecrets {
		secrets = append(secrets, types.NamespacedName{Namespace: pod.Namespace, Name: pod.Spec.ImagePullSecrets[i].Name})
	}
	secrets = append(secrets, secretNamesFromVolumes(ctx, pod)...)
	return secrets
}

func secretNamesFromVolumes(ctx *synccontext.SyncContext, pod *corev1.Pod) []types.NamespacedName {
	secrets := []types.NamespacedName{}
	for i := range pod.Spec.Volumes {
		if pod.Spec.Volumes[i].Secret != nil {
			secrets = append(secrets, types.NamespacedName{Namespace: pod.Namespace, Name: pod.Spec.Volumes[i].Secret.SecretName})
		}
		if pod.Spec.Volumes[i].Projected != nil {
			for j := range pod.Spec.Volumes[i].Projected.Sources {
				if pod.Spec.Volumes[i].Projected.Sources[j].Secret != nil {
					secrets = append(secrets, types.NamespacedName{Namespace: pod.Namespace, Name: pod.Spec.Volumes[i].Projected.Sources[j].Secret.Name})
				}

				// check if projected volume source is a serviceaccount and in such a case
				// we re-write it as a secret too, handle accordingly
				if ctx != nil && ctx.Config != nil && ctx.Config.Sync.ToHost.Pods.UseSecretsForSATokens {
					if pod.Spec.Volumes[i].Projected.Sources[j].ServiceAccountToken != nil {
						secrets = append(secrets, podtranslate.SecretNameFromPodName(ctx, pod.Name, pod.Namespace))
					}
				}
			}
		}
		if pod.Spec.Volumes[i].ISCSI != nil && pod.Spec.Volumes[i].ISCSI.SecretRef != nil {
			secrets = append(secrets, types.NamespacedName{Namespace: pod.Namespace, Name: pod.Spec.Volumes[i].ISCSI.SecretRef.Name})
		}
		if pod.Spec.Volumes[i].RBD != nil && pod.Spec.Volumes[i].RBD.SecretRef != nil {
			secrets = append(secrets, types.NamespacedName{Namespace: pod.Namespace, Name: pod.Spec.Volumes[i].RBD.SecretRef.Name})
		}
		if pod.Spec.Volumes[i].FlexVolume != nil && pod.Spec.Volumes[i].FlexVolume.SecretRef != nil {
			secrets = append(secrets, types.NamespacedName{Namespace: pod.Namespace, Name: pod.Spec.Volumes[i].FlexVolume.SecretRef.Name})
		}
		if pod.Spec.Volumes[i].Cinder != nil && pod.Spec.Volumes[i].Cinder.SecretRef != nil {
			secrets = append(secrets, types.NamespacedName{Namespace: pod.Namespace, Name: pod.Spec.Volumes[i].Cinder.SecretRef.Name})
		}
		if pod.Spec.Volumes[i].CephFS != nil && pod.Spec.Volumes[i].CephFS.SecretRef != nil {
			secrets = append(secrets, types.NamespacedName{Namespace: pod.Namespace, Name: pod.Spec.Volumes[i].CephFS.SecretRef.Name})
		}
		if pod.Spec.Volumes[i].AzureFile != nil && pod.Spec.Volumes[i].AzureFile.SecretName != "" {
			secrets = append(secrets, types.NamespacedName{Namespace: pod.Namespace, Name: pod.Spec.Volumes[i].AzureFile.SecretName})
		}
		if pod.Spec.Volumes[i].ScaleIO != nil && pod.Spec.Volumes[i].ScaleIO.SecretRef != nil {
			secrets = append(secrets, types.NamespacedName{Namespace: pod.Namespace, Name: pod.Spec.Volumes[i].ScaleIO.SecretRef.Name})
		}
		if pod.Spec.Volumes[i].StorageOS != nil && pod.Spec.Volumes[i].StorageOS.SecretRef != nil {
			secrets = append(secrets, types.NamespacedName{Namespace: pod.Namespace, Name: pod.Spec.Volumes[i].StorageOS.SecretRef.Name})
		}
		if pod.Spec.Volumes[i].CSI != nil && pod.Spec.Volumes[i].CSI.NodePublishSecretRef != nil {
			secrets = append(secrets, types.NamespacedName{Namespace: pod.Namespace, Name: pod.Spec.Volumes[i].CSI.NodePublishSecretRef.Name})
		}
	}
	return secrets
}

func secretNamesFromContainer(namespace string, container *corev1.Container) []types.NamespacedName {
	secrets := []types.NamespacedName{}
	for _, env := range container.Env {
		if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil && env.ValueFrom.SecretKeyRef.Name != "" {
			secrets = append(secrets, types.NamespacedName{Namespace: namespace, Name: env.ValueFrom.SecretKeyRef.Name})
		}
	}
	for _, from := range container.EnvFrom {
		if from.SecretRef != nil && from.SecretRef.Name != "" {
			secrets = append(secrets, types.NamespacedName{Namespace: namespace, Name: from.SecretRef.Name})
		}
	}
	return secrets
}

func secretNamesFromEphemeralContainer(namespace string, container *corev1.EphemeralContainer) []types.NamespacedName {
	secrets := []types.NamespacedName{}
	for _, env := range container.Env {
		if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil && env.ValueFrom.SecretKeyRef.Name != "" {
			secrets = append(secrets, types.NamespacedName{Namespace: namespace, Name: env.ValueFrom.SecretKeyRef.Name})
		}
	}
	for _, from := range container.EnvFrom {
		if from.SecretRef != nil && from.SecretRef.Name != "" {
			secrets = append(secrets, types.NamespacedName{Namespace: namespace, Name: from.SecretRef.Name})
		}
	}
	return secrets
}
