package config

import (
	"fmt"
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
	"1.31": "rancher/k3s:v1.31.1-k3s1",
	"1.30": "rancher/k3s:v1.30.2-k3s1",
	"1.29": "rancher/k3s:v1.29.6-k3s1",
	"1.28": "rancher/k3s:v1.28.11-k3s1",
	"1.27": "rancher/k3s:v1.27.16-k3s1",
}

// K0SVersionMap holds the supported k0s versions
var K0SVersionMap = map[string]string{
	"1.31": "k0sproject/k0s:v1.30.2-k0s.0",
	"1.30": "k0sproject/k0s:v1.30.2-k0s.0",
	"1.29": "k0sproject/k0s:v1.29.6-k0s.0",
	"1.28": "k0sproject/k0s:v1.28.11-k0s.0",
	"1.27": "k0sproject/k0s:v1.27.16-k0s.0",
}

// K8SAPIVersionMap holds the supported k8s api servers
var K8SAPIVersionMap = map[string]string{
	"1.31": "registry.k8s.io/kube-apiserver:v1.31.1",
	"1.30": "registry.k8s.io/kube-apiserver:v1.30.2",
	"1.29": "registry.k8s.io/kube-apiserver:v1.29.6",
	"1.28": "registry.k8s.io/kube-apiserver:v1.28.11",
	"1.27": "registry.k8s.io/kube-apiserver:v1.27.16",
}

// K8SControllerVersionMap holds the supported k8s controller managers
var K8SControllerVersionMap = map[string]string{
	"1.31": "registry.k8s.io/kube-controller-manager:v1.31.1",
	"1.30": "registry.k8s.io/kube-controller-manager:v1.30.2",
	"1.29": "registry.k8s.io/kube-controller-manager:v1.29.6",
	"1.28": "registry.k8s.io/kube-controller-manager:v1.28.11",
	"1.27": "registry.k8s.io/kube-controller-manager:v1.27.16",
}

// K8SSchedulerVersionMap holds the supported k8s schedulers
var K8SSchedulerVersionMap = map[string]string{
	"1.31": "registry.k8s.io/kube-scheduler:v1.31.1",
	"1.30": "registry.k8s.io/kube-scheduler:v1.30.2",
	"1.29": "registry.k8s.io/kube-scheduler:v1.29.6",
	"1.28": "registry.k8s.io/kube-scheduler:v1.28.11",
	"1.27": "registry.k8s.io/kube-scheduler:v1.27.16",
}

// K8SEtcdVersionMap holds the supported etcd
var K8SEtcdVersionMap = map[string]string{
	"1.31": "registry.k8s.io/etcd:3.5.15-0",
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

	// add common release values
	addCommonReleaseValues(vConfig, options)
	return vConfig, nil
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
