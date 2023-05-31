package values

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/loft-sh/utils/pkg/helm"
	"github.com/loft-sh/utils/pkg/log"
)

var K3SVersionMap = map[string]string{
	"1.27": "rancher/k3s:v1.27.1-k3s1",
	"1.26": "rancher/k3s:v1.26.4-k3s1",
	"1.25": "rancher/k3s:v1.25.9-k3s1",
	"1.24": "rancher/k3s:v1.24.13-k3s1",
	"1.23": "rancher/k3s:v1.23.17-k3s1",
}

var replaceRegEx = regexp.MustCompile("[^0-9]+")

func getDefaultK3SReleaseValues(chartOptions *helm.ChartOptions, log log.SimpleLogger) (string, error) {
	var (
		image               = chartOptions.K3SImage
		serverVersionString string
		serverMinorInt      int
		err                 error
	)

	if image == "" {
		serverVersionString = GetKubernetesVersion(chartOptions.KubernetesVersion)
		serverMinorInt, err = GetKubernetesMinorVersion(chartOptions.KubernetesVersion)
		if err != nil {
			return "", err
		}

		var ok bool
		image, ok = K3SVersionMap[serverVersionString]
		if !ok {
			if serverMinorInt > 26 {
				log.Infof("officially unsupported host server version %s, will fallback to virtual cluster version v1.26", serverVersionString)
				image = K3SVersionMap["1.26"]
			} else {
				log.Infof("officially unsupported host server version %s, will fallback to virtual cluster version v1.23", serverVersionString)
				image = K3SVersionMap["1.23"]
			}
		}
	}

	// build values
	values := `vcluster:
  image: ##IMAGE##
##BASEARGS##
`
	if chartOptions.Isolate {
		values += `
securityContext:
  runAsUser: 12345
  runAsNonRoot: true`
	}

	values = strings.ReplaceAll(values, "##IMAGE##", image)

	return addCommonReleaseValues(values, chartOptions)
}

func addCommonReleaseValues(values string, chartOptions *helm.ChartOptions) (string, error) {
	if chartOptions.CIDR != "" {
		values += `
serviceCIDR: ##CIDR##`
		values = strings.ReplaceAll(values, "##CIDR##", chartOptions.CIDR)
	}

	if chartOptions.DisableIngressSync {
		values += `
syncer:
  extraArgs: ["--disable-sync-resources=ingresses"]`
	}

	if chartOptions.CreateClusterRole {
		values += `
rbac:
  clusterRole:
    create: true`
	}

	if chartOptions.Expose {
		values += `
service:
  type: LoadBalancer`
	} else if chartOptions.NodePort {
		values += `
service:
  type: NodePort`
	}

	if chartOptions.SyncNodes {
		values += `
sync:
  nodes:
    enabled: true`
	}

	if chartOptions.Isolate {
		values += `
isolation:
  enabled: true`
	}

	if chartOptions.DisableTelemetry {
		values += `
telemetry:
  disabled: "true"`
	} else if chartOptions.InstanceCreatorType != "" || chartOptions.InstanceCreatorUID != "" {
		values += `
telemetry:
  disabled: "false"
  instanceCreator: "##INSTANCE_CREATOR##"
  instanceCreatorUID: "##INSTANCE_CREATOR_UID##"`
		values = strings.ReplaceAll(values, "##INSTANCE_CREATOR##", chartOptions.InstanceCreatorType)
		values = strings.ReplaceAll(values, "##INSTANCE_CREATOR_UID##", chartOptions.InstanceCreatorUID)
	}

	values = strings.TrimSpace(values)
	return values, nil
}

func ParseKubernetesVersionInfo(versionStr string) (*helm.Version, error) {
	if versionStr[0] == 'v' {
		versionStr = versionStr[1:]
	}

	splittedVersion := strings.Split(versionStr, ".")
	if len(splittedVersion) != 2 && len(splittedVersion) != 3 {
		return nil, fmt.Errorf("unrecognized kubernetes version %s, please use format vX.X", versionStr)
	}

	major := splittedVersion[0]
	minor := splittedVersion[1]

	return &helm.Version{
		Major: major,
		Minor: minor,
	}, nil
}

func GetKubernetesVersion(serverVersion helm.Version) string {
	return replaceRegEx.ReplaceAllString(serverVersion.Major, "") + "." + replaceRegEx.ReplaceAllString(serverVersion.Minor, "")
}

func GetKubernetesMinorVersion(serverVersion helm.Version) (int, error) {
	return strconv.Atoi(replaceRegEx.ReplaceAllString(serverVersion.Minor, ""))
}
