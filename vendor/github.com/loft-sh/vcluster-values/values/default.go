package values

import (
	"github.com/go-logr/logr"
)

func GetDefaultReleaseValues(chartOptions *ChartOptions, log logr.Logger) (string, error) {
	switch chartOptions.ChartName {
	case K3SChart:
		return getDefaultK3SReleaseValues(chartOptions, log)
	case K0SChart:
		return getDefaultK0SReleaseValues(chartOptions, log)
	case K8SChart:
		return getDefaultK8SReleaseValues(chartOptions, log)
	case EKSChart:
		return getDefaultEKSReleaseValues(chartOptions, log)
	}

	return "", nil
}
