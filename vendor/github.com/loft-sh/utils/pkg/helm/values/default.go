package values

import (
	"github.com/go-logr/logr"
	"github.com/loft-sh/utils/pkg/helm"
)

func GetDefaultReleaseValues(chartOptions *helm.ChartOptions, log logr.Logger) (string, error) {
	switch chartOptions.ChartName {
	case helm.K3SChart, helm.K3SProChart:
		return getDefaultK3SReleaseValues(chartOptions, log)
	case helm.K0SChart, helm.K0SProChart:
		return getDefaultK0SReleaseValues(chartOptions, log)
	case helm.K8SChart, helm.K8SProChart:
		return getDefaultK8SReleaseValues(chartOptions, log)
	case helm.EKSChart, helm.EKSProChart:
		return getDefaultEKSReleaseValues(chartOptions, log)
	}

	return "", nil
}
