package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	K3SDistro = "k3s"
	K8SDistro = "k8s"
	K0SDistro = "k0s"
	EKSDistro = "eks"
	Unknown   = "unknown"
)

// K3SVersionMap holds the supported k3s versions
var K3SVersionMap = map[string]string{
	"1.29": "rancher/k3s:v1.29.0-k3s1",
	"1.28": "rancher/k3s:v1.28.5-k3s1",
	"1.27": "rancher/k3s:v1.27.9-k3s1",
	"1.26": "rancher/k3s:v1.26.12-k3s1",
}

// K0SVersionMap holds the supported k0s versions
var K0SVersionMap = map[string]string{
	"1.29": "k0sproject/k0s:v1.29.1-k0s.0",
	"1.28": "k0sproject/k0s:v1.28.2-k0s.0",
	"1.27": "k0sproject/k0s:v1.27.6-k0s.0",
	"1.26": "k0sproject/k0s:v1.26.9-k0s.0",
}

// K8SAPIVersionMap holds the supported k8s api servers
var K8SAPIVersionMap = map[string]string{
	"1.29": "registry.k8s.io/kube-apiserver:v1.29.0",
	"1.28": "registry.k8s.io/kube-apiserver:v1.28.4",
	"1.27": "registry.k8s.io/kube-apiserver:v1.27.8",
	"1.26": "registry.k8s.io/kube-apiserver:v1.26.11",
}

// K8SControllerVersionMap holds the supported k8s controller managers
var K8SControllerVersionMap = map[string]string{
	"1.29": "registry.k8s.io/kube-controller-manager:v1.29.0",
	"1.28": "registry.k8s.io/kube-controller-manager:v1.28.4",
	"1.27": "registry.k8s.io/kube-controller-manager:v1.27.8",
	"1.26": "registry.k8s.io/kube-controller-manager:v1.26.11",
}

// K8SSchedulerVersionMap holds the supported k8s schedulers
var K8SSchedulerVersionMap = map[string]string{
	"1.29": "registry.k8s.io/kube-scheduler:v1.29.0",
	"1.28": "registry.k8s.io/kube-scheduler:v1.28.4",
	"1.27": "registry.k8s.io/kube-scheduler:v1.27.8",
	"1.26": "registry.k8s.io/kube-scheduler:v1.26.11",
}

// K8SEtcdVersionMap holds the supported etcd
var K8SEtcdVersionMap = map[string]string{
	"1.29": "registry.k8s.io/etcd:3.5.10-0",
	"1.28": "registry.k8s.io/etcd:3.5.9-0",
	"1.27": "registry.k8s.io/etcd:3.5.7-0",
	"1.26": "registry.k8s.io/etcd:3.5.6-0",
}

// EKSAPIVersionMap holds the supported eks api servers
var EKSAPIVersionMap = map[string]string{
	"1.28": "public.ecr.aws/eks-distro/kubernetes/kube-apiserver:v1.28.2-eks-1-28-6",
	"1.27": "public.ecr.aws/eks-distro/kubernetes/kube-apiserver:v1.27.6-eks-1-27-13",
	"1.26": "public.ecr.aws/eks-distro/kubernetes/kube-apiserver:v1.26.9-eks-1-26-19",
	"1.25": "public.ecr.aws/eks-distro/kubernetes/kube-apiserver:v1.25.14-eks-1-25-23",
}

// EKSControllerVersionMap holds the supported eks controller managers
var EKSControllerVersionMap = map[string]string{
	"1.28": "public.ecr.aws/eks-distro/kubernetes/kube-controller-manager:v1.28.2-eks-1-28-6",
	"1.27": "public.ecr.aws/eks-distro/kubernetes/kube-controller-manager:v1.27.6-eks-1-27-13",
	"1.26": "public.ecr.aws/eks-distro/kubernetes/kube-controller-manager:v1.26.9-eks-1-26-19",
	"1.25": "public.ecr.aws/eks-distro/kubernetes/kube-controller-manager:v1.25.14-eks-1-25-23",
}

