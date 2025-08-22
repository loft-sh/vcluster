package snapshot

import (
	"fmt"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
)

func ValidateConfigAndOptions(vConfig *config.VirtualClusterConfig, options *Options, isRestore, isList bool) error {
	// storage needs to be either s3 or file
	err := Validate(options, isList)
	if err != nil {
		return err
	}

	// only support k3s and k8s distro
	if isRestore && vConfig.Distro() != vclusterconfig.K8SDistro && vConfig.Distro() != vclusterconfig.K3SDistro {
		return fmt.Errorf("unsupported distro: %s", vConfig.Distro())
	}

	return nil
}
