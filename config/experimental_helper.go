package config

import (
	"strings"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/log"
)

type (
	ExperimentalConfig struct {
		Experimental map[string]any
	}
)

func ExperimentalWarning(logger log.Logger, currentValues []byte, advisors map[string]func() string) string {
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
