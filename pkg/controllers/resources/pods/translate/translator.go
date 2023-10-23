package translate

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/loft-sh/vcluster/pkg/controllers/resources/configmaps"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/priorityclasses"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/component-helpers/storage/ephemeral"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	OwnerReferences                      = "vcluster.loft.sh/owner-references"
	OwnerSetKind                         = "vcluster.loft.sh/owner-set-kind"
	NamespaceAnnotation                  = "vcluster.loft.sh/namespace"
	NameAnnotation                       = "vcluster.loft.sh/name"
	VClusterLabelsAnnotation             = "vcluster.loft.sh/labels"
	NamespaceLabelPrefix                 = "vcluster.loft.sh/ns-label"
	UIDAnnotation                        = "vcluster.loft.sh/uid"
	ClusterAutoScalerAnnotation          = "cluster-autoscaler.kubernetes.io/safe-to-evict"
	ClusterAutoScalerDaemonSetAnnotation = "cluster-autoscaler.kubernetes.io/daemonset-pod"
	ServiceAccountNameAnnotation         = "vcluster.loft.sh/service-account-name"
	ServiceAccountTokenAnnotation        = "vcluster.loft.sh/token-"
)

var (
	FieldPathLabelRegEx      = regexp.MustCompile(`^metadata\.labels\['(.+)'\]$`)
	FieldPathAnnotationRegEx = regexp.MustCompile(`^metadata\.annotations\['(.+)'\]$`)
	False                    = false
	maxPriority              = int32(1000000000)
)

type Translator interface {
	Translate(ctx context.Context, vPod *corev1.Pod, services []*corev1.Service, dnsIP string, kubeIP string) (*corev1.Pod, error)
	Diff(ctx context.Context, vPod, pPod *corev1.Pod) (*corev1.Pod, error)
}

func NewTranslator(ctx *synccontext.RegisterContext, eventRecorder record.EventRecorder) (Translator, error) {
	imageTranslator, err := NewImageTranslator(ctx.Options.TranslateImages)
	if err != nil {
		return nil, err
	}

	name := ctx.Options.Name
	if name == "" {
		name = ctx.Options.ServiceName
	}

	virtualPath := fmt.Sprintf(VirtualPathTemplate, ctx.CurrentNamespace, name)
	virtualLogsPath := path.Join(virtualPath, "log")
	virtualKubeletPath := path.Join(virtualPath, "kubelet")

	return &translator{
		vClientConfig: ctx.VirtualManager.GetConfig(),
		vClient:       ctx.VirtualManager.GetClient(),

		pClient:         ctx.PhysicalManager.GetClient(),
		imageTranslator: imageTranslator,
		eventRecorder:   eventRecorder,
		log:             loghelper.New("pods-syncer-translator"),

		defaultImageRegistry: ctx.Options.DefaultImageRegistry,

		serviceAccountSecretsEnabled: ctx.Options.ServiceAccountTokenSecrets,
		clusterDomain:                ctx.Options.ClusterDomain,
		serviceAccount:               ctx.Options.ServiceAccount,
		overrideHosts:                ctx.Options.OverrideHosts,
		overrideHostsImage:           ctx.Options.OverrideHostsContainerImage,
		serviceAccountsEnabled:       ctx.Controllers.Has("serviceaccounts"),
		priorityClassesEnabled:       ctx.Controllers.Has("priorityclasses"),
		enableScheduler:              ctx.Options.EnableScheduler,
		syncedLabels:                 ctx.Options.SyncLabels,

		mountPhysicalHostPaths:   ctx.Options.MountPhysicalHostPaths,
		hostpathMountPropagation: true,
		virtualLogsPath:          virtualLogsPath,
		virtualPodLogsPath:       filepath.Join(virtualLogsPath, "pods"),
		virtualKubeletPodPath:    filepath.Join(virtualKubeletPath, "pods"),
	}, nil
}

type translator struct {
	vClientConfig   *rest.Config
	vClient         client.Client
	pClient         client.Client
	imageTranslator ImageTranslator
	eventRecorder   record.EventRecorder
	log             loghelper.Logger

	defaultImageRegistry string

	serviceAccountsEnabled       bool
	serviceAccountSecretsEnabled bool
	clusterDomain                string
	serviceAccount               string
	overrideHosts                bool
	overrideHostsImage           string
	priorityClassesEnabled       bool
	enableScheduler              bool
	syncedLabels                 []string

	mountPhysicalHostPaths   bool
	hostpathMountPropagation bool
	virtualLogsPath          string
	virtualPodLogsPath       string
	virtualKubeletPodPath    string
}

