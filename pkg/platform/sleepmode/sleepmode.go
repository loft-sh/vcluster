package sleepmode

import (
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"github.com/loft-sh/vcluster/config"
)

const (
	SleepModeLabel                   = "loft.sh/sleep-mode"
	SleepModeSleepingSinceAnnotation = "sleepmode.loft.sh/sleeping-since"
)

func IsConfigured(vClusterConfig *config.Config) bool {
	if vClusterConfig == nil || vClusterConfig.External == nil || vClusterConfig.External["platform"] == nil {
		return false
	}
	return vClusterConfig.External["platform"]["autoSleep"] != nil || vClusterConfig.External["platform"]["autoDelete"] != nil
}

func IsSleeping(labels map[string]string) bool {
	if labels != nil && labels[SleepModeLabel] == "true" {
		return true
	}
	return false
}

func IsInstanceSleeping(instance *managementv1.VirtualClusterInstance) bool {
	if instance == nil {
		return false
	}

	if instance.Annotations != nil && instance.Annotations[SleepModeSleepingSinceAnnotation] != "" {
		return true
	}

	return false
}
