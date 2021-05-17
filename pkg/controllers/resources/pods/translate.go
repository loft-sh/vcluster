package pods

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"regexp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sort"
	"strings"

	"github.com/loft-sh/vcluster/pkg/util/translate"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

const (
	OwnerSetKind                 = "vcluster.loft.sh/owner-set-kind"
	NamespaceAnnotation          = "vcluster.loft.sh/namespace"
	NameAnnotation               = "vcluster.loft.sh/name"
	LabelsAnnotation             = "vcluster.loft.sh/labels"
	UIDAnnotation                = "vcluster.loft.sh/uid"
	ServiceAccountNameAnnotation = "vcluster.loft.sh/service-account-name"

	DisableSubdomainRewriteAnnotation = "vcluster.loft.sh/disable-subdomain-rewrite"
	HostsRewrittenAnnotation          = "vcluster.loft.sh/hosts-rewritten"
	HostsVolumeName                   = "vcluster-rewrite-hosts"
	HostsRewriteImage                 = "alpine:3.13.1"
	HostsRewriteContainerName         = "vcluster-rewrite-hosts"
)

var (
	FieldPathLabelRegEx = regexp.MustCompile("^metadata\\.labels\\['(.+)'\\]$")
)

func translatePod(pPod *corev1.Pod, vPod *corev1.Pod, vClient client.Client, services []*corev1.Service, clusterDomain, dnsIP, kubeIP, serviceAccount string, translator ImageTranslator, enableOverrideHosts bool, overrideHostsImage string) error {
	pPod.Status = corev1.PodStatus{}
	pPod.Spec.DeprecatedServiceAccount = ""
	pPod.Spec.ServiceAccountName = serviceAccount
	pPod.Spec.AutomountServiceAccountToken = &False
	pPod.Spec.EnableServiceLinks = &False

	// Add an annotation for namespace, name and uid
	if pPod.Annotations == nil {
		pPod.Annotations = map[string]string{}
	}
	if vPod.Annotations != nil {
		pPod.Annotations[NamespaceAnnotation] = vPod.Annotations[NamespaceAnnotation]
		pPod.Annotations[NameAnnotation] = vPod.Annotations[NameAnnotation]
		pPod.Annotations[UIDAnnotation] = vPod.Annotations[UIDAnnotation]
		pPod.Annotations[ServiceAccountNameAnnotation] = vPod.Annotations[ServiceAccountNameAnnotation]
		if _, ok := vPod.Annotations[LabelsAnnotation]; ok {
			pPod.Annotations[LabelsAnnotation] = vPod.Annotations[LabelsAnnotation]
		}
	}
	if pPod.Annotations[NamespaceAnnotation] == "" {
		pPod.Annotations[NamespaceAnnotation] = vPod.Namespace
	}
	if pPod.Annotations[NameAnnotation] == "" {
		pPod.Annotations[NameAnnotation] = vPod.Name
	}
	if pPod.Annotations[UIDAnnotation] == "" {
		pPod.Annotations[UIDAnnotation] = string(vPod.UID)
	}
	if pPod.Annotations[ServiceAccountNameAnnotation] == "" {
		pPod.Annotations[ServiceAccountNameAnnotation] = vPod.Spec.ServiceAccountName
	}
	if _, ok := pPod.Annotations[LabelsAnnotation]; !ok {
		pPod.Annotations[LabelsAnnotation] = translateLabelsAnnotation(vPod)
	}

	// translate services to environment variables
	serviceEnv := translateServicesToEnvironmentVariables(vPod.Spec.EnableServiceLinks, services, kubeIP)

	// add the required kubernetes hosts entry
	pPod.Spec.HostAliases = append(pPod.Spec.HostAliases, corev1.HostAlias{
		IP:        kubeIP,
		Hostnames: []string{"kubernetes", "kubernetes.default", "kubernetes.default.svc"},
	})

	// truncate hostname if needed
	if pPod.Spec.Hostname == "" {
		if len(vPod.Name) > 63 {
			pPod.Spec.Hostname = vPod.Name[0:63]
		} else {
			pPod.Spec.Hostname = vPod.Name
		}
	}

	// translate the dns config
	translateDNSConfig(pPod, vPod, clusterDomain, dnsIP)

	// if spec.subdomain is set we have to translate the /etc/hosts
	// because otherwise we could get a different hostname as if the pod
	// would be deployed in a non virtual kubernetes cluster
	overrideHosts := false
	if pPod.Spec.Subdomain != "" {
		if enableOverrideHosts == true && (pPod.Annotations == nil || pPod.Annotations[DisableSubdomainRewriteAnnotation] != "true") {
			overrideHosts = true
			initContainer := corev1.Container{
				Name:    HostsRewriteContainerName,
				Image:   overrideHostsImage,
				Command: []string{"sh"},
				Args:    []string{"-c", "sed -E -e 's/^(\\d+.\\d+.\\d+.\\d+\\s+)" + pPod.Spec.Hostname + "$/\\1 " + pPod.Spec.Hostname + "." + pPod.Spec.Subdomain + "." + vPod.Namespace + ".svc." + clusterDomain + " " + pPod.Spec.Hostname + "/' /etc/hosts > /hosts/hosts"},
				VolumeMounts: []corev1.VolumeMount{
					{
						MountPath: "/hosts",
						Name:      HostsVolumeName,
					},
				},
			}

			// Add volume
			if pPod.Spec.Volumes == nil {
				pPod.Spec.Volumes = []corev1.Volume{}
			}
			pPod.Spec.Volumes = append(pPod.Spec.Volumes, corev1.Volume{
				Name: HostsVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			})

			// Add init container
			newContainers := []corev1.Container{initContainer}
			newContainers = append(newContainers, pPod.Spec.InitContainers...)
			pPod.Spec.InitContainers = newContainers

			if pPod.Annotations == nil {
				pPod.Annotations = map[string]string{}
			}
			pPod.Annotations[HostsRewrittenAnnotation] = "true"
		}

		pPod.Spec.Subdomain = ""
	}

	// translate containers
	for i := range pPod.Spec.Containers {
		translateContainerEnv(&pPod.Spec.Containers[i], vPod, serviceEnv)
		pPod.Spec.Containers[i].Image = translator.Translate(pPod.Spec.Containers[i].Image)

		if overrideHosts {
			if pPod.Spec.Containers[i].VolumeMounts == nil {
				pPod.Spec.Containers[i].VolumeMounts = []corev1.VolumeMount{}
			}
			pPod.Spec.Containers[i].VolumeMounts = append(pPod.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
				MountPath: "/etc/hosts",
				Name:      HostsVolumeName,
				SubPath:   "hosts",
			})
		}
	}

	// translate init containers
	for i := range pPod.Spec.InitContainers {
		translateContainerEnv(&pPod.Spec.InitContainers[i], vPod, serviceEnv)
		pPod.Spec.InitContainers[i].Image = translator.Translate(pPod.Spec.InitContainers[i].Image)

		if pPod.Spec.InitContainers[i].Name != HostsRewriteContainerName && overrideHosts {
			if pPod.Spec.InitContainers[i].VolumeMounts == nil {
				pPod.Spec.InitContainers[i].VolumeMounts = []corev1.VolumeMount{}
			}
			pPod.Spec.InitContainers[i].VolumeMounts = append(pPod.Spec.InitContainers[i].VolumeMounts, corev1.VolumeMount{
				MountPath: "/etc/hosts",
				Name:      HostsVolumeName,
				SubPath:   "hosts",
			})
		}
	}

	// translate ephemereal containers
	for i := range pPod.Spec.EphemeralContainers {
		translateEphemerealContainerEnv(&pPod.Spec.EphemeralContainers[i], vPod, serviceEnv)
		pPod.Spec.EphemeralContainers[i].Image = translator.Translate(pPod.Spec.EphemeralContainers[i].Image)

		if overrideHosts {
			if pPod.Spec.EphemeralContainers[i].VolumeMounts == nil {
				pPod.Spec.EphemeralContainers[i].VolumeMounts = []corev1.VolumeMount{}
			}
			pPod.Spec.EphemeralContainers[i].VolumeMounts = append(pPod.Spec.EphemeralContainers[i].VolumeMounts, corev1.VolumeMount{
				MountPath: "/etc/hosts",
				Name:      HostsVolumeName,
				SubPath:   "hosts",
			})
		}
	}

	// translate image pull secrets
	for i := range pPod.Spec.ImagePullSecrets {
		pPod.Spec.ImagePullSecrets[i].Name = translate.PhysicalName(pPod.Spec.ImagePullSecrets[i].Name, vPod.Namespace)
	}

	// translate volumes
	for i := range pPod.Spec.Volumes {
		if pPod.Spec.Volumes[i].ConfigMap != nil {
			pPod.Spec.Volumes[i].ConfigMap.Name = translate.PhysicalName(pPod.Spec.Volumes[i].ConfigMap.Name, vPod.Namespace)
		}
		if pPod.Spec.Volumes[i].Secret != nil {
			pPod.Spec.Volumes[i].Secret.SecretName = translate.PhysicalName(pPod.Spec.Volumes[i].Secret.SecretName, vPod.Namespace)
		}
		if pPod.Spec.Volumes[i].PersistentVolumeClaim != nil {
			pPod.Spec.Volumes[i].PersistentVolumeClaim.ClaimName = translate.PhysicalName(pPod.Spec.Volumes[i].PersistentVolumeClaim.ClaimName, vPod.Namespace)
		}
		if pPod.Spec.Volumes[i].Projected != nil {
			// get old service account name
			err := translateProjectedVolume(pPod.Spec.Volumes[i].Projected, vClient, vPod)
			if err != nil {
				return err
			}
		}
		if pPod.Spec.Volumes[i].DownwardAPI != nil {
			for j := range pPod.Spec.Volumes[i].DownwardAPI.Items {
				translateFieldRef(vPod, pPod.Spec.Volumes[i].DownwardAPI.Items[j].FieldRef)
			}
		}
	}

	// we add an annotation if the pod has a replica set or statefulset owner
	for _, ownerReference := range vPod.OwnerReferences {
		if ownerReference.APIVersion == appsv1.SchemeGroupVersion.String() && (ownerReference.Kind == "StatefulSet" || ownerReference.Kind == "ReplicaSet" || ownerReference.Kind == "DaemonSet") {
			if pPod.Annotations == nil {
				pPod.Annotations = map[string]string{}
			}

			pPod.Annotations[OwnerSetKind] = ownerReference.Kind
			break
		}
	}

	// translate topology spread constraints
	translateTopologySpreadConstraints(vPod, pPod)

	return nil
}

