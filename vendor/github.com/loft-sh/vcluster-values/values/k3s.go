package values

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
)

var K3SVersionMap = map[string]string{
	"1.29": "rancher/k3s:v1.29.0-k3s1",
	"1.28": "rancher/k3s:v1.28.5-k3s1",
	"1.27": "rancher/k3s:v1.27.9-k3s1",
	"1.26": "rancher/k3s:v1.26.12-k3s1",
}

var replaceRegEx = regexp.MustCompile("[^0-9]+")

func getDefaultK3SReleaseValues(chartOptions *ChartOptions, log logr.Logger) (string, error) {
	var (
		image               = chartOptions.K3SImage
		serverVersionString string
		serverMinorInt      int
		err                 error
	)

	if image == "" && chartOptions.KubernetesVersion.Major != "" && chartOptions.KubernetesVersion.Minor != "" {
		serverVersionString = GetKubernetesVersion(chartOptions.KubernetesVersion)
		serverMinorInt, err = GetKubernetesMinorVersion(chartOptions.KubernetesVersion)
		if err != nil {
			return "", err
		}

		var ok bool
		image, ok = K3SVersionMap[serverVersionString]
		if !ok {
			if serverMinorInt > 29 {
				log.Info("officially unsupported host server version, will fallback to virtual cluster version v1.28", "serverVersion", serverVersionString)
				image = K3SVersionMap["1.29"]
			} else {
				log.Info("officially unsupported host server version, will fallback to virtual cluster version v1.26", "serverVersion", serverVersionString)
				image = K3SVersionMap["1.26"]
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
	if chartOptions.Isolate {
		values += `
securityContext:
  runAsUser: 12345
  runAsNonRoot: true`
	}
	return addCommonReleaseValues(values, chartOptions)
}

func addCommonReleaseValues(values string, chartOptions *ChartOptions) (string, error) {
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
  disabled: true`
	} else if chartOptions.InstanceCreatorType != "" {
		values += `
telemetry:
  disabled: false
  instanceCreator: "##INSTANCE_CREATOR##"
  platformUserID: "##PLATFORM_USER_ID##"
  platformInstanceID: "##PLATFORM_INSTANCE_ID##"
  machineID: "##MACHINE_ID##"`
		values = strings.ReplaceAll(values, "##INSTANCE_CREATOR##", chartOptions.InstanceCreatorType)
		values = strings.ReplaceAll(values, "##PLATFORM_USER_ID##", chartOptions.PlatformUserID)
		values = strings.ReplaceAll(values, "##PLATFORM_INSTANCE_ID##", chartOptions.PlatformInstanceID)
		values = strings.ReplaceAll(values, "##MACHINE_ID##", chartOptions.MachineID)
	}

	values = strings.TrimSpace(values)
	return values, nil
}

func ParseKubernetesVersionInfo(versionStr string) (*Version, error) {
	if versionStr[0] == 'v' {
		versionStr = versionStr[1:]
	}

	splittedVersion := strings.Split(versionStr, ".")
	if len(splittedVersion) != 2 && len(splittedVersion) != 3 {
		return nil, fmt.Errorf("unrecognized kubernetes version %s, please use format vX.X", versionStr)
	}

	major := splittedVersion[0]
	minor := splittedVersion[1]

	return &Version{
		Major: major,
		Minor: minor,
	}, nil
}

func GetKubernetesVersion(serverVersion Version) string {
	return replaceRegEx.ReplaceAllString(serverVersion.Major, "") + "." + replaceRegEx.ReplaceAllString(serverVersion.Minor, "")
}

func GetKubernetesMinorVersion(serverVersion Version) (int, error) {
	return strconv.Atoi(replaceRegEx.ReplaceAllString(serverVersion.Minor, ""))
}
