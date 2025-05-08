package config

import (
	"fmt"
	"strings"
)

const (
	K3SDistro = "k3s"
	K8SDistro = "k8s"
	Unknown   = "unknown"
)

type StoreType string

const (
	StoreTypeEmbeddedEtcd     StoreType = "embedded-etcd"
	StoreTypeExternalEtcd     StoreType = "external-etcd"
	StoreTypeDeployedEtcd     StoreType = "deployed-etcd"
	StoreTypeEmbeddedDatabase StoreType = "embedded-database"
	StoreTypeExternalDatabase StoreType = "external-database"
)

// K3SVersionMap holds the supported k3s versions
var K3SVersionMap = map[string]string{
	"1.32": "rancher/k3s:v1.32.1-k3s1",
	"1.31": "rancher/k3s:v1.31.1-k3s1",
	"1.30": "rancher/k3s:v1.30.2-k3s1",
}

// K8SVersionMap holds the supported k8s api servers
var K8SVersionMap = map[string]string{
	"1.32": "ghcr.io/loft-sh/kubernetes:v1.32.1",
	"1.31": "ghcr.io/loft-sh/kubernetes:v1.31.1",
	"1.30": "ghcr.io/loft-sh/kubernetes:v1.30.2",
}

// K8SEtcdVersionMap holds the supported etcd
var K8SEtcdVersionMap = map[string]string{
	"1.32": "registry.k8s.io/etcd:3.5.21-0",
	"1.31": "registry.k8s.io/etcd:3.5.15-0",
	"1.30": "registry.k8s.io/etcd:3.5.13-0",
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
		case K8SDistro:
		}
	}
}
