package sleepmode

import (
	"github.com/loft-sh/vcluster/pkg/kube"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"golang.org/x/mod/semver"
)

const (
	Label                   = "loft.sh/sleep-mode"
	SleepingSinceAnnotation = "sleepmode.loft.sh/sleeping-since"

	// AnnotationAgentInstalled is set on the vc-config secret when the vCluster platform agent
	// is installed and managing sleep state. When present, the native sleep mode controller defers
	// to the agent.
	AnnotationAgentInstalled = "vcluster.loft.sh/agent-installed"

	// StandaloneSleepSecretName is the name of the secret in the virtual cluster's default namespace
	// that tracks sleep state for standalone vClusters (ControlPlane.Standalone.Enabled = true).
	StandaloneSleepSecretName = "vc-standalone-sleep-state"
)

const v32 = "v0.32.0-alpha.0"

func IsSleeping(labeled kube.Labeled) bool {
	return labeled.GetLabels()[Label] == "true"
}

func IsInstanceSleeping(annotated kube.Annotated) bool {
	return annotated != nil && annotated.GetAnnotations()[SleepingSinceAnnotation] != ""
}

func Warning() string {
	if semver.Compare("v"+upgrade.GetVersion(), v32) == -1 {
		return ""
	}

	return `Sleep configuration is no longer under "sleepMode" and "enabled" has been removed. Please update your values and specify them with --values.
For example:

|sleepMode:                           |sleep:
|  enabled: true            ---->     |  auto:
|  autoSleep:                         |    afterInactivity: 24h
|    afterInactivity: 24h             |

`
}
