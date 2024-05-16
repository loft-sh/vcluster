package config

import (
	"github.com/loft-sh/vcluster/pkg/platform"
)

const (
	ManagerHelm     ManagerType = "helm"
	ManagerPlatform ManagerType = "platform"
)

type Config struct {
	TelemetryDisabled bool            `json:"telemetryDisabled,omitempty"`
	Platform          *PlatformConfig `json:"platform,omitempty"`
}

// CLIConfig defines the cli config structure
type CLIConfig struct {
	TelemetryDisabled bool `json:"telemetryDisabled,omitempty"`
}

// PlatformConfig defines the platform client config structure
type PlatformConfig struct {
	platform.Config
}

type ManagerType string

type ManagerConfig struct {
	// Manager is the current manager that is used, either helm or platform
	Manager ManagerType `json:"manager,omitempty"`
}
