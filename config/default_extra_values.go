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
	Unknown   = "unknown"
)

type StoreType string

const (
	StoreTypeEmbeddedEtcd     StoreType = "embedded-etcd"
	StoreTypeExternalEtcd     StoreType = "external-etcd"
	StoreTypeEmbeddedDatabase StoreType = "embedded-database"
	StoreTypeExternalDatabase StoreType = "external-database"
)

// K3SVersionMap holds the supported k3s versions
var K3SVersionMap = map[string]string{
	"1.30": "rancher/k3s:v1.30.2-k3s1",
	"1.29": "rancher/k3s:v1.29.6-k3s1",
	"1.28": "rancher/k3s:v1.28.11-k3s1",
	"1.27": "rancher/k3s:v1.27.15-k3s1",
}

// K0SVersionMap holds the supported k0s versions
var K0SVersionMap = map[string]string{
	"1.30": "k0sproject/k0s:v1.30.2-k0s.0",
	"1.29": "k0sproject/k0s:v1.29.6-k0s.0",
	"1.28": "k0sproject/k0s:v1.28.11-k0s.0",
	"1.27": "k0sproject/k0s:v1.27.15-k0s.0",
}

// K8SAPIVersionMap holds the supported k8s api servers
var K8SAPIVersionMap = map[string]string{
	"1.30": "registry.k8s.io/kube-apiserver:v1.30.2",
	"1.29": "registry.k8s.io/kube-apiserver:v1.29.6",
	"1.28": "registry.k8s.io/kube-apiserver:v1.28.11",
	"1.27": "registry.k8s.io/kube-apiserver:v1.27.15",
}

// K8SControllerVersionMap holds the supported k8s controller managers
var K8SControllerVersionMap = map[string]string{
	"1.30": "registry.k8s.io/kube-controller-manager:v1.30.2",
	"1.29": "registry.k8s.io/kube-controller-manager:v1.29.6",
	"1.28": "registry.k8s.io/kube-controller-manager:v1.28.11",
	"1.27": "registry.k8s.io/kube-controller-manager:v1.27.15",
}

// K8SSchedulerVersionMap holds the supported k8s schedulers
var K8SSchedulerVersionMap = map[string]string{
	"1.30": "registry.k8s.io/kube-scheduler:v1.30.2",
	"1.29": "registry.k8s.io/kube-scheduler:v1.29.6",
	"1.28": "registry.k8s.io/kube-scheduler:v1.28.11",
	"1.27": "registry.k8s.io/kube-scheduler:v1.27.15",
}

// K8SEtcdVersionMap holds the supported etcd
var K8SEtcdVersionMap = map[string]string{
	"1.30": "registry.k8s.io/etcd:3.5.13-0",
	"1.29": "registry.k8s.io/etcd:3.5.10-0",
	"1.28": "registry.k8s.io/etcd:3.5.9-0",
	"1.27": "registry.k8s.io/etcd:3.5.7-0",
}

// ExtraValuesOptions holds the chart options
type ExtraValuesOptions struct {
	Distro string

	Expose            bool
	NodePort          bool
	KubernetesVersion KubernetesVersion

	DisableTelemetry    bool
	InstanceCreatorType string
	MachineID           string
	PlatformInstanceID  string
	PlatformUserID      string
}

type KubernetesVersion struct {
	Major string
	Minor string
}

func GetExtraValues(options *ExtraValuesOptions) (string, error) {
	fromConfig, err := NewDefaultConfig()
	if err != nil {
		return "", err
	}

	toConfig, err := getExtraValues(options)
	if err != nil {
		return "", fmt.Errorf("get extra values: %w", err)
	}

	return Diff(fromConfig, toConfig)
}

