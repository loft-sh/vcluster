package translate

import (
	"context"
	"encoding/json"
	"fmt"
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/priorityclasses"
	"github.com/loft-sh/vcluster/pkg/serviceaccount"
	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/keyutil"
	"k8s.io/utils/pointer"
	"regexp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sort"
	"strings"
)

const (
	OwnerSetKind                  = "vcluster.loft.sh/owner-set-kind"
	NamespaceAnnotation           = "vcluster.loft.sh/namespace"
	NameAnnotation                = "vcluster.loft.sh/name"
	LabelsAnnotation              = "vcluster.loft.sh/labels"
	UIDAnnotation                 = "vcluster.loft.sh/uid"
	ServiceAccountNameAnnotation  = "vcluster.loft.sh/service-account-name"
	ServiceAccountTokenAnnotation = "vcluster.loft.sh/token-"
)

var (
	FieldPathLabelRegEx      = regexp.MustCompile("^metadata\\.labels\\['(.+)'\\]$")
	FieldPathAnnotationRegEx = regexp.MustCompile("^metadata\\.annotations\\['(.+)'\\]$")

	zero        = int64(0)
	False       = false
	maxPriority = int32(1000000000)
)

type Translator interface {
	Translate(vPod *corev1.Pod, services []*corev1.Service, dnsIP string, kubeIP string) (*corev1.Pod, error)
	Diff(vPod, pPod *corev1.Pod) (*corev1.Pod, error)
}

func NewTranslator(ctx *context2.ControllerContext) (Translator, error) {
	// create token generator
	privateKey, err := keyutil.PrivateKeyFromFile(ctx.Options.ServiceAccountKey)
	if err != nil {
		return nil, err
	}
	tokenGenerator, err := serviceaccount.JWTTokenGenerator("https://kubernetes.default.svc."+ctx.Options.ClusterDomain, privateKey)
	if err != nil {
		return nil, errors.Wrap(err, "create token generator")
	}

	imageTranslator, err := NewImageTranslator(ctx.Options.TranslateImages)
	if err != nil {
		return nil, err
	}

	return &translator{
		vClientConfig:   ctx.VirtualManager.GetConfig(),
		vClient:         ctx.VirtualManager.GetClient(),
		imageTranslator: imageTranslator,
		tokenGenerator:  tokenGenerator,

		targetNamespace:        ctx.Options.TargetNamespace,
		clusterDomain:          ctx.Options.ClusterDomain,
		serviceAccount:         ctx.Options.ServiceAccount,
		overrideHosts:          ctx.Options.OverrideHosts,
		overrideHostsImage:     ctx.Options.OverrideHostsContainerImage,
		priorityClassesEnabled: ctx.Options.EnablePriorityClasses,
	}, nil
}

type translator struct {
	vClientConfig   *rest.Config
	vClient         client.Client
	tokenGenerator  serviceaccount.TokenGenerator
	imageTranslator ImageTranslator

	targetNamespace        string
	clusterDomain          string
	serviceAccount         string
	overrideHosts          bool
	overrideHostsImage     string
	priorityClassesEnabled bool
}

func (t *translator) Diff(vPod, pPod *corev1.Pod) (*corev1.Pod, error) {
	var updatedPod *corev1.Pod
	updatedPodSpec := t.calcSpecDiff(pPod, vPod)
	if updatedPodSpec != nil {
		updatedPod = pPod.DeepCopy()
		updatedPod.Spec = *updatedPodSpec
	}

	// there are some annotations which should be excluded
	excludeAnnotations := getExcludedAnnotations(pPod)

	// check annotations
	if !translate.EqualExcept(pPod.Annotations, vPod.Annotations, excludeAnnotations...) {
		if updatedPod == nil {
			updatedPod = pPod.DeepCopy()
		}
		updatedPod.Annotations = translate.SetExcept(vPod.Annotations, pPod.Annotations, excludeAnnotations...)
	}

	// check labels annotation
	if (vPod.Labels == nil || vPod.Labels[translate.MarkerLabel] == "") && pPod.Annotations[LabelsAnnotation] != translateLabelsAnnotation(vPod) {
		if updatedPod == nil {
			updatedPod = pPod.DeepCopy()
		}
		updatedPod.Annotations[LabelsAnnotation] = translateLabelsAnnotation(vPod)
	}

	// check labels
	if !translate.LabelsEqual(vPod.Namespace, vPod.Labels, pPod.Labels) {
		if updatedPod == nil {
			updatedPod = pPod.DeepCopy()
		}
		updatedPod.Labels = translate.TranslateLabels(vPod.Namespace, vPod.Labels)
	}

	return updatedPod, nil
}