func translateLabelsAnnotation(vPod *corev1.Pod) string {
	labelsString := []string{}
	for k, v := range vPod.Labels {
		// escape pod labels
		out, err := json.Marshal(v)
		if err != nil {
			continue
		}

		labelsString = append(labelsString, k+"="+string(out))
	}

	return strings.Join(labelsString, "\n")
}

func secretNameFromServiceAccount(vClient client.Client, vPod *corev1.Pod) (string, error) {
	vServiceAccount := ""
	if vPod.Spec.ServiceAccountName != "" {
		vServiceAccount = vPod.Spec.ServiceAccountName
	} else if vPod.Spec.DeprecatedServiceAccount != "" {
		vServiceAccount = vPod.Spec.DeprecatedServiceAccount
	}

	secretList := &corev1.SecretList{}
	err := vClient.List(context.Background(), secretList, client.InNamespace(vPod.Namespace))
	if err != nil {
		return "", errors.Wrap(err, "list secrets in "+vPod.Namespace)
	}
	for _, secret := range secretList.Items {
		if secret.Annotations["kubernetes.io/service-account.name"] == vServiceAccount {
			return secret.Name, nil
		}
	}

	return "", nil
}

func translateProjectedVolume(projectedVolume *corev1.ProjectedVolumeSource, vClient client.Client, vPod *corev1.Pod) error {
	for i := range projectedVolume.Sources {
		if projectedVolume.Sources[i].Secret != nil {
			projectedVolume.Sources[i].Secret.Name = translate.PhysicalName(projectedVolume.Sources[i].Secret.Name, vPod.Namespace)
		}
		if projectedVolume.Sources[i].ConfigMap != nil {
			projectedVolume.Sources[i].ConfigMap.Name = translate.PhysicalName(projectedVolume.Sources[i].ConfigMap.Name, vPod.Namespace)
		}
		if projectedVolume.Sources[i].DownwardAPI != nil {
			for j := range projectedVolume.Sources[i].DownwardAPI.Items {
				translateFieldRef(vPod, projectedVolume.Sources[i].DownwardAPI.Items[j].FieldRef)
			}
		}
		if projectedVolume.Sources[i].ServiceAccountToken != nil {
			secretName, err := secretNameFromServiceAccount(vClient, vPod)
			if err != nil {
				return err
			} else if secretName == "" {
				return fmt.Errorf("couldn't find service account secret for pod %s/%s", vPod.Namespace, vPod.Name)
			}

			allRights := int32(0644)
			projectedVolume.Sources[i].Secret = &corev1.SecretProjection{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: translate.PhysicalName(secretName, vPod.Namespace),
				},
				Items: []corev1.KeyToPath{
					{
						Path: projectedVolume.Sources[i].ServiceAccountToken.Path,
						Key:  "token",
						Mode: &allRights,
					},
				},
			}
			projectedVolume.Sources[i].ServiceAccountToken = nil
		}
	}

	return nil
}

