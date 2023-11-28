package telemetry

type ChartInfo struct {
	Name    string
	Version string

	Values map[string]interface{}
}

type Config struct {
	Disabled           bool   `json:"disabled,omitempty"`
	InstanceCreator    string `json:"instanceCreator,omitempty"`
	PlatformUserID     string `json:"platformUserID,omitempty"`
	PlatformInstanceID string `json:"platformInstanceID,omitempty"`
	MachineID          string `json:"machineID,omitempty"`
}

type KubernetesVersion struct {
	Major      string `json:"major"`
	Minor      string `json:"minor"`
	GitVersion string `json:"gitVersion"`
}