// EKSSchedulerVersionMap holds the supported eks controller managers
var EKSSchedulerVersionMap = map[string]string{
	"1.28": "public.ecr.aws/eks-distro/kubernetes/kube-scheduler:v1.28.2-eks-1-28-6",
	"1.27": "public.ecr.aws/eks-distro/kubernetes/kube-scheduler:v1.27.6-eks-1-27-13",
	"1.26": "public.ecr.aws/eks-distro/kubernetes/kube-scheduler:v1.26.9-eks-1-26-19",
	"1.25": "public.ecr.aws/eks-distro/kubernetes/kube-scheduler:v1.25.14-eks-1-25-23",
}

// EKSEtcdVersionMap holds the supported eks etcd
var EKSEtcdVersionMap = map[string]string{
	"1.28": "public.ecr.aws/eks-distro/etcd-io/etcd:v3.5.9-eks-1-28-6",
	"1.27": "public.ecr.aws/eks-distro/etcd-io/etcd:v3.5.8-eks-1-27-13",
	"1.26": "public.ecr.aws/eks-distro/etcd-io/etcd:v3.5.8-eks-1-26-19",
	"1.25": "public.ecr.aws/eks-distro/etcd-io/etcd:v3.5.8-eks-1-25-23",
}

// EKSCoreDNSVersionMap holds the supported eks core dns
var EKSCoreDNSVersionMap = map[string]string{
	"1.28": "public.ecr.aws/eks-distro/coredns/coredns:v1.10.1-eks-1-28-6",
	"1.27": "public.ecr.aws/eks-distro/coredns/coredns:v1.10.1-eks-1-27-13",
	"1.26": "public.ecr.aws/eks-distro/coredns/coredns:v1.9.3-eks-1-26-19",
	"1.25": "public.ecr.aws/eks-distro/coredns/coredns:v1.9.3-eks-1-25-23",
}

// ExtraValuesOptions holds the chart options
type ExtraValuesOptions struct {
	Distro string

	Expose            bool
	NodePort          bool
	SyncNodes         bool
	KubernetesVersion KubernetesVersion

	DisableTelemetry    bool
	InstanceCreatorType string
	MachineID           string
	PlatformInstanceID  string
	PlatformUserID      string
}

type Logger interface {
	Info(msg string, keysAndValues ...any)
}

type KubernetesVersion struct {
	Major string
	Minor string
}

func GetExtraValues(options *ExtraValuesOptions, log Logger) (string, error) {
	fromConfig, err := NewDefaultConfig()
	if err != nil {
		return "", err
	}

	toConfig, err := getExtraValues(options, log)
	if err != nil {
		return "", fmt.Errorf("get extra values: %w", err)
	}

	return Diff(fromConfig, toConfig)
}

func getExtraValues(options *ExtraValuesOptions, log Logger) (*Config, error) {
	vConfig, err := NewDefaultConfig()
	if err != nil {
		return nil, err
	}

	switch options.Distro {
	case K3SDistro:
		return getK3SExtraValues(vConfig, options, log)
	case K0SDistro:
		return getK0SExtraValues(vConfig, options, log)
	case K8SDistro:
		return getK8SExtraValues(vConfig, options, log)
	case EKSDistro:
		return getEKSExtraValues(vConfig, options, log)
	}

	return vConfig, nil
}

var replaceRegEx = regexp.MustCompile("[^0-9]+")

func getK3SExtraValues(vConfig *Config, options *ExtraValuesOptions, log Logger) (*Config, error) {
	// get k3s image
	image, err := getImageByVersion(options.KubernetesVersion, K3SVersionMap, log)
	if err != nil {
		return nil, err
	}

	// build values
	vConfig.ControlPlane.Distro.K3S.Enabled = true
	if image != "" {
		vConfig.ControlPlane.Distro.K3S.Image = parseImage(image)
	}

	// add common release values
	addCommonReleaseValues(vConfig, options)
	return vConfig, nil
}

