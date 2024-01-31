package helmvalues

type K8s struct {
	BaseHelm
	Syncer       K8sSyncerValues    `json:"syncer,omitempty"`
	API          APIServerValues    `json:"api,omitempty"`
	Controller   ControllerValues   `json:"controller,omitempty"`
	Scheduler    SchedulerValues    `json:"scheduler,omitempty"`
	Etcd         EtcdValues         `json:"etcd,omitempty"`
	EmbeddedEtcd EmbeddedEtcdValues `json:"embeddedEtcd,omitempty"`
}

type K8sSyncerValues struct {
	SyncerValues
	CommonValues
	SecurityContext    map[string]interface{} `json:"securityContext,omitempty"`
	PodSecurityContext map[string]interface{} `json:"podSecurityContext,omitempty"`
}

type APIServerValues struct {
	SyncerExORCommonValues
	ControlPlaneCommonValues
	SecurityContext    map[string]interface{} `json:"securityContext,omitempty"`
	ServiceAnnotations map[string]string      `json:"serviceAnnotations,omitempty"`
}

type ControllerValues struct {
	SyncerExORCommonValues
	ControlPlaneCommonValues
}

type SchedulerValues struct {
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
	SecurityContext                  map[string]interface{} `json:"securityContext,omitempty"`
	ServiceAnnotations               map[string]string      `json:"serviceAnnotations,omitempty"`
	AutoDeletePersistentVolumeClaims bool                   `json:"autoDeletePersistentVolumeClaims,omitempty"`
	Replicas                         uint32                 `json:"replicas,omitempty"`
	Labels                           map[string]string      `json:"labels,omitempty"`
	Annotations                      map[string]string      `json:"annotations,omitempty"`
	Storage                          Storage                `json:"storage,omitempty"`
}
