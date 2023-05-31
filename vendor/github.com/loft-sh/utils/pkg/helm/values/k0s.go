package values

import (
	"strings"

	"github.com/loft-sh/utils/pkg/helm"
	"github.com/loft-sh/utils/pkg/log"
)

var K0SVersionMap = map[string]string{
	"1.27": "k0sproject/k0s:v1.27.1-k0s.0",
	"1.26": "k0sproject/k0s:v1.26.4-k0s.0",
	"1.25": "k0sproject/k0s:v1.25.8-k0s.1",
	"1.24": "k0sproject/k0s:v1.24.12-k0s.1",
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
		if serverMinorInt > 26 {
			log.Infof("officially unsupported host server version %s, will fallback to virtual cluster version v1.26", serverVersionString)
			image = K0SVersionMap["1.26"]
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