func getK0SExtraValues(vConfig *Config, options *ExtraValuesOptions, log Logger) (*Config, error) {
	// get k0s image
	image, err := getImageByVersion(options.KubernetesVersion, K0SVersionMap, log)
	if err != nil {
		return nil, err
	}

	// build values
	vConfig.ControlPlane.Distro.K0S.Enabled = true
	if image != "" {
		vConfig.ControlPlane.Distro.K0S.Image = parseImage(image)
	}

	// add common release values
	addCommonReleaseValues(vConfig, options)
	return vConfig, nil
}

func getEKSExtraValues(vConfig *Config, options *ExtraValuesOptions, log Logger) (*Config, error) {
	// get api server image
	apiImage, err := getImageByVersion(options.KubernetesVersion, EKSAPIVersionMap, log)
	if err != nil {
		return nil, err
	}

	// get controller image
	controllerImage, err := getImageByVersion(options.KubernetesVersion, EKSControllerVersionMap, log)
	if err != nil {
		return nil, err
	}

	// get scheduler image
	schedulerImage, err := getImageByVersion(options.KubernetesVersion, EKSSchedulerVersionMap, log)
	if err != nil {
		return nil, err
	}

	// get etcd image
	etcdImage, err := getImageByVersion(options.KubernetesVersion, EKSEtcdVersionMap, log)
	if err != nil {
		return nil, err
	}

	// get coredns image
	coreDNSImage, err := getImageByVersion(options.KubernetesVersion, EKSCoreDNSVersionMap, log)
	if err != nil {
		return nil, err
	}

	// build values
	vConfig.ControlPlane.Distro.EKS.Enabled = true
	if apiImage != "" {
		vConfig.ControlPlane.Distro.EKS.APIServer.Image = parseImage(apiImage)
	}
	if controllerImage != "" {
		vConfig.ControlPlane.Distro.EKS.ControllerManager.Image = parseImage(controllerImage)
	}
	if schedulerImage != "" {
		vConfig.ControlPlane.Distro.EKS.Scheduler.Image = parseImage(schedulerImage)
	}
	if etcdImage != "" {
		vConfig.ControlPlane.BackingStore.Etcd.Deploy.StatefulSet.Image = parseImage(etcdImage)
	}
	if coreDNSImage != "" {
		vConfig.ControlPlane.CoreDNS.Deployment.Image = coreDNSImage
	}

	addCommonReleaseValues(vConfig, options)
	return vConfig, nil
}

func getK8SExtraValues(vConfig *Config, options *ExtraValuesOptions, log Logger) (*Config, error) {
	// get api server image
	apiImage, err := getImageByVersion(options.KubernetesVersion, K8SAPIVersionMap, log)
	if err != nil {
		return nil, err
	}

	// get controller image
	controllerImage, err := getImageByVersion(options.KubernetesVersion, K8SControllerVersionMap, log)
	if err != nil {
		return nil, err
	}

	// get scheduler image
	schedulerImage, err := getImageByVersion(options.KubernetesVersion, K8SSchedulerVersionMap, log)
	if err != nil {
		return nil, err
	}

	// get etcd image
	etcdImage, err := getImageByVersion(options.KubernetesVersion, K8SEtcdVersionMap, log)
	if err != nil {
		return nil, err
	}

	// build values
	if apiImage != "" {
		vConfig.ControlPlane.Distro.K8S.APIServer.Image = parseImage(apiImage)
	}
	if controllerImage != "" {
		vConfig.ControlPlane.Distro.K8S.ControllerManager.Image = parseImage(controllerImage)
	}
	if schedulerImage != "" {
		vConfig.ControlPlane.Distro.K8S.Scheduler.Image = parseImage(schedulerImage)
	}
	if etcdImage != "" {
		vConfig.ControlPlane.BackingStore.Etcd.Deploy.StatefulSet.Image = parseImage(etcdImage)
	}

	addCommonReleaseValues(vConfig, options)
	return vConfig, nil
}

func parseImage(image string) Image {
	splitTag := strings.SplitN(image, ":", 2)
	if len(splitTag) == 2 {
		return Image{
			Repository: splitTag[0],
			Tag:        splitTag[1],
		}
	}

	return Image{}
}

