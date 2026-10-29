package deploy

type Status struct {
	Phase   string `json:"phase,omitempty"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`

	Charts    []ChartStatus   `json:"charts,omitempty"`
	Manifests ManifestsStatus `json:"manifests,omitempty"`
}

type ManifestsStatus struct {
	Phase                string `json:"phase,omitempty"`
	Reason               string `json:"reason,omitempty"`
	Message              string `json:"message,omitempty"`
	LastAppliedManifests string `json:"lastAppliedManifests,omitempty"`
}

type ChartStatus struct {
	Name                       string `json:"name,omitempty"`
	Namespace                  string `json:"namespace,omitempty"`
	Phase                      string `json:"phase,omitempty"`
	Reason                     string `json:"reason,omitempty"`
	Message                    string `json:"message,omitempty"`
	LastAppliedChartConfigHash string `json:"lastAppliedChartConfigHash,omitempty"`
}