func getExcludedAnnotations(pPod *corev1.Pod) []string {
	annotations := []string{OwnerSetKind, NamespaceAnnotation, NameAnnotation, UIDAnnotation, ServiceAccountNameAnnotation, HostsRewrittenAnnotation, LabelsAnnotation}
	for _, v := range pPod.Spec.Volumes {
		if v.Projected != nil {
			for _, source := range v.Projected.Sources {
				if source.DownwardAPI != nil {
					for _, item := range source.DownwardAPI.Items {
						if item.FieldRef != nil {
							// check if its a label we have to rewrite
							annotationsMatch := FieldPathAnnotationRegEx.FindStringSubmatch(item.FieldRef.FieldPath)
							if len(annotationsMatch) == 2 {
								if strings.HasPrefix(annotationsMatch[1], ServiceAccountTokenAnnotation) {
									annotations = append(annotations, annotationsMatch[1])
								}
							}
						}
					}
				}
			}
		}
	}

	return annotations
}

// Changeable fields within the pod:
// - spec.containers[*].image
// - spec.initContainers[*].image
// - spec.activeDeadlineSeconds
//
// TODO: check for ephemereal containers
func (t *translator) calcSpecDiff(pObj, vObj *corev1.Pod) *corev1.PodSpec {
	var updatedPodSpec *corev1.PodSpec

	// active deadlines different?
	val, equal := isInt64Different(pObj.Spec.ActiveDeadlineSeconds, vObj.Spec.ActiveDeadlineSeconds)
	if !equal {
		updatedPodSpec = pObj.Spec.DeepCopy()
		updatedPodSpec.ActiveDeadlineSeconds = val
	}

	// is image different?
	updatedContainer := calcContainerImageDiff(pObj.Spec.Containers, vObj.Spec.Containers, t.imageTranslator, nil)
	if len(updatedContainer) != 0 {
		if updatedPodSpec == nil {
			updatedPodSpec = pObj.Spec.DeepCopy()
		}
		updatedPodSpec.Containers = updatedContainer
	}

	// we have to skip some init images that are injected by us to change the /etc/hosts file
	var skipContainers map[string]bool
	if pObj.Annotations != nil && pObj.Annotations[HostsRewrittenAnnotation] == "true" {
		skipContainers = map[string]bool{
			HostsRewriteContainerName: true,
		}
	}

	updatedContainer = calcContainerImageDiff(pObj.Spec.InitContainers, vObj.Spec.InitContainers, t.imageTranslator, skipContainers)
	if len(updatedContainer) != 0 {
		if updatedPodSpec == nil {
			updatedPodSpec = pObj.Spec.DeepCopy()
		}
		updatedPodSpec.InitContainers = updatedContainer
	}

	return updatedPodSpec
}

func calcContainerImageDiff(pContainers, vContainers []corev1.Container, translateImages ImageTranslator, skipContainers map[string]bool) []corev1.Container {
	newContainers := []corev1.Container{}
	changed := false
	for _, p := range pContainers {
		if skipContainers != nil && skipContainers[p.Name] {
			newContainers = append(newContainers, p)
			continue
		}

		for _, v := range vContainers {
			if p.Name == v.Name {
				if p.Image != translateImages.Translate(v.Image) {
					newContainer := *p.DeepCopy()
					newContainer.Image = translateImages.Translate(v.Image)
					newContainers = append(newContainers, newContainer)
					changed = true
				} else {
					newContainers = append(newContainers, p)
				}

				break
			}
		}
	}

	if changed == false {
		return nil
	}
	return newContainers
}

