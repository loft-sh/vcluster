package platform

type ManagerType string

const (
	ManagerHelm     ManagerType = "helm"
	ManagerPlatform ManagerType = "platform"
)

type ManagerConfig struct {
	// Manager is the current manager that is used, either helm or platform
	Manager ManagerType `json:"manager,omitempty"`
}
