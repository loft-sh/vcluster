package manifests

type Status struct {
	Phase   string `json:"phase,omitempty"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`

	Charts    []ChartStatus   `json:"charts,omitempty"`
	Manifests ManifestsStatus `json:"manifests,omitempty"`
}

//revive:disable

type ManifestsStatus struct {
	Phase                string `json:"phase,omitempty"`
	Reason               string `json:"reason,omitempty"`
	Message              string `json:"message,omitempty"`
	LastAppliedManifests string `json:"lastAppliedManifests,omitempty"`
}

//revive:enable

type ChartStatus struct {
	Name                       string `json:"name,omitempty"`
	Namespace                  string `json:"namespace,omitempty"`
	Phase                      string `json:"phase,omitempty"`
	Reason                     string `json:"reason,omitempty"`
	Message                    string `json:"message,omitempty"`
	LastAppliedChartConfigHash string `json:"lastAppliedChartConfigHash,omitempty"`
}

type Chart struct {
	Name             string `json:"name,omitempty"`
	Repo             string `json:"repo,omitempty"`
	Insecure         bool   `json:"insecure,omitempty"`
	Version          string `json:"version,omitempty"`
	Username         string `json:"username,omitempty"`
	Password         string `json:"password,omitempty"`
	Values           string `json:"values,omitempty"`
	Timeout          string `json:"timeout,omitempty"`
	Bundle           string `json:"bundle,omitempty"`
	ReleaseName      string `json:"releaseName,omitempty"`
	ReleaseNamespace string `json:"releaseNamespace,omitempty"`
}
