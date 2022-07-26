package manifests

type Chart struct {
	Name             string `json:"name,omitempty"`
	Repo             string `json:"repo,omitempty"`
	Version          string `json:"version,omitempty"`
	Username         string `json:"username,omitempty"`
	Password         string `json:"password,omitempty"`
	Values           string `json:"values,omitempty"`
	Timeout          string `json:"timeout,omitempty"`
	ReleaseName      string `json:"releaseName,omitempty"`
	ReleaseNamespace string `json:"releaseNamespace,omitempty"`
	Bundle           string `json:"bundle,omitempty"`
}

type ChartStatus struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Phase     string `json:"phase,omitempty"`
	Message   string `json:"message,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type Ready struct {
	Ready  bool   `json:"ready,omitempty"`
	Phase  string `json:"phase,omitempty"`
	Reason string `json:"reason,omitempty"`
}
