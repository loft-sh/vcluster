package server

// +genclient
// +genclient:nonNamespaced

// InstanceCreateInput is the required input data for "instance create" operations, that is, the
// primary endpoint that Loft instances will hit to register to the license server as well as get
// information about the instance's current license.
type InstanceCreateInput struct {
	*InstanceTokenAuth

	LoftVersion string `form:"version"     json:"version"     validate:"required"`
	KubeVersion string `form:"kubeVersion" json:"kubeVersion" validate:"required"`

	KubeSystemNamespaceUID string `form:"kubeSystemNamespaceUID" json:"kubeSystemNamespace" validate:"required"`

	// AllocatedResources is a mapping of all resources (that we track, i.e. vcluster instances)
	// deployed on the Loft instance. The Loft instance passes this information along when it
	// performs it's checkins with the license server.
	AllocatedResources *map[string]ResourceQuantity `form:"quantities" json:"quantities,omitempty"`

	// Config is the current configuration of the Loft instance checking in.
	Config string `form:"config" json:"config,omitempty"`
}

// InstanceCreateOutput is the struct holding all information returned from "instance create"
// requests.
type InstanceCreateOutput struct {
	CurrentTime int64 `json:"currentTime"`
	// License is the license data for the requested Loft instance.
	License *License `json:"license,omitempty"`
}

// License is a struct representing the license data sent to a Loft instance after checking in with
// the license server.
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type License struct {
	// Buttons is a slice of license server endpoints (buttons) that the Loft instance may need to
	// hit. Each Button contains the display text and link for the front end to work with.
	Buttons Buttons `json:"buttons,omitempty"`
	// IsOffline indicates if the license is an offline license or not.
	// +optional
	IsOffline bool `json:"isOffline,omitempty"`
	// Announcements is a map string/string such that we can easily add any additional data without
	// needing to change types. For now, we will use the keys "name" and "content".
	// +optional
	Announcements map[string]string `json:"announcement,omitempty"`
	// BlockRequests is a slice of Request objects that the Loft instance should block from being
	// created due to license usage overrun.
	// +optional
	BlockRequests *[]Request `json:"blockRequests,omitempty"`
	// Limits is the number of resources allowed by the current license.
	// +optional
	Limits []ResourceQuantity `json:"limits,omitempty"`
	// Features is a map of enabled features.
	// +optional
	Features map[string]bool `json:"features,omitempty"`
	// Analytics indicates the analytics endpoints and which requests should be sent to the
	// analytics server.
	// +optional
	Analytics *Analytics `json:"analytics,omitempty"`
}

// Analytics is a struct that represents the analytics server and the requests that should be sent
// to it. This information is sent to Loft instances when they check in with the license server.
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type Analytics struct {
	// Endpoint is the endpoint for the analytics server.
	Endpoint string `json:"endpoint,omitempty"`
	// Requests is a slice of requested resources to return analytics for.
	// +optional
	Requests []Request `json:"requests,omitempty"`
}

// Request represents a request analytics information for an apigroup/resource and a list of verb actions for that
// resource.
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type Request struct {
	// Verbs is the list of verbs for the request.
	// +optional
	Verbs []string `json:"verbs,omitempty"`
	// Group is the api group.
	// +optional
	Group string `json:"group,omitempty"`
	// Resource is the resource name for the request.
	// +optional
	Resource string `json:"resource,omitempty"`
}

// ResourceQuantity represents an api resource and a quantity counter for that resource type (used for limits for example).
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type ResourceQuantity struct {
	// Group is the api group.
	// +optional
	Group string `json:"group,omitempty"`
	// Version is the api version.
	// +optional
	Version string `json:"version,omitempty"`
	// Kind is the resource kind.
	// +optional
	Kind string `json:"kind,omitempty"`
	// Quantity is the quantity for hte limit (for example).
	// +optional
	Quantity int64 `json:"quantity,omitempty"`
}
