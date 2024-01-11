package helmvalues

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
	SecurityContext    map[string]interface{} `json:"securityContext,omitempty"`
	PodSecurityContext map[string]interface{} `json:"podSecurityContext,omitempty"`
}

type APIServerValues struct {
	CommonValues
	SyncerExORCommonValues
	ControlPlaneCommonValues
	SecurityContext    map[string]interface{} `json:"securityContext,omitempty"`
	ServiceAnnotations map[string]string      `json:"serviceAnnotations,omitempty"`
}

type ControllerValues struct {
	CommonValues
	SyncerExORCommonValues
	ControlPlaneCommonValues
	SecurityContext map[string]interface{} `json:"securityContext,omitempty"`
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
	SecurityContext                  map[string]interface{} `json:"securityContext,omitempty"`
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
	VolumeMounts []map[string]interface{} `json:"volumeMounts,omitempty"`
	ExtraArgs    []string                 `json:"extraArgs,omitempty"`
	Resources    map[string]interface{}   `json:"resources,omitempty"`
}

type CommonValues struct {
	Volumes           []map[string]interface{} `json:"volumes,omitempty"`
	PriorityClassName string                   `json:"priorityClassName,omitempty"`
	NodeSelector      map[string]interface{}   `json:"nodeSelector,omitempty"`
	Affinity          map[string]interface{}   `json:"affinity,omitempty"`
	Tolerations       []map[string]interface{} `json:"tolerations,omitempty"`
	PodAnnotations    map[string]string        `json:"podAnnotations,omitempty"`
	PodLabels         map[string]string        `json:"podLabels,omitempty"`
}
