package values

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/loft-sh/vcluster/pkg/log"
	"k8s.io/apimachinery/pkg/version"
)

var K3SVersionMap = map[string]string{
	"1.24": "rancher/k3s:v1.24.3-k3s1",
	"1.23": "rancher/k3s:v1.23.9-k3s1",
	"1.22": "rancher/k3s:v1.22.12-k3s1",
	"1.21": "rancher/k3s:v1.21.14-k3s1",
	"1.20": "rancher/k3s:v1.20.15-k3s1",
	"1.19": "rancher/k3s:v1.19.16-k3s1",
	"1.18": "rancher/k3s:v1.18.20-k3s1",
	"1.17": "rancher/k3s:v1.17.17-k3s1",
	"1.16": "rancher/k3s:v1.16.15-k3s1",
}

const noDeployValues = `  baseArgs:
    - server
    - --write-kubeconfig=/k3s-config/kube-config.yaml
    - --data-dir=/data
    - --no-deploy=traefik,servicelb,metrics-server,local-storage
    - --disable-network-policy
    - --disable-agent
    - --disable-cloud-controller
    - --flannel-backend=none`

var baseArgsMap = map[string]string{
	"1.17": noDeployValues,
	"1.16": noDeployValues,
}

var replaceRegEx = regexp.MustCompile("[^0-9]+")

func getDefaultK3SReleaseValues(chartOptions *helm.ChartOptions, log log.Logger) (string, error) {
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
			if serverMinorInt > 24 {
				log.Infof("officially unsupported host server version %s, will fallback to virtual cluster version v1.24", serverVersionString)
				image = K3SVersionMap["1.24"]
				serverVersionString = "1.24"
			} else {
				log.Infof("officially unsupported host server version %s, will fallback to virtual cluster version v1.16", serverVersionString)
				image = K3SVersionMap["1.16"]
				serverVersionString = "1.16"
			}
		}
	}

	valuesString := ""
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
		valuesString += ",securityContext.runAsUser=12345,securityContext.runAsNonRoot=true"
	}

	values = strings.ReplaceAll(values, "##IMAGE##", image)
	valuesString += "vcluster.image=" + image
	if chartOptions.K3SImage == "" {
		baseArgs := baseArgsMap[serverVersionString]
		values = strings.ReplaceAll(values, "##BASEARGS##", baseArgs)
	}

	return addCommonReleaseValues(values, valuesString, chartOptions)
}

func addCommonReleaseValues(values string, valuesString string, chartOptions *helm.ChartOptions) (string, error) {
	if chartOptions.CIDR != "" {
		values += `
serviceCIDR: ##CIDR##`
		values = strings.ReplaceAll(values, "##CIDR##", chartOptions.CIDR)
		valuesString += ",serviceCIDR=" + chartOptions.CIDR
	}

	if chartOptions.DisableIngressSync {
		values += `
syncer:
  extraArgs: ["--disable-sync-resources=ingresses"]`
		valuesString += ",syncer.extraArgs=[\"--disable-sync-resources=ingresses\"]"
	}

	if chartOptions.CreateClusterRole {
		values += `
rbac:
  clusterRole:
    create: true`
		valuesString += ",rbac.clusterRole.create=true"
	}

	if chartOptions.Expose {
		values += `
service:
  type: LoadBalancer`
		valuesString += ",service.type=LoadBalancer"
	} else if chartOptions.NodePort {
		values += `
service:
  type: NodePort`
		valuesString += ",service.type=NodePort"
	}

	if chartOptions.SyncNodes {
		values += `
sync:
  nodes:
    enabled: true`
		valuesString += ",sync.nodes.enabled=true"
	}

	if chartOptions.Isolate {
		values += `
isolation:
  enabled: true`
		valuesString += ",isolation.enabled=true"
	}

	values = strings.TrimSpace(values)
	return valuesString, nil
}

func ParseKubernetesVersionInfo(versionStr string) (*version.Info, error) {
	if versionStr[0] == 'v' {
		versionStr = versionStr[1:]
	}

	splittedVersion := strings.Split(versionStr, ".")
	if len(splittedVersion) != 2 && len(splittedVersion) != 3 {
		return nil, fmt.Errorf("unrecognized kubernetes version %s, please use format vX.X", versionStr)
	}

	major := splittedVersion[0]
	minor := splittedVersion[1]

	return &version.Info{
		Major: major,
		Minor: minor,
	}, nil
}

func GetKubernetesVersion(serverVersion *version.Info) string {
	return replaceRegEx.ReplaceAllString(serverVersion.Major, "") + "." + replaceRegEx.ReplaceAllString(serverVersion.Minor, "")
}

func GetKubernetesMinorVersion(serverVersion *version.Info) (int, error) {
	return strconv.Atoi(replaceRegEx.ReplaceAllString(serverVersion.Minor, ""))
}