func translateFieldRef(vPod *corev1.Pod, fieldSelector *corev1.ObjectFieldSelector) {
	if fieldSelector == nil {
		return
	}

	// check if its a label we have to rewrite
	labelsMatch := FieldPathLabelRegEx.FindStringSubmatch(fieldSelector.FieldPath)
	if len(labelsMatch) == 2 {
		fieldSelector.FieldPath = "metadata.labels['" + translate.ConvertLabelKey(labelsMatch[0]) + "']"
		return
	}

	switch fieldSelector.FieldPath {
	case "metadata.labels":
		fieldSelector.FieldPath = "metadata.annotations['" + LabelsAnnotation + "']"
	case "metadata.name":
		fieldSelector.FieldPath = "metadata.annotations['" + NameAnnotation + "']"
	case "metadata.namespace":
		fieldSelector.FieldPath = "metadata.annotations['" + NamespaceAnnotation + "']"
	case "metadata.uid":
		fieldSelector.FieldPath = "metadata.annotations['" + UIDAnnotation + "']"
	case "spec.serviceAccountName":
		fieldSelector.FieldPath = "metadata.annotations['" + ServiceAccountNameAnnotation + "']"
	}
}

func stripHostRewriteContainer(pPod *corev1.Pod) *corev1.Pod {
	if pPod.Annotations == nil || pPod.Annotations[HostsRewrittenAnnotation] != "true" {
		return pPod
	}

	newPod := pPod.DeepCopy()
	newInitContainerStatuses := []corev1.ContainerStatus{}
	if len(newPod.Status.InitContainerStatuses) > 0 {
		for _, v := range newPod.Status.InitContainerStatuses {
			if v.Name == HostsRewriteContainerName {
				continue
			}
			newInitContainerStatuses = append(newInitContainerStatuses, v)
		}
		newPod.Status.InitContainerStatuses = newInitContainerStatuses
	}
	return newPod
}

