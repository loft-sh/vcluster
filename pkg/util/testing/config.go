package testing

import (
	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
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
		Config:                  *defaultConfig,
		Name:                    DefaultTestVClusterName,
		ControlPlaneService:     DefaultTestVClusterName,
		WorkloadService:         DefaultTestVClusterServiceName,
		WorkloadNamespace:       DefaultTestTargetNamespace,
		WorkloadTargetNamespace: DefaultTestTargetNamespace,
	}

	err = config.ValidateConfigAndSetDefaults(vConfig)
	if err != nil {
		panic(err.Error())
	}

	return vConfig
}