func isInt64Different(i1, i2 *int64) (*int64, bool) {
	if i1 == nil && i2 == nil {
		return nil, true
	} else if i1 != nil && i2 != nil {
		return pointer.Int64Ptr(*i2), *i1 == *i2
	}

	var updated *int64
	if i2 != nil {
		updated = pointer.Int64Ptr(*i2)
	}

	return updated, false
}

func (t *translator) Translate(vPod *corev1.Pod, services []*corev1.Service, dnsIP string, kubeIP string) (*corev1.Pod, error) {
	pPodRaw, err := translate.SetupMetadata(t.targetNamespace, vPod)
	if err != nil {
		return nil, errors.Wrap(err, "error setting metadata")
	}

	// convert to core object
	pPod := pPodRaw.(*corev1.Pod)

	// override pod fields
	pPod.Status = corev1.PodStatus{}
	pPod.Spec.DeprecatedServiceAccount = ""
	pPod.Spec.ServiceAccountName = t.serviceAccount
	pPod.Spec.AutomountServiceAccountToken = &False
	pPod.Spec.EnableServiceLinks = &False

	// check if priority classes are enabled
	if !t.priorityClassesEnabled {
		pPod.Spec.PriorityClassName = ""
		pPod.Spec.Priority = nil
	} else if pPod.Spec.PriorityClassName != "" {
		pPod.Spec.PriorityClassName = priorityclasses.TranslatePriorityClassName(pPod.Spec.PriorityClassName, pPod.Namespace)
		if pPod.Spec.Priority != nil && *pPod.Spec.Priority > maxPriority {
			pPod.Spec.Priority = &maxPriority
		}
	}

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

	// translate the dns config
	t.translateDNSConfig(pPod, vPod, dnsIP)

	// truncate hostname if needed
	if pPod.Spec.Hostname == "" {
		if len(vPod.Name) > 63 {
			pPod.Spec.Hostname = vPod.Name[0:63]
		} else {
			pPod.Spec.Hostname = vPod.Name
		}

		// Kubernetes does not support setting the hostname to a value that
		// includes a '.', therefore we need to rewrite the hostname. This is really bad
		// and wrong, but unfortunately there is currently no other solution as there is
		// no other way to change the container's hostname.
		if strings.Contains(pPod.Spec.Hostname, ".") {
			pPod.Spec.Hostname = strings.Replace(pPod.Spec.Hostname, ".", "-", -1)
		}
	}

	// if spec.subdomain is set we have to translate the /etc/hosts
	// because otherwise we could get a different hostname as if the pod
	// would be deployed in a non virtual kubernetes cluster
	if pPod.Spec.Subdomain != "" {
		if t.overrideHosts == true {
			rewritePodHostnameFQDN(pPod, t.overrideHostsImage, pPod.Spec.Hostname, pPod.Spec.Hostname, pPod.Spec.Hostname+"."+pPod.Spec.Subdomain+"."+vPod.Namespace+".svc."+t.clusterDomain)
		}

		pPod.Spec.Subdomain = ""
	}

	// translate containers
	for i := range pPod.Spec.Containers {
		translateContainerEnv(&pPod.Spec.Containers[i], vPod, serviceEnv)
		pPod.Spec.Containers[i].Image = t.imageTranslator.Translate(pPod.Spec.Containers[i].Image)
	}

	// translate init containers
	for i := range pPod.Spec.InitContainers {
		translateContainerEnv(&pPod.Spec.InitContainers[i], vPod, serviceEnv)
		pPod.Spec.InitContainers[i].Image = t.imageTranslator.Translate(pPod.Spec.InitContainers[i].Image)
	}

	// translate ephemereal containers
	for i := range pPod.Spec.EphemeralContainers {
		translateEphemerealContainerEnv(&pPod.Spec.EphemeralContainers[i], vPod, serviceEnv)
		pPod.Spec.EphemeralContainers[i].Image = t.imageTranslator.Translate(pPod.Spec.EphemeralContainers[i].Image)
	}

	// translate image pull secrets
	for i := range pPod.Spec.ImagePullSecrets {
		pPod.Spec.ImagePullSecrets[i].Name = translate.PhysicalName(pPod.Spec.ImagePullSecrets[i].Name, vPod.Namespace)
	}

	// translate volumes
	err = t.translateVolumes(pPod, vPod)
	if err != nil {
		return nil, err
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

	// translate pod affinity
	translatePodAffinity(vPod, pPod)

	return pPod, nil
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

func (t *translator) translateVolumes(pPod *corev1.Pod, vPod *corev1.Pod) error {
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
			err := t.translateProjectedVolume(pPod.Spec.Volumes[i].Projected, pPod, vPod)
			if err != nil {
				return err
			}
		}
		if pPod.Spec.Volumes[i].DownwardAPI != nil {
			for j := range pPod.Spec.Volumes[i].DownwardAPI.Items {
				translateFieldRef(pPod.Spec.Volumes[i].DownwardAPI.Items[j].FieldRef)
			}
		}
		if pPod.Spec.Volumes[i].ISCSI != nil && pPod.Spec.Volumes[i].ISCSI.SecretRef != nil {
			pPod.Spec.Volumes[i].ISCSI.SecretRef.Name = translate.PhysicalName(pPod.Spec.Volumes[i].ISCSI.SecretRef.Name, vPod.Namespace)
		}
		if pPod.Spec.Volumes[i].RBD != nil && pPod.Spec.Volumes[i].RBD.SecretRef != nil {
			pPod.Spec.Volumes[i].RBD.SecretRef.Name = translate.PhysicalName(pPod.Spec.Volumes[i].RBD.SecretRef.Name, vPod.Namespace)
		}
		if pPod.Spec.Volumes[i].FlexVolume != nil && pPod.Spec.Volumes[i].FlexVolume.SecretRef != nil {
			pPod.Spec.Volumes[i].FlexVolume.SecretRef.Name = translate.PhysicalName(pPod.Spec.Volumes[i].FlexVolume.SecretRef.Name, vPod.Namespace)
		}
		if pPod.Spec.Volumes[i].Cinder != nil && pPod.Spec.Volumes[i].Cinder.SecretRef != nil {
			pPod.Spec.Volumes[i].Cinder.SecretRef.Name = translate.PhysicalName(pPod.Spec.Volumes[i].Cinder.SecretRef.Name, vPod.Namespace)
		}
		if pPod.Spec.Volumes[i].CephFS != nil && pPod.Spec.Volumes[i].CephFS.SecretRef != nil {
			pPod.Spec.Volumes[i].CephFS.SecretRef.Name = translate.PhysicalName(pPod.Spec.Volumes[i].CephFS.SecretRef.Name, vPod.Namespace)
		}
		if pPod.Spec.Volumes[i].AzureFile != nil && pPod.Spec.Volumes[i].AzureFile.SecretName != "" {
			pPod.Spec.Volumes[i].AzureFile.SecretName = translate.PhysicalName(pPod.Spec.Volumes[i].AzureFile.SecretName, vPod.Namespace)
		}
		if pPod.Spec.Volumes[i].ScaleIO != nil && pPod.Spec.Volumes[i].ScaleIO.SecretRef != nil {
			pPod.Spec.Volumes[i].ScaleIO.SecretRef.Name = translate.PhysicalName(pPod.Spec.Volumes[i].ScaleIO.SecretRef.Name, vPod.Namespace)
		}
		if pPod.Spec.Volumes[i].StorageOS != nil && pPod.Spec.Volumes[i].StorageOS.SecretRef != nil {
			pPod.Spec.Volumes[i].StorageOS.SecretRef.Name = translate.PhysicalName(pPod.Spec.Volumes[i].StorageOS.SecretRef.Name, vPod.Namespace)
		}
		if pPod.Spec.Volumes[i].CSI != nil && pPod.Spec.Volumes[i].CSI.NodePublishSecretRef != nil {
			pPod.Spec.Volumes[i].CSI.NodePublishSecretRef.Name = translate.PhysicalName(pPod.Spec.Volumes[i].CSI.NodePublishSecretRef.Name, vPod.Namespace)
		}
	}

	return nil
}

