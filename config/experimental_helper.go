package config

import (
	"strings"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"golang.org/x/mod/semver"
)

type (
	ExperimentalConfig struct {
		Experimental map[string]any
	}
)

var advisors = map[string]func() (warning string){
	"sleepMode": SleepModeWarning,
}

func ExperimentalWarning(logger log.Logger, currentValues []byte) string {
	exp := &ExperimentalConfig{}
	if err := yaml.Unmarshal(currentValues, exp); err != nil {
		logger.Warn(err)
		return ""
	}

	var advice []string
	for k := range exp.Experimental {
		if advisor, ok := advisors[k]; ok {
			if warning := advisor(); warning != "" {
				advice = append(advice, warning)
			}
		}
	}

	if len(advice) == 0 {
		return ""
	}

	expWarning := "An experimental feature you were using has been promoted! 🎉 See below on tips to update."
	return strings.Join(append([]string{expWarning}, advice...), "\n")
}

const v24 = "v0.24.0-alpha.0"

func SleepModeWarning() string {
	// if we're not upgrading to v0.24+ no warning
	if semver.Compare("v"+upgrade.GetVersion(), v24) == -1 {
		return ""
	}

	return `
sleepMode configuration is no longer under experimental. Please update your values and specify them with --values.

For example

|experimental:                          |sleepMode:
|  sleepMode:                           |  enabled: true
|    enabled: true            ---->     |  autoSleep:
|    autoSleep:                         |    afterInactivity: 24h
|      afterInactivity: 24h             |

`
}