// TODO: refactor this to not duplicate code from translateContainerEnv
func translateEphemerealContainerEnv(c *corev1.EphemeralContainer, vPod *corev1.Pod, serviceEnvMap map[string]string) {
	envNameMap := make(map[string]struct{})
	for j, env := range c.Env {
		translateDownwardAPI(vPod, &c.Env[j])
		if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil && env.ValueFrom.ConfigMapKeyRef.Name != "" {
			c.Env[j].ValueFrom.ConfigMapKeyRef.Name = translate.PhysicalName(c.Env[j].ValueFrom.ConfigMapKeyRef.Name, vPod.Namespace)
		}
		if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil && env.ValueFrom.SecretKeyRef.Name != "" {
			c.Env[j].ValueFrom.SecretKeyRef.Name = translate.PhysicalName(c.Env[j].ValueFrom.SecretKeyRef.Name, vPod.Namespace)
		}

		envNameMap[env.Name] = struct{}{}
	}
	for j, from := range c.EnvFrom {
		if from.ConfigMapRef != nil && from.ConfigMapRef.Name != "" {
			c.EnvFrom[j].ConfigMapRef.Name = translate.PhysicalName(from.ConfigMapRef.Name, vPod.Namespace)
		}
		if from.SecretRef != nil && from.SecretRef.Name != "" {
			c.EnvFrom[j].SecretRef.Name = translate.PhysicalName(from.SecretRef.Name, vPod.Namespace)
		}
	}

	additionalEnvVars := []corev1.EnvVar{}
	for k, v := range serviceEnvMap {
		if _, exists := envNameMap[k]; !exists {
			additionalEnvVars = append(additionalEnvVars, corev1.EnvVar{Name: k, Value: v})
		}
	}

	// sort the additional env vars to avoid random ordering
	if len(additionalEnvVars) > 0 {
		sort.Slice(additionalEnvVars, func(i, j int) bool {
			return additionalEnvVars[i].Name < additionalEnvVars[j].Name
		})
	}

	if c.Env == nil {
		c.Env = []corev1.EnvVar{}
	}
	for _, e := range additionalEnvVars {
		c.Env = append(c.Env, e)
	}
}