func getImageByVersion(kubernetesVersion KubernetesVersion, versionImageMap map[string]string, log Logger) (string, error) {
	// check if there is a minor and major version
	if kubernetesVersion.Minor == "" || kubernetesVersion.Major == "" {
		return "", nil
	}

	// find highest and lowest supported version for this map
	highestMinorVersion := 0
	lowestMinorVersion := 0
	for version := range versionImageMap {
		kubeVersion, err := ParseKubernetesVersionInfo(version)
		if err != nil {
			return "", fmt.Errorf("parse kube version %s: %w", version, err)
		}

		minorVersion, err := strconv.Atoi(kubeVersion.Minor)
		if err != nil {
			return "", fmt.Errorf("convert minor version %s: %w", kubeVersion.Minor, err)
		}

		if lowestMinorVersion == 0 || minorVersion < lowestMinorVersion {
			lowestMinorVersion = minorVersion
		}
		if highestMinorVersion == 0 || minorVersion > highestMinorVersion {
			highestMinorVersion = minorVersion
		}
	}

	// figure out what image to use
	serverVersionString := getKubernetesVersion(kubernetesVersion)
	serverMinorInt, err := getKubernetesMinorVersion(kubernetesVersion)
	if err != nil {
		return "", err
	}

	// try to get from map
	image, ok := versionImageMap[serverVersionString]
	if !ok {
		if serverMinorInt > highestMinorVersion {
			log.Info(fmt.Sprintf("officially unsupported host server version %s, will fallback to virtual cluster version v1.%d", serverVersionString, highestMinorVersion))
			image = versionImageMap["1."+strconv.Itoa(highestMinorVersion)]
		} else {
			log.Info(fmt.Sprintf("officially unsupported host server version %s, will fallback to virtual cluster version v1.%d", serverVersionString, lowestMinorVersion))
			image = versionImageMap["1."+strconv.Itoa(lowestMinorVersion)]
		}
	}

	return image, nil
}

func addCommonReleaseValues(config *Config, options *ExtraValuesOptions) {
	if options.Expose {
		if config.ControlPlane.Service.Spec == nil {
			config.ControlPlane.Service.Spec = map[string]interface{}{}
		}

		config.ControlPlane.Service.Spec["type"] = "LoadBalancer"
	} else if options.NodePort {
		if config.ControlPlane.Service.Spec == nil {
			config.ControlPlane.Service.Spec = map[string]interface{}{}
		}

		config.ControlPlane.Service.Spec["type"] = "NodePort"
	}

	if options.SyncNodes {
		config.Sync.FromHost.Nodes.Enabled = true
	}

	if options.DisableTelemetry {
		config.Telemetry.Enabled = false
	} else if options.InstanceCreatorType != "" {
		config.Telemetry.InstanceCreator = options.InstanceCreatorType
		config.Telemetry.PlatformUserID = options.PlatformUserID
		config.Telemetry.PlatformInstanceID = options.PlatformInstanceID
		config.Telemetry.MachineID = options.MachineID
	}
}

func getKubernetesVersion(serverVersion KubernetesVersion) string {
	return replaceRegEx.ReplaceAllString(serverVersion.Major, "") + "." + replaceRegEx.ReplaceAllString(serverVersion.Minor, "")
}

func getKubernetesMinorVersion(serverVersion KubernetesVersion) (int, error) {
	return strconv.Atoi(replaceRegEx.ReplaceAllString(serverVersion.Minor, ""))
}

func ParseKubernetesVersionInfo(versionStr string) (*KubernetesVersion, error) {
	if versionStr[0] == 'v' {
		versionStr = versionStr[1:]
	}

	splittedVersion := strings.Split(versionStr, ".")
	if len(splittedVersion) != 2 && len(splittedVersion) != 3 {
		return nil, fmt.Errorf("unrecognized kubernetes version %s, please use format vX.X", versionStr)
	}

	major := splittedVersion[0]
	minor := splittedVersion[1]
	return &KubernetesVersion{
		Major: major,
		Minor: minor,
	}, nil
}
