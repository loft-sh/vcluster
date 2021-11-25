package values

import (
	"context"
	"fmt"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/app/create"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"regexp"
	"strconv"
	"strings"
)

var K3SVersionMap = map[string]string{
	"1.22": "rancher/k3s:v1.22.2-k3s2",
	"1.21": "rancher/k3s:v1.21.5-k3s2",
	"1.20": "rancher/k3s:v1.20.11-k3s2",
	"1.19": "rancher/k3s:v1.19.13-k3s1",
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
    - --disable-scheduler
    - --disable-cloud-controller
    - --flannel-backend=none
    - --kube-controller-manager-arg=controllers=*,-nodeipam,-nodelifecycle,-persistentvolume-binder,-attachdetach,-persistentvolume-expander,-cloud-node-lifecycle`

var baseArgsMap = map[string]string{
	"1.17": noDeployValues,
	"1.16": noDeployValues,
}

var replaceRegEx = regexp.MustCompile("[^0-9]+")
var errorMessageFind = "provided IP is not in the valid range. The range of valid IPs is "

func getDefaultK3SReleaseValues(client kubernetes.Interface, createOptions *create.CreateOptions, log log.Logger) (string, error) {
	var (
		image               = createOptions.K3SImage
		serverVersionString string
		serverMinorInt      int
		err                 error
	)

	if image == "" {
		serverVersionString, serverMinorInt, err = getKubernetesVersion(client, createOptions)
		if err != nil {
			return "", err
		}

		var ok bool
		image, ok = K3SVersionMap[serverVersionString]
		if !ok {
			if serverMinorInt > 22 {
				log.Infof("officially unsupported host server version %s, will fallback to virtual cluster version v1.22", serverVersionString)
				image = K3SVersionMap["1.22"]
				serverVersionString = "1.22"
			} else {
				log.Infof("officially unsupported host server version %s, will fallback to virtual cluster version v1.16", serverVersionString)
				image = K3SVersionMap["1.16"]
				serverVersionString = "1.16"
			}
		}
	}

	// build values
	values := `vcluster:
  image: ##IMAGE##
##BASEARGS##
`
	values = strings.ReplaceAll(values, "##IMAGE##", image)
	if createOptions.K3SImage == "" {
		baseArgs := baseArgsMap[serverVersionString]
		values = strings.ReplaceAll(values, "##BASEARGS##", baseArgs)
	}

	return addCommonReleaseValues(values, createOptions)
}

func addCommonReleaseValues(values string, createOptions *create.CreateOptions) (string, error) {
	values += `
serviceCIDR: ##CIDR##
storage:
  size: 5Gi`
	if createOptions.DisableIngressSync {
		values += `
syncer:
  extraArgs: ["--disable-sync-resources=ingresses"]`
	}
	if createOptions.CreateClusterRole {
		values += `
rbac:
  clusterRole:
    create: true`
	}

	if createOptions.Expose {
		values += `
service:
  type: LoadBalancer`
	}

	values = strings.ReplaceAll(values, "##CIDR##", createOptions.CIDR)
	values = strings.TrimSpace(values)
	return values, nil
}

func getKubernetesVersion(client kubernetes.Interface, createOptions *create.CreateOptions) (string, int, error) {
	if createOptions.KubernetesVersion != "" {
		version := createOptions.KubernetesVersion
		if version[0] == 'v' {
			version = version[1:]
		}

		splittedVersion := strings.Split(version, ".")
		if len(splittedVersion) != 2 && len(splittedVersion) != 3 {
			return "", 0, fmt.Errorf("unrecognized kubernetes version %s, please use format vX.X", version)
		}

		minor := splittedVersion[1]
		minorParsed, err := strconv.Atoi(minor)
		if err != nil {
			return "", 0, errors.Wrap(err, "parse minor version")
		}

		return splittedVersion[0] + "." + splittedVersion[1], minorParsed, nil
	}

	serverVersion, err := client.Discovery().ServerVersion()
	if err != nil {
		return "", 0, err
	}

	serverVersionString := replaceRegEx.ReplaceAllString(serverVersion.Major, "") + "." + replaceRegEx.ReplaceAllString(serverVersion.Minor, "")
	serverMinorInt, err := strconv.Atoi(replaceRegEx.ReplaceAllString(serverVersion.Minor, ""))
	if err != nil {
		return "", 0, err
	}

	return serverVersionString, serverMinorInt, nil
}

func GetServiceCIDR(client kubernetes.Interface, namespace string, ipv6 bool) (string, error) {
	clusterIP := "4.4.4.4"
	if ipv6 {
		// https://www.ietf.org/rfc/rfc3849.txt
		clusterIP = "2001:DB8::1"
	}
	_, err := client.CoreV1().Services(namespace).Create(context.Background(), &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-service-",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port: 80,
				},
			},
			ClusterIP: clusterIP,
		},
	}, metav1.CreateOptions{})
	if err == nil {
		return "", fmt.Errorf("couldn't find cluster service cidr, will fallback to 10.96.0.0/12, however this is probably wrong, please make sure the host cluster service cidr and virtual cluster service cidr match")
	}

	errorMessage := err.Error()
	idx := strings.Index(errorMessage, errorMessageFind)
	if idx == -1 {
		return "", fmt.Errorf("couldn't find cluster service cidr (" + errorMessage + "), will fallback to 10.96.0.0/12, however this is probably wrong, please make sure the host cluster service cidr and virtual cluster service cidr match")
	}

	return strings.TrimSpace(errorMessage[idx+len(errorMessageFind):]), nil
}
