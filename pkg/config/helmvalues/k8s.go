package helmvalues

import corev1 "k8s.io/api/core/v1"

type K8s struct {
	BaseHelm
	Syncer     K8sSyncerValues  `json:"syncer,omitempty"`
	API        APIServerValues  `json:"api,omitempty"`
	Controller ControllerValues `json:"controller,omitempty"`
	Scheduler  SchedulerValues  `json:"scheduler,omitempty"`
	Etcd       EtcdValues       `json:"etcd,omitempty"`
}

type K8sSyncerValues struct {
	SyncerValues
	CommonValues
	SecurityContext    corev1.SecurityContext    `json:"securityContext,omitempty"`
	PodSecurityContext corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`
}

type APIServerValues struct {
	CommonValues
	SyncerExORCommonValues
	ControlPlaneCommonValues
	SecurityContext    corev1.SecurityContext `json:"securityContext,omitempty"`
	ServiceAnnotations map[string]string      `json:"serviceAnnotations,omitempty"`
}

type ControllerValues struct {
	CommonValues
	SyncerExORCommonValues
	ControlPlaneCommonValues
	SecurityContext corev1.SecurityContext `json:"securityContext,omitempty"`
}

type SchedulerValues struct {
	CommonValues
	SyncerExORCommonValues
	ControlPlaneCommonValues
	Disabled bool `json:"disabled,omitempty"`
}

type EtcdValues struct {
	// Disabled is allowed for k8s & eks
	Disabled bool `json:"disabled,omitempty"`
	CommonValues
	SyncerExORCommonValues
	ControlPlaneCommonValues
	Storage struct {
		Persistence bool   `json:"persistence,omitempty"`
		Size        string `json:"size,omitempty"`
	} `json:"storage,omitempty"`
	SecurityContext                  corev1.SecurityContext `json:"securityContext,omitempty"`
	ServiceAnnotations               map[string]string      `json:"serviceAnnotations,omitempty"`
	AutoDeletePersistentVolumeClaims bool                   `json:"autoDeletePersistentVolumeClaims,omitempty"`
}

type ControlPlaneCommonValues struct {
	Image           string            `json:"image,omitempty"`
	ImagePullPolicy string            `json:"imagePullPolicy,omitempty"`
	Replicas        uint32            `json:"replicas,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
	Annotations     map[string]string `json:"annotations,omitempty"`
}

type SyncerExORCommonValues struct {
	VolumeMounts []corev1.VolumeMount        `json:"volumeMounts,omitempty"`
	ExtraArgs    []string                    `json:"extraArgs,omitempty"`
	Resources    corev1.ResourceRequirements `json:"resources,omitempty"`
}

type CommonValues struct {
	Volumes           []corev1.Volume     `json:"volumes,omitempty"`
	PriorityClassName string              `json:"priorityClassName,omitempty"`
	NodeSelector      corev1.NodeSelector `json:"nodeSelector,omitempty"`
	Affinity          corev1.Affinity     `json:"affinity,omitempty"`
	Tolerations       []corev1.Toleration `json:"tolerations,omitempty"`
	PodAnnotations    map[string]string   `json:"podAnnotations,omitempty"`
	PodLabels         map[string]string   `json:"podLabels,omitempty"`
}
