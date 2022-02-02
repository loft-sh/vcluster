package values

import (
	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/loft-sh/vcluster/pkg/log"
)

func GetDefaultReleaseValues(chartOptions *helm.ChartOptions, log log.Logger) (string, error) {
	if chartOptions.ChartName == helm.K3SChart {
		return getDefaultK3SReleaseValues(chartOptions, log)
	} else if chartOptions.ChartName == helm.K0SChart {
		return getDefaultK0SReleaseValues(chartOptions, log)
	} else if chartOptions.ChartName == helm.K8SChart {
		return getDefaultK8SReleaseValues(chartOptions, log)
	} else if chartOptions.ChartName == helm.EKSChart {
		return getDefaultEKSReleaseValues(chartOptions, log)
	}

	return "", nil
}
