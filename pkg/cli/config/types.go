package config

import (
	"github.com/loft-sh/vcluster/pkg/manager"
	"github.com/loft-sh/vcluster/pkg/platform"
)

type Config struct {
	TelemetryDisabled bool           `json:"telemetryDisabled,omitempty"`
	Platform          PlatformConfig `json:"platform,omitempty"`
	Manager           ManagerConfig  `json:"manager,omitempty"`
}

// PlatformConfig defines the platform client config structure
type PlatformConfig struct {
	platform.Config `json:",inline"`
}

type ManagerConfig struct {
	// Type is the current manager type that is used, either helm or platform
	Type manager.Type `json:"type,omitempty"`
}
