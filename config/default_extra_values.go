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
	"1.33": "rancher/k3s:v1.33.1-k3s1",
	"1.32": "rancher/k3s:v1.32.1-k3s1",
	"1.31": "rancher/k3s:v1.31.1-k3s1",
	"1.30": "rancher/k3s:v1.30.2-k3s1",
}

// K8SVersionMap holds the supported k8s api servers
var K8SVersionMap = map[string]string{
	"1.33": "ghcr.io/loft-sh/kubernetes:v1.33.1",
	"1.32": "ghcr.io/loft-sh/kubernetes:v1.32.1",
	"1.31": "ghcr.io/loft-sh/kubernetes:v1.31.1",
	"1.30": "ghcr.io/loft-sh/kubernetes:v1.30.2",
}

// K8SEtcdVersionMap holds the supported etcd
var K8SEtcdVersionMap = map[string]string{
	"1.33": "registry.k8s.io/etcd:3.5.21-0",
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

func ParseImageRef(ref string, image *Image) {
	*image = Image{}

	splitRepoAndTag := func(s string) {
		split := strings.SplitN(s, ":", 2)
		switch len(split) {
		case 1:
			image.Repository = s
		case 2:
			image.Repository = split[0]
			image.Tag = split[1]
		}
		image.Repository = strings.TrimPrefix(image.Repository, "library/")
	}

	parts := strings.SplitN(ref, "/", 2)
	switch {
	case len(parts) == 1: // <repo>[:<tag>]
		splitRepoAndTag(parts[0])
	case strings.ContainsAny(parts[0], ".:"): // <registry>/<repo>[:<tag>]
		image.Registry = parts[0]
		splitRepoAndTag(parts[1])
	default: // <repo/repo>[:<tag]
		splitRepoAndTag(ref)
	}
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