func (t *translator) translateProjectedVolume(projectedVolume *corev1.ProjectedVolumeSource, pPod *corev1.Pod, vPod *corev1.Pod) error {
	for i := range projectedVolume.Sources {
		if projectedVolume.Sources[i].Secret != nil {
			projectedVolume.Sources[i].Secret.Name = translate.PhysicalName(projectedVolume.Sources[i].Secret.Name, vPod.Namespace)
		}
		if projectedVolume.Sources[i].ConfigMap != nil {
			projectedVolume.Sources[i].ConfigMap.Name = translate.PhysicalName(projectedVolume.Sources[i].ConfigMap.Name, vPod.Namespace)
		}
		if projectedVolume.Sources[i].DownwardAPI != nil {
			for j := range projectedVolume.Sources[i].DownwardAPI.Items {
				translateFieldRef(projectedVolume.Sources[i].DownwardAPI.Items[j].FieldRef)
			}
		}
		if projectedVolume.Sources[i].ServiceAccountToken != nil {
			serviceAccountName := "default"
			if vPod.Spec.ServiceAccountName != "" {
				serviceAccountName = vPod.Spec.ServiceAccountName
			} else if vPod.Spec.DeprecatedServiceAccount != "" {
				serviceAccountName = vPod.Spec.DeprecatedServiceAccount
			}

			// create new client
			vClient, err := kubernetes.NewForConfig(t.vClientConfig)
			if err != nil {
				return errors.Wrap(err, "create client")
			}

			audience := "https://kubernetes.default.svc." + t.clusterDomain
			if projectedVolume.Sources[i].ServiceAccountToken.Audience != "" {
				audience = projectedVolume.Sources[i].ServiceAccountToken.Audience
			}

			expirationSeconds := int64(10 * 365 * 24 * 60 * 60)
			token, err := vClient.CoreV1().ServiceAccounts(vPod.Namespace).CreateToken(context.Background(), serviceAccountName, &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					Audiences: []string{audience},
					BoundObjectRef: &authenticationv1.BoundObjectReference{
						APIVersion: corev1.SchemeGroupVersion.String(),
						Kind:       "Pod",
						Name:       vPod.Name,
						UID:        vPod.UID,
					},
					ExpirationSeconds: &expirationSeconds,
				},
			}, metav1.CreateOptions{})
			if err != nil {
				return errors.Wrap(err, "create token")
			} else if token.Status.Token == "" {
				return errors.New("received empty token")
			}

			// set the token as annotation
			if pPod.Annotations == nil {
				pPod.Annotations = map[string]string{}
			}
			var annotation string
			for {
				annotation = ServiceAccountTokenAnnotation + random.RandomString(8)
				if pPod.Annotations[annotation] == "" {
					pPod.Annotations[annotation] = token.Status.Token
					break
				}
			}

			// rewrite projected volume
			allRights := int32(0644)
			projectedVolume.Sources[i].DownwardAPI = &corev1.DownwardAPIProjection{
				Items: []corev1.DownwardAPIVolumeFile{
					{
						Path: projectedVolume.Sources[i].ServiceAccountToken.Path,
						FieldRef: &corev1.ObjectFieldSelector{
							APIVersion: "v1",
							FieldPath:  "metadata.annotations['" + annotation + "']",
						},
						Mode: &allRights,
					},
				},
			}
			projectedVolume.Sources[i].ServiceAccountToken = nil
		}
	}

	return nil
}

