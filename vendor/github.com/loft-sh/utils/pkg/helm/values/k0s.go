package values

import (
	"strings"

	"github.com/go-logr/logr"
	"github.com/loft-sh/utils/pkg/helm"
)

var K0SVersionMap = map[string]string{
	"1.28": "k0sproject/k0s:v1.28.2-k0s.0",
	"1.27": "k0sproject/k0s:v1.27.6-k0s.0",
	"1.26": "k0sproject/k0s:v1.26.9-k0s.0",
	"1.25": "k0sproject/k0s:v1.25.14-k0s.0",
}

func getDefaultK0SReleaseValues(chartOptions *helm.ChartOptions, log logr.Logger) (string, error) {
	image := ""
	if chartOptions.KubernetesVersion.Major != "" && chartOptions.KubernetesVersion.Minor != "" {
		serverVersionString := GetKubernetesVersion(chartOptions.KubernetesVersion)
		serverMinorInt, err := GetKubernetesMinorVersion(chartOptions.KubernetesVersion)
		if err != nil {
			return "", err
		}

		var ok bool
		image, ok = K0SVersionMap[serverVersionString]
		if !ok {
			if serverMinorInt > 28 {
				log.Info("officially unsupported host server version, will fallback to virtual cluster version v1.28", "serverVersion", serverVersionString)
				image = K0SVersionMap["1.28"]
			} else {
				log.Info("officially unsupported host server version, will fallback to virtual cluster version v1.25", "serverVersion", serverVersionString)
				image = K0SVersionMap["1.25"]
			}
		}
	}

	// build values
	values := ""
	if image != "" {
		values = `vcluster:
  image: ##IMAGE##
`
		values = strings.ReplaceAll(values, "##IMAGE##", image)
	}
	return addCommonReleaseValues(values, chartOptions)
}
