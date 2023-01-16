package values

import (
	"strings"

	"github.com/loft-sh/utils/pkg/helm"
	"github.com/loft-sh/utils/pkg/log"
)

var K0SVersionMap = map[string]string{
	"1.25": "k0sproject/k0s:v1.25.3-k0s.0",
	"1.24": "k0sproject/k0s:v1.24.7-k0s.0",
	"1.23": "k0sproject/k0s:v1.23.13-k0s.0",
	"1.22": "k0sproject/k0s:v1.22.15-k0s.0",
}

func getDefaultK0SReleaseValues(chartOptions *helm.ChartOptions, log log.Logger) (string, error) {
	serverVersionString := GetKubernetesVersion(chartOptions.KubernetesVersion)
	serverMinorInt, err := GetKubernetesMinorVersion(chartOptions.KubernetesVersion)
	if err != nil {
		return "", err
	}

	image, ok := K0SVersionMap[serverVersionString]
	if !ok {
		if serverMinorInt > 25 {
			log.Infof("officially unsupported host server version %s, will fallback to virtual cluster version v1.25", serverVersionString)
			image = K0SVersionMap["1.25"]
		} else {
			log.Infof("officially unsupported host server version %s, will fallback to virtual cluster version v1.22", serverVersionString)
			image = K0SVersionMap["1.22"]
		}
	}

	// build values
	values := `vcluster:
  image: ##IMAGE##
`
	values = strings.ReplaceAll(values, "##IMAGE##", image)
	return addCommonReleaseValues(values, chartOptions)
}
