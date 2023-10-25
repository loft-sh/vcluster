package helmvalues

import corev1 "k8s.io/api/core/v1"

type K8s struct {
	BaseHelm
	Syncer     K8sSyncerValues  `json:"syncer,omitempty"`
	API        APIServerValues  `json:"api,omitempty"`
	Controller ControllerValues `json:"controller,omitempty"`
	Scheduler  SchedulerValues  `json:"scheduler,omitempty"`
	Etcd       EtcdValues       `json:"etcd,omitempty"`
	Job        JobValues        `json:"job,omitempty"`
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
}

type EtcdValues struct {
	CommonValues
	SyncerExORCommonValues
	ControlPlaneCommonValues
	Storage struct {
		Persistence bool   `json:"persistence,omitempty"`
		Size        string `json:"size,omitempty"`
	} `json:"storage,omitempty"`
	SecurityContext    corev1.SecurityContext `json:"securityContext,omitempty"`
	ServiceAnnotations map[string]string      `json:"serviceAnnotations,omitempty"`
}

type JobValues struct {
	CommonValues
	SyncerExORCommonValues
	Enabled         bool `json:"enabled,omitempty"`
	SecurityContext struct {
		Capabilities             corev1.Capabilities `json:"capabilities,omitempty"`
		AllowPrivilegeEscalation bool                `json:"allowPrivilegeEscalation,omitempty"`
		ReadOnlyRootFilesystem   bool                `json:"readOnlyRootFilesystem,omitempty"`
		RunAsNonRoot             bool                `json:"runAsNonRoot,omitempty"`
		RunAsUser                uint32              `json:"runAsUser,omitempty"`
		RunAsGroup               uint32              `json:"runAsGroup,omitempty"`
	} `json:"securityContext,omitempty"`
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
