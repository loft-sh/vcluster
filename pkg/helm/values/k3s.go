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

var baseArgsSlice = []string{
	"1.17",
	"1.16",
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

	// build values
	var values []string
	values = append(values, "vcluster.image="+image)
	if chartOptions.Isolate {
		values = append(values, "securityContext.runAsUser=12345,securityContext.runAsNonRoot=true")
	}

	if chartOptions.K3SImage == "" {
		for _, a := range baseArgsSlice {
			if a == serverVersionString {
				values = append(values, "vcluster.baseArgs=server --write-kubeconfig=/k3s-config/kube-config.yaml --data-dir=/data --no-deploy=traefik,servicelb,metrics-server,local-storage --disable-network-policy --disable-agent --disable-cloud-controller --flannel-backend=none")
				break
			}
		}
	}

	return addCommonReleaseValues(values, chartOptions)
}

func addCommonReleaseValues(values []string, chartOptions *helm.ChartOptions) (string, error) {
	if chartOptions.CIDR != "" {
		values = append(values, "serviceCIDR="+chartOptions.CIDR)
	}

	if chartOptions.DisableIngressSync {
		values = append(values, "syncer.extraArgs=[\"--disable-sync-resources=ingresses\"]")
	}

	if chartOptions.CreateClusterRole {
		values = append(values, "rbac.clusterRole.create=true")
	}

	if chartOptions.Expose {
		values = append(values, "service.type=LoadBalancer")
	} else if chartOptions.NodePort {
		values = append(values, "service.type=NodePort")
	}

	if chartOptions.SyncNodes {
		values = append(values, "sync.nodes.enabled=true")
	}

	if chartOptions.Isolate {
		values = append(values, "isolation.enabled=true")
	}

	return strings.Join(values, ","), nil
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