func (t *translator) Translate(ctx context.Context, vPod *corev1.Pod, services []*corev1.Service, dnsIP string, kubeIP string) (*corev1.Pod, error) {
	// get Namespace resource in order to have access to its labels
	vNamespace := &corev1.Namespace{}
	err := t.vClient.Get(ctx, client.ObjectKey{Name: vPod.ObjectMeta.GetNamespace()}, vNamespace)
	if err != nil {
		return nil, err
	}

	// convert to core object
	pPod := translate.Default.ApplyMetadata(vPod, t.syncedLabels).(*corev1.Pod)

	// override pod fields
	pPod.Status = corev1.PodStatus{}
	pPod.Spec.DeprecatedServiceAccount = ""
	pPod.Spec.ServiceAccountName = t.serviceAccount
	if t.serviceAccountsEnabled {
		vPodServiceAccountName := vPod.Spec.DeprecatedServiceAccount
		if vPod.Spec.ServiceAccountName != "" {
			vPodServiceAccountName = vPod.Spec.ServiceAccountName
		}
		if vPodServiceAccountName != "" {
			pPod.Spec.ServiceAccountName = translate.Default.PhysicalName(vPodServiceAccountName, vPod.Namespace)
		}
	}

	pPod.Spec.AutomountServiceAccountToken = &False
	pPod.Spec.EnableServiceLinks = &False

	// check if priority classes are enabled
	if !t.priorityClassesEnabled {
		pPod.Spec.PriorityClassName = ""
		pPod.Spec.Priority = nil
	} else if pPod.Spec.PriorityClassName != "" {
		pPod.Spec.PriorityClassName = priorityclasses.NewPriorityClassTranslator()(pPod.Spec.PriorityClassName, nil)
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
		if _, ok := vPod.Annotations[VClusterLabelsAnnotation]; ok {
			pPod.Annotations[VClusterLabelsAnnotation] = vPod.Annotations[VClusterLabelsAnnotation]
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
	if _, ok := pPod.Annotations[VClusterLabelsAnnotation]; !ok {
		pPod.Annotations[VClusterLabelsAnnotation] = LabelsAnnotation(vPod)
	}
	if _, ok := pPod.Annotations[ClusterAutoScalerAnnotation]; !ok {
		// check if the vPod would be evictable
		controller := metav1.GetControllerOf(vPod)
		isEvictable := controller != nil
		pPod.Annotations[ClusterAutoScalerAnnotation] = strconv.FormatBool(isEvictable)
		if controller != nil && controller.Kind == "DaemonSet" {
			pPod.Annotations[ClusterAutoScalerDaemonSetAnnotation] = "true"
		}
	}
	if _, ok := pPod.Annotations[OwnerReferences]; !ok && len(vPod.OwnerReferences) > 0 {
		ownerReferencesData, err := json.Marshal(vPod.OwnerReferences)
		if err == nil {
			if pPod.Annotations == nil {
				pPod.Annotations = map[string]string{}
			}

			pPod.Annotations[OwnerReferences] = string(ownerReferencesData)
			for _, ownerReference := range vPod.OwnerReferences {
				if ownerReference.APIVersion == appsv1.SchemeGroupVersion.String() && canAnnotateOwnerSetKind(ownerReference.Kind) {
					pPod.Annotations[OwnerSetKind] = ownerReference.Kind
					break
				}
			}
		}
	}

	// Add Namespace labels
	updatedLabels := pPod.GetLabels()
	if updatedLabels == nil {
		updatedLabels = map[string]string{}
	}
	for k, v := range vNamespace.GetLabels() {
		updatedLabels[translate.ConvertLabelKeyWithPrefix(NamespaceLabelPrefix, k)] = v
	}
	pPod.SetLabels(updatedLabels)

	// translate services to environment variables
	serviceEnv := ServicesToEnvironmentVariables(vPod.Spec.EnableServiceLinks, services, kubeIP)

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
		pPod.Spec.Hostname = strings.TrimSuffix(strings.Replace(pPod.Spec.Hostname, ".", "-", -1), "-")
	}

	// if spec.subdomain is set we have to translate the /etc/hosts
	// because otherwise we could get a different hostname as if the pod
	// would be deployed in a non virtual kubernetes cluster
	if pPod.Spec.Subdomain != "" {
		if t.overrideHosts {
			rewritePodHostnameFQDN(pPod, t.defaultImageRegistry, t.overrideHostsImage, pPod.Spec.Hostname, pPod.Spec.Hostname, pPod.Spec.Hostname+"."+pPod.Spec.Subdomain+"."+vPod.Namespace+".svc."+t.clusterDomain)
		}

		pPod.Spec.Subdomain = ""
	}

	// translate containers
	for i := range pPod.Spec.Containers {
		envVar, envFrom := ContainerEnv(pPod.Spec.Containers[i].Env, pPod.Spec.Containers[i].EnvFrom, vPod, serviceEnv)
		pPod.Spec.Containers[i].Env = envVar
		pPod.Spec.Containers[i].EnvFrom = envFrom
		pPod.Spec.Containers[i].Image = t.imageTranslator.Translate(pPod.Spec.Containers[i].Image)
	}

	// translate init containers
	for i := range pPod.Spec.InitContainers {
		envVar, envFrom := ContainerEnv(pPod.Spec.InitContainers[i].Env, pPod.Spec.InitContainers[i].EnvFrom, vPod, serviceEnv)
		pPod.Spec.InitContainers[i].Env = envVar
		pPod.Spec.InitContainers[i].EnvFrom = envFrom
		pPod.Spec.InitContainers[i].Image = t.imageTranslator.Translate(pPod.Spec.InitContainers[i].Image)
	}

	// translate ephemeral containers
	for i := range pPod.Spec.EphemeralContainers {
		envVar, envFrom := ContainerEnv(pPod.Spec.EphemeralContainers[i].Env, pPod.Spec.EphemeralContainers[i].EnvFrom, vPod, serviceEnv)
		pPod.Spec.EphemeralContainers[i].Env = envVar
		pPod.Spec.EphemeralContainers[i].EnvFrom = envFrom
		pPod.Spec.EphemeralContainers[i].Image = t.imageTranslator.Translate(pPod.Spec.EphemeralContainers[i].Image)
	}

	// translate image pull secrets
	for i := range pPod.Spec.ImagePullSecrets {
		pPod.Spec.ImagePullSecrets[i].Name = translate.Default.PhysicalName(pPod.Spec.ImagePullSecrets[i].Name, vPod.Namespace)
	}

	// translate volumes
	err = t.translateVolumes(ctx, pPod, vPod)
	if err != nil {
		return nil, err
	}

	// translate topology spread constraints
	if t.enableScheduler {
		pPod.Spec.TopologySpreadConstraints = nil
		pPod.Spec.Affinity = nil
		pPod.Spec.NodeSelector = nil
	} else {
		// translate topology spread constraints
		if translate.Default.SingleNamespaceTarget() {
			translateTopologySpreadConstraints(vPod, pPod)
		}

		// translate pod affinity
		t.translatePodAffinity(vPod, pPod)

		// translate node selector
		for k, v := range vPod.Spec.NodeSelector {
			if pPod.Spec.NodeSelector == nil {
				pPod.Spec.NodeSelector = map[string]string{}
			}
			pPod.Spec.NodeSelector[k] = v
		}
	}

	return pPod, nil
}

func canAnnotateOwnerSetKind(kind string) bool {
	return kind == "DaemonSet" || kind == "Job" || kind == "ReplicaSet" || kind == "StatefulSet"
}

func LabelsAnnotation(obj client.Object) string {
	labelsString := []string{}
	for k, v := range obj.GetLabels() {
		// escape pod labels
		out, err := json.Marshal(v)
		if err != nil {
			continue
		}

		labelsString = append(labelsString, k+"="+string(out))
	}

	sort.Strings(labelsString)
	return strings.Join(labelsString, "\n")
}

func translateVolumes(pPod, vPod *corev1.Pod) {
    for i, volume := range pPod.Spec.Volumes {
        translateConfigMap(&volume, &vPod.Namespace)
        translateSecret(&volume, &vPod.Namespace)
        translatePVC(&volume, &vPod.Namespace)
        translateEphemeral(&volume, &vPod, &vPod.Spec.Volumes[i])
        if volume.Projected != nil {
            translateProjectedVolume(&pPod.Spec.Volumes[i], vPod)
        }
        if volume.DownwardAPI != nil {
            translateDownwardAPI(&pPod.Spec.Volumes[i])
        }
        // Handle other volume types
    }
}

func translateConfigMap(volume *corev1.Volume, namespace *string) {
    if volume.ConfigMap != nil {
        volume.ConfigMap.Name = translateName(*namespace, volume.ConfigMap.Name)
    }
}

func translateSecret(volume *corev1.Volume, namespace *string) {
    if volume.Secret != nil {
        volume.Secret.SecretName = translateName(*namespace, volume.Secret.SecretName)
    }
}

func translatePVC(volume *corev1.Volume, namespace *string) {
    if volume.PersistentVolumeClaim != nil {
        volume.PersistentVolumeClaim.ClaimName = translateName(*namespace, volume.PersistentVolumeClaim.ClaimName)
    }
}

func translateEphemeral(volume *corev1.Volume, vPod *corev1.Pod, vPodVolume *corev1.Volume) {
    if volume.Ephemeral != nil {
        // Translate ephemeral volume to use PVC reference
        volume.PersistentVolumeClaim = &corev1.PersistentVolumeClaimVolumeSource{
            ClaimName: translateName(*vPod.Namespace, ephemeral.VolumeClaimName(vPod, vPodVolume)),
        }
        volume.Ephemeral = nil
    }
}

func translateProjectedVolume(volume *corev1.Volume, vPod *corev1.Pod) {
    for i, source := range volume.Projected.Sources {
        translateConfigMap(&source.ConfigMap, &vPod.Namespace)
        translateSecret(&source.Secret, &vPod.Namespace)
        translateDownwardAPI(&source.DownwardAPI)
        // Handle other source types
    }
}

func translateDownwardAPI(source *corev1.DownwardAPIProjection) {
    if source != nil {
        for j, item := range source.Items {
            translateFieldRef(&source.Items[j].FieldRef)
        }
    }
}

func translateFieldRef(fieldSelector *corev1.ObjectFieldSelector) {
    if fieldSelector != nil {
        // Translate field references if needed
    }
}

func translateName(namespace, name string) string {
    return translate.Default.PhysicalName(name, namespace)
}
