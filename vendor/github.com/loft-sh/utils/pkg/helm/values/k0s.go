package values

import (
	"strings"

	"github.com/loft-sh/utils/pkg/helm"
	"github.com/loft-sh/utils/pkg/log"
)

var K0SVersionMap = map[string]string{
	"1.27": "k0sproject/k0s:v1.27.2-k0s.0",
	"1.26": "k0sproject/k0s:v1.26.5-k0s.0",
	"1.25": "k0sproject/k0s:v1.25.10-k0s.0",
	"1.24": "k0sproject/k0s:v1.24.14-k0s.0",
	"1.23": "k0sproject/k0s:v1.23.15-k0s.0",
}

func getDefaultK0SReleaseValues(chartOptions *helm.ChartOptions, log log.SimpleLogger) (string, error) {
	serverVersionString := GetKubernetesVersion(chartOptions.KubernetesVersion)
	serverMinorInt, err := GetKubernetesMinorVersion(chartOptions.KubernetesVersion)
	if err != nil {
		return "", err
	}

	image, ok := K0SVersionMap[serverVersionString]
	if !ok {
		if serverMinorInt > 27 {
			log.Infof("officially unsupported host server version %s, will fallback to virtual cluster version v1.27", serverVersionString)
			image = K0SVersionMap["1.27"]
		} else {
			log.Infof("officially unsupported host server version %s, will fallback to virtual cluster version v1.23", serverVersionString)
			image = K0SVersionMap["1.23"]
		}
	}

	// build values
	values := `vcluster:
  image: ##IMAGE##
`
	values = strings.ReplaceAll(values, "##IMAGE##", image)
	return addCommonReleaseValues(values, chartOptions)
}