func translateFieldRef(fieldSelector *corev1.ObjectFieldSelector) {
	if fieldSelector == nil {
		return
	}

	// check if its a label we have to rewrite
	labelsMatch := FieldPathLabelRegEx.FindStringSubmatch(fieldSelector.FieldPath)
	if len(labelsMatch) == 2 {
		fieldSelector.FieldPath = "metadata.labels['" + translate.ConvertLabelKey(labelsMatch[1]) + "']"
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

// TODO: refactor this to not duplicate code from translateContainerEnv
func translateEphemerealContainerEnv(c *corev1.EphemeralContainer, vPod *corev1.Pod, serviceEnvMap map[string]string) {
	envNameMap := make(map[string]struct{})
	for j, env := range c.Env {
		translateDownwardAPI(&c.Env[j])
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
		translateDownwardAPI(&c.Env[j])
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

func translateDownwardAPI(env *corev1.EnvVar) {
	if env.ValueFrom == nil {
		return
	}
	if env.ValueFrom.FieldRef == nil {
		return
	}
	translateFieldRef(env.ValueFrom.FieldRef)
}

func (t *translator) translateDNSConfig(pPod *corev1.Pod, vPod *corev1.Pod, nameServer string) {
	dnsPolicy := pPod.Spec.DNSPolicy

	switch dnsPolicy {
	case corev1.DNSNone:
		return
	case corev1.DNSClusterFirstWithHostNet:
		translateDNSClusterFirstConfig(pPod, vPod, t.clusterDomain, nameServer)
		return
	case corev1.DNSClusterFirst:
		if !pPod.Spec.HostNetwork {
			translateDNSClusterFirstConfig(pPod, vPod, t.clusterDomain, nameServer)
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

func translatePodAffinity(vPod *corev1.Pod, pPod *corev1.Pod) {
	if pPod.Spec.Affinity != nil {
		if pPod.Spec.Affinity.PodAffinity != nil {
			for i, term := range pPod.Spec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
				pPod.Spec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution[i].PodAffinityTerm = translatePodAffinityTerm(vPod, term.PodAffinityTerm)
			}
			for i, term := range pPod.Spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
				pPod.Spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution[i] = translatePodAffinityTerm(vPod, term)
			}
		}
		if pPod.Spec.Affinity.PodAntiAffinity != nil {
			for i, term := range pPod.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
				pPod.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[i].PodAffinityTerm = translatePodAffinityTerm(vPod, term.PodAffinityTerm)
			}
			for i, term := range pPod.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
				pPod.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution[i] = translatePodAffinityTerm(vPod, term)
			}
		}
	}
}

func translatePodAffinityTerm(vPod *corev1.Pod, term corev1.PodAffinityTerm) corev1.PodAffinityTerm {
	// We never select pods that are not in the vcluster namespace on the host, so we will
	// omit Namespaces and namespaceSelector here
	newAffinityTerm := corev1.PodAffinityTerm{
		LabelSelector: translate.TranslateLabelSelector(term.LabelSelector),
		TopologyKey:   term.TopologyKey,
	}

	if term.LabelSelector != nil {
		if term.NamespaceSelector == nil {
			if len(term.Namespaces) > 0 {
				translatedNamespaces := []string{}
				for _, ns := range term.Namespaces {
					translatedNamespaces = append(translatedNamespaces, translate.NamespaceLabelValue(ns))
				}

				// Match specific namespaces
				if newAffinityTerm.LabelSelector.MatchExpressions == nil {
					newAffinityTerm.LabelSelector.MatchExpressions = []metav1.LabelSelectorRequirement{}
				}
				newAffinityTerm.LabelSelector.MatchExpressions = append(newAffinityTerm.LabelSelector.MatchExpressions, metav1.LabelSelectorRequirement{
					Key:      translate.NamespaceLabel,
					Operator: metav1.LabelSelectorOpIn,
					Values:   translatedNamespaces,
				})
			} else {
				// Match namespace where pod is in
				if newAffinityTerm.LabelSelector.MatchLabels == nil {
					newAffinityTerm.LabelSelector.MatchLabels = map[string]string{}
				}
				newAffinityTerm.LabelSelector.MatchLabels[translate.NamespaceLabel] = translate.NamespaceLabelValue(vPod.Namespace)
			}
		} else {
			// TODO: Support selecting namespaces by label here
			// Match all namespaces
			if newAffinityTerm.LabelSelector.MatchLabels == nil {
				newAffinityTerm.LabelSelector.MatchLabels = map[string]string{}
			}
			newAffinityTerm.LabelSelector.MatchLabels[translate.MarkerLabel] = translate.Suffix
		}
	} else {
		// Match all namespaces
		newAffinityTerm.LabelSelector = &metav1.LabelSelector{}
		newAffinityTerm.LabelSelector.MatchLabels = map[string]string{
			translate.MarkerLabel: translate.Suffix,
		}
	}

	return newAffinityTerm
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
