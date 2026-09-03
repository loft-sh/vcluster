package config

import (
	"strings"

	"github.com/loft-sh/api/v4/pkg/vclusterconfig"
	"github.com/loft-sh/log"
	"sigs.k8s.io/yaml"
)

type (
	LegacyConfig struct {
		Experimental map[string]any

		// nolint:staticcheck // SA1019: vclusterconfig.LegacySleepMode is deprecated
		SleepMode *vclusterconfig.LegacySleepMode `json:"sleepMode,omitempty" yaml:"sleepMode,omitempty"`

		External *struct {
			// nolint:staticcheck // SA1019: vclusterconfig.LegacyPlatformConfig is deprecated
			Platform *vclusterconfig.LegacyPlatformConfig `json:"platform,omitempty" yaml:"platform,omitempty"`
		} `json:"external,omitempty" yaml:"external,omitempty"`
	}
)

func ConfigStructureWarning(logger log.Logger, currentValues []byte, advisors map[string]func() string) string {
	legacy := &LegacyConfig{}
	if err := yaml.Unmarshal(currentValues, legacy); err != nil {
		logger.Warn(err)
		return ""
	}

	var warnings []string
	for k := range legacy.Experimental {
		if advisor, ok := advisors[k]; ok {
			if warning := advisor(); warning != "" {
				warnings = append(warnings, warning)
			}
		}
	}

	if legacy.SleepMode != nil {
		if advisor, ok := advisors["sleepMode"]; ok {
			if warning := advisor(); warning != "" {
				warnings = append(warnings, warning)
			}
		}
	}

	if legacy.External != nil {
		if legacy.External.Platform != nil {
			if legacy.External.Platform.APIKey != nil || legacy.External.Platform.Project != "" {
				if advisor, ok := advisors["platform"]; ok {
					if warning := advisor(); warning != "" {
						warnings = append(warnings, warning)
					}
				}
			}
			if legacy.External.Platform.AutoDelete != nil {
				if advisor, ok := advisors["autoDelete"]; ok {
					if warning := advisor(); warning != "" {
						warnings = append(warnings, warning)
					}
				}
			}
			if legacy.External.Platform.AutoSleep != nil {
				if advisor, ok := advisors["autoSleep"]; ok {
					if warning := advisor(); warning != "" {
						warnings = append(warnings, warning)
					}
				}
			}
			if legacy.External.Platform.AutoSnapshot != nil {
				if advisor, ok := advisors["autoSnapshot"]; ok {
					if warning := advisor(); warning != "" {
						warnings = append(warnings, warning)
					}
				}
			}
		}
	}

	if len(warnings) == 0 {
		return ""
	}

	return strings.Join(warnings, "\n")
}
