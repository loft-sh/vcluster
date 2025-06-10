package sleepmode

import "github.com/loft-sh/vcluster/pkg/kube"

const (
	Label                   = "loft.sh/sleep-mode"
	SleepingSinceAnnotation = "sleepmode.loft.sh/sleeping-since"
)

func IsSleeping(labeled kube.Labeled) bool {
	return labeled.GetLabels()[Label] == "true"
}

func IsInstanceSleeping(annotated kube.Annotated) bool {
	return annotated != nil && annotated.GetAnnotations()[SleepingSinceAnnotation] != ""
}
