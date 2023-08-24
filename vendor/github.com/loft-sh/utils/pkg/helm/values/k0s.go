package values

import (
	"strings"

	"github.com/go-logr/logr"
	"github.com/loft-sh/utils/pkg/helm"
)

var K0SVersionMap = map[string]string{
	"1.27": "k0sproject/k0s:v1.27.3-k0s.0",
	"1.26": "k0sproject/k0s:v1.26.6-k0s.0",
	"1.25": "k0sproject/k0s:v1.25.11-k0s.0",
	"1.24": "k0sproject/k0s:v1.24.15-k0s.0",
}

func getDefaultK0SReleaseValues(chartOptions *helm.ChartOptions, log logr.Logger) (string, error) {
	serverVersionString := GetKubernetesVersion(chartOptions.KubernetesVersion)
	serverMinorInt, err := GetKubernetesMinorVersion(chartOptions.KubernetesVersion)
	if err != nil {
		return "", err
	}

	image, ok := K0SVersionMap[serverVersionString]
	if !ok {
		if serverMinorInt > 27 {
			log.Info("officially unsupported host server version, will fallback to virtual cluster version v1.27", "serverVersion", serverVersionString)
			image = K0SVersionMap["1.27"]
		} else {
			log.Info("officially unsupported host server version, will fallback to virtual cluster version v1.24", "serverVersion", serverVersionString)
			image = K0SVersionMap["1.24"]
		}
	}

	// build values
	values := `vcluster:
  image: ##IMAGE##
`
	values = strings.ReplaceAll(values, "##IMAGE##", image)
	return addCommonReleaseValues(values, chartOptions)
}