func getExtraValues(options *ExtraValuesOptions) (*Config, error) {
	vConfig, err := NewDefaultConfig()
	if err != nil {
		return nil, err
	}

	// apply k3s values
	err = applyK3SExtraValues(vConfig, options)
	if err != nil {
		return nil, err
	}

	// apply k0s values
	err = applyK0SExtraValues(vConfig, options)
	if err != nil {
		return nil, err
	}

	// apply k8s values
	err = applyK8SExtraValues(vConfig, options)
	if err != nil {
		return nil, err
	}

	// add common release values
	addCommonReleaseValues(vConfig, options)
	return vConfig, nil
}

var replaceRegEx = regexp.MustCompile("[^0-9]+")

func applyK3SExtraValues(vConfig *Config, options *ExtraValuesOptions) error {
	// get k3s image
	image, err := getImageByVersion(options.KubernetesVersion, K3SVersionMap)
	if err != nil {
		return err
	}

	// build values
	if image != "" {
		vConfig.ControlPlane.Distro.K3S.Image = parseImage(image)
	}

	return nil
}

func applyK0SExtraValues(vConfig *Config, options *ExtraValuesOptions) error {
	// get k0s image
	image, err := getImageByVersion(options.KubernetesVersion, K0SVersionMap)
	if err != nil {
		return err
	}

	// build values
	if image != "" {
		vConfig.ControlPlane.Distro.K0S.Image = parseImage(image)
	}

	return nil
}

func applyK8SExtraValues(vConfig *Config, options *ExtraValuesOptions) error {
	// get api server image
	apiImage, err := getImageByVersion(options.KubernetesVersion, K8SAPIVersionMap)
	if err != nil {
		return err
	}

	// get etcd image
	etcdImage, err := getImageByVersion(options.KubernetesVersion, K8SEtcdVersionMap)
	if err != nil {
		return err
	}

	// build values
	if apiImage != "" {
		vConfig.ControlPlane.Distro.K8S.Version = parseImage(apiImage).Tag
	}
	if etcdImage != "" {
		vConfig.ControlPlane.BackingStore.Etcd.Deploy.StatefulSet.Image = parseImage(etcdImage)
	}

	return nil
}

func parseImage(image string) Image {
	registry, repository, tag := SplitImage(image)
	return Image{
		Registry:   registry,
		Repository: repository,
		Tag:        tag,
	}
}

func SplitImage(image string) (string, string, string) {
	imageSplitted := strings.Split(image, ":")
	if len(imageSplitted) == 1 {
		return "", "", ""
	}

	// check if registry needs to be filled
	registryAndRepository := strings.Join(imageSplitted[:len(imageSplitted)-1], ":")
	parts := strings.Split(registryAndRepository, "/")
	registry := ""
	repository := strings.Join(parts, "/")
	if len(parts) >= 2 && (strings.ContainsRune(parts[0], '.') || strings.ContainsRune(parts[0], ':')) {
		// The first part of the repository is treated as the registry domain
		// iff it contains a '.' or ':' character, otherwise it is all repository
		// and the domain defaults to Docker Hub.
		registry = parts[0]
		repository = strings.Join(parts[1:], "/")
	}

	return registry, repository, imageSplitted[len(imageSplitted)-1]
}

func getImageByVersion(kubernetesVersion KubernetesVersion, versionImageMap map[string]string) (string, error) {
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
			image = versionImageMap["1."+strconv.Itoa(highestMinorVersion)]
		} else {
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

	if options.DisableTelemetry {
		config.Telemetry.Enabled = false
	} else if options.InstanceCreatorType != "" {
		config.Telemetry.InstanceCreator = options.InstanceCreatorType
		config.Telemetry.PlatformUserID = options.PlatformUserID
		config.Telemetry.PlatformInstanceID = options.PlatformInstanceID
		config.Telemetry.MachineID = options.MachineID
	}

	if options.Distro != "" {
		switch options.Distro {
		case K3SDistro:
			config.ControlPlane.Distro.K3S.Enabled = true
		case K0SDistro:
			config.ControlPlane.Distro.K0S.Enabled = true
		case K8SDistro:
		}
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