func translateContainerEnv(c *corev1.Container, vPod *corev1.Pod, serviceEnvMap map[string]string) {
	envNameMap := make(map[string]struct{})
	for j, env := range c.Env {
		translateDownwardAPI(vPod, &c.Env[j])
		if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil && env.ValueFrom.ConfigMapKeyRef.Name != "" {
			c.Env[j].ValueFrom.ConfigMapKeyRef.Name = translate.PhysicalName(c.Env[j].ValueFrom.ConfigMapKeyRef.Name, vPod.Namespace)
		}
		if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil && env.ValueFrom.SecretKeyRef.Name != "" {
			c.Env[j].ValueFrom.SecretKeyRef.Name = translate.PhysicalName(c.Env[j].ValueFrom.SecretKeyRef.Name, vPod.Namespace)
		}

		envNameMap[env.Name] = struct{}{}
	}
	for j, from := range c.EnvFrom {
		if from.ConfigMapRef != nil && from.ConfigMapRef.Name != "" {
			c.EnvFrom[j].ConfigMapRef.Name = translate.PhysicalName(from.ConfigMapRef.Name, vPod.Namespace)
		}
		if from.SecretRef != nil && from.SecretRef.Name != "" {
			c.EnvFrom[j].SecretRef.Name = translate.PhysicalName(from.SecretRef.Name, vPod.Namespace)
		}
	}

	additionalEnvVars := []corev1.EnvVar{}
	for k, v := range serviceEnvMap {
		if _, exists := envNameMap[k]; !exists {
			additionalEnvVars = append(additionalEnvVars, corev1.EnvVar{Name: k, Value: v})
		}
	}

	// sort the additional env vars to avoid random ordering
	if len(additionalEnvVars) > 0 {
		sort.Slice(additionalEnvVars, func(i, j int) bool {
			return additionalEnvVars[i].Name < additionalEnvVars[j].Name
		})
	}

	if c.Env == nil {
		c.Env = []corev1.EnvVar{}
	}
	for _, e := range additionalEnvVars {
		c.Env = append(c.Env, e)
	}
}

func translateDownwardAPI(vPod *corev1.Pod, env *corev1.EnvVar) {
	if env.ValueFrom == nil {
		return
	}
	if env.ValueFrom.FieldRef == nil {
		return
	}
	translateFieldRef(vPod, env.ValueFrom.FieldRef)
}

func translateDNSConfig(pPod *corev1.Pod, vPod *corev1.Pod, clusterDomain, nameServer string) {
	dnsPolicy := pPod.Spec.DNSPolicy

	switch dnsPolicy {
	case corev1.DNSNone:
		return
	case corev1.DNSClusterFirstWithHostNet:
		translateDNSClusterFirstConfig(pPod, vPod, clusterDomain, nameServer)
		return
	case corev1.DNSClusterFirst:
		if !pPod.Spec.HostNetwork {
			translateDNSClusterFirstConfig(pPod, vPod, clusterDomain, nameServer)
			return
		}
		// Fallback to DNSDefault for pod on hostnetwork.
		fallthrough
	case corev1.DNSDefault:
		return
	}
}

