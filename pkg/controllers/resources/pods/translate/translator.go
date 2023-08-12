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
	"k8s.io/apimachinery/pkg/api/equality"
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
	OwnerSetKind                         = "vcluster.loft.sh/owner-set-kind"
	NamespaceAnnotation                  = "vcluster.loft.sh/namespace"
	NameAnnotation                       = "vcluster.loft.sh/name"
	LabelsAnnotation                     = "vcluster.loft.sh/labels"
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

		projectedVolumeSAToken: make(map[string]string),
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

	projectedVolumeSAToken map[string]string
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
		pPod.Annotations[LabelsAnnotation] = TranslateLabelsAnnotation(vPod)
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
	serviceEnv := TranslateServicesToEnvironmentVariables(vPod.Spec.EnableServiceLinks, services, kubeIP)

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
		envVar, envFrom := TranslateContainerEnv(pPod.Spec.Containers[i].Env, pPod.Spec.Containers[i].EnvFrom, vPod, serviceEnv)
		pPod.Spec.Containers[i].Env = envVar
		pPod.Spec.Containers[i].EnvFrom = envFrom
		pPod.Spec.Containers[i].Image = t.imageTranslator.Translate(pPod.Spec.Containers[i].Image)
	}

	// translate init containers
	for i := range pPod.Spec.InitContainers {
		envVar, envFrom := TranslateContainerEnv(pPod.Spec.InitContainers[i].Env, pPod.Spec.InitContainers[i].EnvFrom, vPod, serviceEnv)
		pPod.Spec.InitContainers[i].Env = envVar
		pPod.Spec.InitContainers[i].EnvFrom = envFrom
		pPod.Spec.InitContainers[i].Image = t.imageTranslator.Translate(pPod.Spec.InitContainers[i].Image)
	}

	// translate ephemeral containers
	for i := range pPod.Spec.EphemeralContainers {
		envVar, envFrom := TranslateContainerEnv(pPod.Spec.EphemeralContainers[i].Env, pPod.Spec.EphemeralContainers[i].EnvFrom, vPod, serviceEnv)
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

	// add an owner-set-kind annotation to each pod with an owner
	for _, ownerReference := range vPod.OwnerReferences {
		if ownerReference.APIVersion == appsv1.SchemeGroupVersion.String() && canAnnotateOwnerSetKind(ownerReference.Kind) {
			if pPod.Annotations == nil {
				pPod.Annotations = map[string]string{}
			}

			pPod.Annotations[OwnerSetKind] = ownerReference.Kind
			break
		}
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

func TranslateLabelsAnnotation(obj client.Object) string {
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

func (t *translator) translateVolumes(ctx context.Context, pPod *corev1.Pod, vPod *corev1.Pod) error {
	var shouldCreateTokenSecret bool

	for i := range pPod.Spec.Volumes {
		if pPod.Spec.Volumes[i].ConfigMap != nil {
			pPod.Spec.Volumes[i].ConfigMap.Name = configmaps.ConfigMapNameTranslator(types.NamespacedName{Name: pPod.Spec.Volumes[i].ConfigMap.Name, Namespace: vPod.Namespace}, nil)
		}
		if pPod.Spec.Volumes[i].Secret != nil {
			pPod.Spec.Volumes[i].Secret.SecretName = translate.Default.PhysicalName(pPod.Spec.Volumes[i].Secret.SecretName, vPod.Namespace)
		}
		if pPod.Spec.Volumes[i].PersistentVolumeClaim != nil {
			pPod.Spec.Volumes[i].PersistentVolumeClaim.ClaimName = translate.Default.PhysicalName(pPod.Spec.Volumes[i].PersistentVolumeClaim.ClaimName, vPod.Namespace)
		}
		if pPod.Spec.Volumes[i].Ephemeral != nil {
			// An ephemeral volume is created as a PVC by the "ephemeral_volume" K8s controller.
			// We shall replace this volume type with a PVC reference to avoid PVC duplication
			// which would occur if both vcluster and host kube controllers would create the PVC
			// based on the pPod.Spec.Volumes[i].Ephemeral.VolumeClaimTemplate
			// What makes this volume ephemeral is an ownerReference set on PVC, which references
			// the Pod, and that remains unchanged and thus the PVC will be removed by the kube
			// controllers of the vcluster, and syncer will then remove the PVC from the host.
			pPod.Spec.Volumes[i].PersistentVolumeClaim = &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: translate.Default.PhysicalName(ephemeral.VolumeClaimName(vPod, &vPod.Spec.Volumes[i]), vPod.Namespace),
			}
			pPod.Spec.Volumes[i].Ephemeral = nil
		}
		if pPod.Spec.Volumes[i].Projected != nil {
			err := t.translateProjectedVolume(ctx, pPod.Spec.Volumes[i].Projected, pPod.Spec.Volumes[i].Name, pPod, vPod, &shouldCreateTokenSecret)
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
			pPod.Spec.Volumes[i].ISCSI.SecretRef.Name = translate.Default.PhysicalName(pPod.Spec.Volumes[i].ISCSI.SecretRef.Name, vPod.Namespace)
		}
		if pPod.Spec.Volumes[i].RBD != nil && pPod.Spec.Volumes[i].RBD.SecretRef != nil {
			pPod.Spec.Volumes[i].RBD.SecretRef.Name = translate.Default.PhysicalName(pPod.Spec.Volumes[i].RBD.SecretRef.Name, vPod.Namespace)
		}
		if pPod.Spec.Volumes[i].FlexVolume != nil && pPod.Spec.Volumes[i].FlexVolume.SecretRef != nil {
			pPod.Spec.Volumes[i].FlexVolume.SecretRef.Name = translate.Default.PhysicalName(pPod.Spec.Volumes[i].FlexVolume.SecretRef.Name, vPod.Namespace)
		}
		if pPod.Spec.Volumes[i].Cinder != nil && pPod.Spec.Volumes[i].Cinder.SecretRef != nil {
			pPod.Spec.Volumes[i].Cinder.SecretRef.Name = translate.Default.PhysicalName(pPod.Spec.Volumes[i].Cinder.SecretRef.Name, vPod.Namespace)
		}
		if pPod.Spec.Volumes[i].CephFS != nil && pPod.Spec.Volumes[i].CephFS.SecretRef != nil {
			pPod.Spec.Volumes[i].CephFS.SecretRef.Name = translate.Default.PhysicalName(pPod.Spec.Volumes[i].CephFS.SecretRef.Name, vPod.Namespace)
		}
		if pPod.Spec.Volumes[i].AzureFile != nil && pPod.Spec.Volumes[i].AzureFile.SecretName != "" {
			pPod.Spec.Volumes[i].AzureFile.SecretName = translate.Default.PhysicalName(pPod.Spec.Volumes[i].AzureFile.SecretName, vPod.Namespace)
		}
		if pPod.Spec.Volumes[i].ScaleIO != nil && pPod.Spec.Volumes[i].ScaleIO.SecretRef != nil {
			pPod.Spec.Volumes[i].ScaleIO.SecretRef.Name = translate.Default.PhysicalName(pPod.Spec.Volumes[i].ScaleIO.SecretRef.Name, vPod.Namespace)
		}
		if pPod.Spec.Volumes[i].StorageOS != nil && pPod.Spec.Volumes[i].StorageOS.SecretRef != nil {
			pPod.Spec.Volumes[i].StorageOS.SecretRef.Name = translate.Default.PhysicalName(pPod.Spec.Volumes[i].StorageOS.SecretRef.Name, vPod.Namespace)
		}
		if pPod.Spec.Volumes[i].CSI != nil && pPod.Spec.Volumes[i].CSI.NodePublishSecretRef != nil {
			pPod.Spec.Volumes[i].CSI.NodePublishSecretRef.Name = translate.Default.PhysicalName(pPod.Spec.Volumes[i].CSI.NodePublishSecretRef.Name, vPod.Namespace)
		}
		if pPod.Spec.Volumes[i].Glusterfs != nil && pPod.Spec.Volumes[i].Glusterfs.EndpointsName != "" {
			pPod.Spec.Volumes[i].Glusterfs.EndpointsName = translate.Default.PhysicalName(pPod.Spec.Volumes[i].Glusterfs.EndpointsName, vPod.Namespace)
		}
	}

	if shouldCreateTokenSecret {
		// create the service account token holder secret
		err := SATokenSecret(ctx, t.pClient, vPod, t.projectedVolumeSAToken)
		if err != nil {
			return nil
		}
	}

	// rewrite host paths if enabled
	if t.mountPhysicalHostPaths || t.hostpathMountPropagation {
		t.rewriteHostPaths(pPod)
	}

	return nil
}

func (t *translator) translateProjectedVolume(ctx context.Context, projectedVolume *corev1.ProjectedVolumeSource, volumeName string, pPod *corev1.Pod, vPod *corev1.Pod, shouldCreateTokenSecret *bool) error {
	for i := range projectedVolume.Sources {
		if projectedVolume.Sources[i].Secret != nil {
			projectedVolume.Sources[i].Secret.Name = translate.Default.PhysicalName(projectedVolume.Sources[i].Secret.Name, vPod.Namespace)
		}
		if projectedVolume.Sources[i].ConfigMap != nil {
			projectedVolume.Sources[i].ConfigMap.Name = translate.Default.PhysicalName(projectedVolume.Sources[i].ConfigMap.Name, vPod.Namespace)
			if projectedVolume.Sources[i].ConfigMap.Name == "kube-root-ca.crt" {
				projectedVolume.Sources[i].ConfigMap.Name = translate.SafeConcatName("vcluster", "kube-root-ca.crt", "x", translate.Suffix)
			}
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

			audiences := []string{"https://kubernetes.default.svc." + t.clusterDomain, "https://kubernetes.default.svc", "https://kubernetes.default"}
			if projectedVolume.Sources[i].ServiceAccountToken.Audience != "" {
				audiences = []string{projectedVolume.Sources[i].ServiceAccountToken.Audience}
			}

			expirationSeconds := int64(10 * 365 * 24 * 60 * 60)
			token, err := vClient.CoreV1().ServiceAccounts(vPod.Namespace).CreateToken(ctx, serviceAccountName, &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					Audiences: audiences,
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

			// rewrite projected volume
			allRights := int32(0644)

			if t.serviceAccountSecretsEnabled {
				// populate service account map
				t.projectedVolumeSAToken[volumeName] = token.Status.Token

				// rewrite projected volume to use sources as secret
				projectedVolume.Sources[i].Secret = &corev1.SecretProjection{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: SecretNameFromPodName(vPod.Name, vPod.Namespace),
					},
					Items: []corev1.KeyToPath{
						{
							Key:  volumeName,
							Path: projectedVolume.Sources[i].ServiceAccountToken.Path,
							Mode: &allRights,
						},
					},
				}

				*shouldCreateTokenSecret = true
			} else {
				// set annotation on physical pod
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

				// rewrite projected volume to use DownwardAPI as source
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
		fieldSelector.FieldPath = "metadata.labels['" + translate.Default.ConvertLabelKey(labelsMatch[1]) + "']"
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

func TranslateContainerEnv(envVar []corev1.EnvVar, envFrom []corev1.EnvFromSource, vPod *corev1.Pod, serviceEnvMap map[string]string) ([]corev1.EnvVar, []corev1.EnvFromSource) {
	envNameMap := make(map[string]struct{})
	for j, env := range envVar {
		translateDownwardAPI(&envVar[j])
		if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil && env.ValueFrom.ConfigMapKeyRef.Name != "" {
			envVar[j].ValueFrom.ConfigMapKeyRef.Name = translate.Default.PhysicalName(envVar[j].ValueFrom.ConfigMapKeyRef.Name, vPod.Namespace)
		}
		if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil && env.ValueFrom.SecretKeyRef.Name != "" {
			envVar[j].ValueFrom.SecretKeyRef.Name = translate.Default.PhysicalName(envVar[j].ValueFrom.SecretKeyRef.Name, vPod.Namespace)
		}

		envNameMap[env.Name] = struct{}{}
	}
	for j, from := range envFrom {
		if from.ConfigMapRef != nil && from.ConfigMapRef.Name != "" {
			envFrom[j].ConfigMapRef.Name = translate.Default.PhysicalName(from.ConfigMapRef.Name, vPod.Namespace)
		}
		if from.SecretRef != nil && from.SecretRef.Name != "" {
			envFrom[j].SecretRef.Name = translate.Default.PhysicalName(from.SecretRef.Name, vPod.Namespace)
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

	if envVar == nil {
		envVar = []corev1.EnvVar{}
	}
	// additional env vars should come first to allow for dependent environment variables
	envVar = append(additionalEnvVars, envVar...)
	return envVar, envFrom
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
				Value: pointer.String("5"),
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

func (t *translator) translatePodAffinity(vPod *corev1.Pod, pPod *corev1.Pod) {
	if pPod.Spec.Affinity != nil {
		if pPod.Spec.Affinity.PodAffinity != nil {
			for i, term := range pPod.Spec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
				pPod.Spec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution[i].PodAffinityTerm = t.translatePodAffinityTerm(vPod, term.PodAffinityTerm)
			}
			for i, term := range pPod.Spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
				pPod.Spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution[i] = t.translatePodAffinityTerm(vPod, term)
			}
		}
		if pPod.Spec.Affinity.PodAntiAffinity != nil {
			for i, term := range pPod.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
				pPod.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[i].PodAffinityTerm = t.translatePodAffinityTerm(vPod, term.PodAffinityTerm)
			}
			for i, term := range pPod.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
				pPod.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution[i] = t.translatePodAffinityTerm(vPod, term)
			}
		}
	}
}

func (t *translator) translatePodAffinityTerm(vPod *corev1.Pod, term corev1.PodAffinityTerm) corev1.PodAffinityTerm {
	// TODO(Multi-Namespace): Add multi-namespace support for this - condition below might be enough
	if !translate.Default.SingleNamespaceTarget() {
		return term
	}

	// We never select pods that are not in the vcluster namespace on the host, so we will
	// omit Namespaces and namespaceSelector here
	newAffinityTerm := corev1.PodAffinityTerm{
		LabelSelector: translate.Default.TranslateLabelSelector(term.LabelSelector),
		TopologyKey:   term.TopologyKey,
	}

	// execute logic only if LabelSelector can match something (nil doesn't match anything)
	if term.LabelSelector != nil {
		if len(term.Namespaces) > 0 {
			// handle a case where .namespaces is used together with the .namespaceSelector
			if term.NamespaceSelector != nil && len(term.NamespaceSelector.MatchLabels)+len(term.NamespaceSelector.MatchExpressions) != 0 {
				// TODO: implement support for using .namespaces together with the .namespaceSelector
				// this can't be done with simple a translation to label selectors because .namespaces
				// and matches of the .namespaceSelector are combined as a union(logical OR),
				// but label selectors are combined with logical AND.
				// Proposed solution is to add a label, per each affinity term that uses .labelselector together with .namespaces,
				// to each matching pod during pod reconcile and translate .labelselector + .namespaces to a label
				// selector with the value that is unique for the particular affinity term

				// Create and event and log entry until the above is implemented
				t.eventRecorder.Eventf(vPod, "Warning", "SyncWarning", "Inter-pod affinity rule(s) that use both .namespaces and .namespaceSelector fields in the same term are not supported by vcluster yet. The .namespaceSelector fields of the unsupported affinity entries will be ignored.")
				t.log.Infof("Inter-pod affinity rule(s) that use both .namespaces and .namespaceSelector fields in the same term are not supported by vcluster yet. The .namespaceSelector fields of the unsupported affinity entries of the %s pod in %s namespace will be ignored.", vPod.GetName(), vPod.GetNamespace())
			}

			// create a selector for .namespaces only if .namespaceSelector is nil or not empty
			// if .namespaceSelector is empty then it matches all namespaces and we can ignore .namespaces
			if term.NamespaceSelector == nil || len(term.NamespaceSelector.MatchLabels)+len(term.NamespaceSelector.MatchExpressions) != 0 {
				// Match specific namespaces
				if newAffinityTerm.LabelSelector.MatchExpressions == nil {
					newAffinityTerm.LabelSelector.MatchExpressions = []metav1.LabelSelectorRequirement{}
				}
				newAffinityTerm.LabelSelector.MatchExpressions = append(newAffinityTerm.LabelSelector.MatchExpressions, metav1.LabelSelectorRequirement{
					Key:      translate.NamespaceLabel,
					Operator: metav1.LabelSelectorOpIn,
					Values:   append([]string{}, term.Namespaces...),
				})
			}
		} else if term.NamespaceSelector != nil {
			// translate namespace label selector
			newAffinityTerm.LabelSelector = translate.TranslateLabelSelectorWithPrefix(NamespaceLabelPrefix, term.NamespaceSelector)
		} else {
			// Match namespace where pod is in
			// k8s docs: "a null or empty namespaces list and null namespaceSelector means "this pod's namespace""
			if newAffinityTerm.LabelSelector.MatchLabels == nil {
				newAffinityTerm.LabelSelector.MatchLabels = map[string]string{}
			}
			newAffinityTerm.LabelSelector.MatchLabels[translate.NamespaceLabel] = vPod.Namespace
		}

		if newAffinityTerm.LabelSelector.MatchLabels == nil {
			newAffinityTerm.LabelSelector.MatchLabels = map[string]string{}
		}
		newAffinityTerm.LabelSelector.MatchLabels[translate.MarkerLabel] = translate.Suffix
	}
	return newAffinityTerm
}

func translateTopologySpreadConstraints(vPod *corev1.Pod, pPod *corev1.Pod) {
	for i := range pPod.Spec.TopologySpreadConstraints {
		pPod.Spec.TopologySpreadConstraints[i].LabelSelector = translate.Default.TranslateLabelSelector(pPod.Spec.TopologySpreadConstraints[i].LabelSelector)

		// make sure we only select pods in the current namespace
		if pPod.Spec.TopologySpreadConstraints[i].LabelSelector != nil {
			if pPod.Spec.TopologySpreadConstraints[i].LabelSelector.MatchLabels == nil {
				pPod.Spec.TopologySpreadConstraints[i].LabelSelector.MatchLabels = map[string]string{}
			}
			pPod.Spec.TopologySpreadConstraints[i].LabelSelector.MatchLabels[translate.NamespaceLabel] = vPod.Namespace
			pPod.Spec.TopologySpreadConstraints[i].LabelSelector.MatchLabels[translate.MarkerLabel] = translate.Suffix
		}
	}
}

func TranslateServicesToEnvironmentVariables(enableServiceLinks *bool, services []*corev1.Service, kubeIP string) map[string]string {
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

func (t *translator) Diff(ctx context.Context, vPod, pPod *corev1.Pod) (*corev1.Pod, error) {
	// get Namespace resource in order to have access to its labels
	vNamespace := &corev1.Namespace{}
	err := t.vClient.Get(ctx, client.ObjectKey{Name: vPod.ObjectMeta.GetNamespace()}, vNamespace)
	if err != nil {
		return nil, err
	}

	var updatedPod *corev1.Pod
	updatedPodSpec := t.calcSpecDiff(pPod, vPod)
	if updatedPodSpec != nil {
		updatedPod = pPod.DeepCopy()
		updatedPod.Spec = *updatedPodSpec
	}

	// check annotations
	_, updatedAnnotations, updatedLabels := translate.Default.ApplyMetadataUpdate(vPod, pPod, t.syncedLabels, getExcludedAnnotations(pPod)...)
	if updatedAnnotations == nil {
		updatedAnnotations = map[string]string{}
	}
	if updatedLabels == nil {
		updatedLabels = map[string]string{}
	}

	updatedAnnotations[LabelsAnnotation] = TranslateLabelsAnnotation(vPod)
	if !equality.Semantic.DeepEqual(updatedAnnotations, pPod.Annotations) {
		if updatedPod == nil {
			updatedPod = pPod.DeepCopy()
		}
		updatedPod.Annotations = updatedAnnotations
	}

	// check pod and namespace labels
	for k, v := range vNamespace.GetLabels() {
		updatedLabels[translate.ConvertLabelKeyWithPrefix(NamespaceLabelPrefix, k)] = v
	}
	if !equality.Semantic.DeepEqual(updatedLabels, pPod.Labels) {
		if updatedPod == nil {
			updatedPod = pPod.DeepCopy()
		}
		updatedPod.Labels = updatedLabels
	}

	return updatedPod, nil
}

func getExcludedAnnotations(pPod *corev1.Pod) []string {
	annotations := []string{ClusterAutoScalerAnnotation, OwnerSetKind, NamespaceAnnotation, NameAnnotation, UIDAnnotation, ServiceAccountNameAnnotation, HostsRewrittenAnnotation, LabelsAnnotation}
	if pPod != nil {
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

	isEqual := isPodSpecSchedulingGatesDiff(pObj.Spec.SchedulingGates, vObj.Spec.SchedulingGates)
	if !isEqual {
		if updatedPodSpec == nil {
			updatedPodSpec = pObj.Spec.DeepCopy()
		}
		updatedPodSpec.SchedulingGates = vObj.Spec.SchedulingGates
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

	if !changed {
		return nil
	}
	return newContainers
}

func isInt64Different(i1, i2 *int64) (*int64, bool) {
	if i1 == nil && i2 == nil {
		return nil, true
	} else if i1 != nil && i2 != nil {
		return pointer.Int64(*i2), *i1 == *i2
	}

	var updated *int64
	if i2 != nil {
		updated = pointer.Int64(*i2)
	}

	return updated, false
}

func isPodSpecSchedulingGatesDiff(pGates, vGates []corev1.PodSchedulingGate) bool {
	if len(vGates) != len(pGates) {
		return false
	}
	for i, v := range vGates {
		if v.Name != pGates[i].Name {
			return false
		}
	}
	return true
}
