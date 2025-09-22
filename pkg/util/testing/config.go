package testing

import (
	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"k8s.io/client-go/rest"
)

const (
	DefaultTestTargetNamespace     = "test"
	DefaultTestCurrentNamespace    = "vcluster"
	DefaultTestVClusterName        = "vcluster"
	DefaultTestVClusterServiceName = "vcluster"
)

func NewFakeConfig() *config.VirtualClusterConfig {
	// default config
	defaultConfig, err := vclusterconfig.NewDefaultConfig()
	if err != nil {
		panic(err.Error())
	}

	// parse config
	vConfig := &config.VirtualClusterConfig{
		Config:        *defaultConfig,
		Name:          DefaultTestVClusterName,
		HostNamespace: DefaultTestTargetNamespace,
	}

	err = config.ValidateConfigAndSetDefaults(vConfig)
	if err != nil {
		panic(err.Error())
	}

	// SyncController builder expects non-nil HostConfig
	vConfig.HostConfig = &rest.Config{
		Host:    "",
		APIPath: "",
	}
	return vConfig
}