func translateDNSClusterFirstConfig(pPod *corev1.Pod, vPod *corev1.Pod, clusterDomain, nameServer string) {
	if nameServer == "" {
		return
	}
	dnsConfig := &corev1.PodDNSConfig{
		Nameservers: []string{nameServer},
		Options: []corev1.PodDNSConfigOption{
			{
				Name:  "ndots",
				Value: pointer.StringPtr("5"),
			},
		},
	}

	if clusterDomain != "" {
		nsSvcDomain := fmt.Sprintf("%s.svc.%s", vPod.Namespace, clusterDomain)
		svcDomain := fmt.Sprintf("svc.%s", clusterDomain)
		dnsConfig.Searches = []string{nsSvcDomain, svcDomain, clusterDomain}
	}

	// don't lose existing dns config
	existingDNSConfig := pPod.Spec.DNSConfig
	if existingDNSConfig != nil {
		dnsConfig.Nameservers = deleteDuplicates(append(dnsConfig.Nameservers, existingDNSConfig.Nameservers...))
		dnsConfig.Searches = deleteDuplicates(append(dnsConfig.Searches, existingDNSConfig.Searches...))
	}

	pPod.Spec.DNSPolicy = corev1.DNSNone
	pPod.Spec.DNSConfig = dnsConfig
}

func deleteDuplicates(strs []string) []string {
	strsMap := make(map[string]bool)
	ret := []string{}
	for _, str := range strs {
		if !strsMap[str] {
			ret = append(ret, str)
			strsMap[str] = true
		}
	}
	return ret
}

func hasClusterIP(service *corev1.Service) bool {
	return service.Spec.ClusterIP != "None" && service.Spec.ClusterIP != ""
}

func translateTopologySpreadConstraints(vPod *corev1.Pod, pPod *corev1.Pod) {
	for i := range pPod.Spec.TopologySpreadConstraints {
		pPod.Spec.TopologySpreadConstraints[i].LabelSelector = translate.TranslateLabelSelector(pPod.Spec.TopologySpreadConstraints[i].LabelSelector)

		// make sure we only select pods in the current namespace
		if pPod.Spec.TopologySpreadConstraints[i].LabelSelector != nil {
			if pPod.Spec.TopologySpreadConstraints[i].LabelSelector.MatchLabels == nil {
				pPod.Spec.TopologySpreadConstraints[i].LabelSelector.MatchLabels = map[string]string{}
			}
			pPod.Spec.TopologySpreadConstraints[i].LabelSelector.MatchLabels[translate.NamespaceLabel] = translate.NamespaceLabelValue(vPod.Namespace)
		}
	}
}

func translateServicesToEnvironmentVariables(enableServiceLinks *bool, services []*corev1.Service, kubeIP string) map[string]string {
	var (
		serviceMap = make(map[string]*corev1.Service)
		retMap     = make(map[string]string)
	)

	// check if we should add services
	for i := range services {
		service := services[i]
		if !hasClusterIP(service) {
			continue
		}

		serviceName := service.Name
		if enableServiceLinks != nil && *enableServiceLinks {
			serviceMap[serviceName] = service
		}
	}

	// TODO: figure out if this is an issue because services are now not ordered anymore
	var mappedServices []*corev1.Service
	for key := range serviceMap {
		mappedServices = append(mappedServices, serviceMap[key])
	}

	// service -> env
	for _, e := range buildEnvironmentVariables(mappedServices) {
		retMap[e.Name] = e.Value
	}

	// finally add kubernetes environment variables
	for _, val := range []string{
		"KUBERNETES_PORT=tcp://IP:443",
		"KUBERNETES_PORT_443_TCP=tcp://IP:443",
		"KUBERNETES_PORT_443_TCP_ADDR=IP",
		"KUBERNETES_PORT_443_TCP_PORT=443",
		"KUBERNETES_PORT_443_TCP_PROTO=tcp",
		"KUBERNETES_SERVICE_HOST=IP",
		"KUBERNETES_SERVICE_PORT=443",
		"KUBERNETES_SERVICE_PORT_HTTPS=443",
	} {
		k, v := translate.Split(val, "=")
		retMap[k] = strings.ReplaceAll(v, "IP", kubeIP)
	}
	return retMap
}
