package sleepmode

import (
	"github.com/loft-sh/vcluster/pkg/kube"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"golang.org/x/mod/semver"
)

const (
	Label                   = "loft.sh/sleep-mode"
	SleepingSinceAnnotation = "sleepmode.loft.sh/sleeping-since"
)

const v24 = "v0.24.0-alpha.0"

func IsSleeping(labeled kube.Labeled) bool {
	return labeled.GetLabels()[Label] == "true"
}

func IsInstanceSleeping(annotated kube.Annotated) bool {
	return annotated != nil && annotated.GetAnnotations()[SleepingSinceAnnotation] != ""
}

func Warning() string {
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
