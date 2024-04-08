package setup

import (
	"fmt"
	"os"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/k3s"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/client-go/kubernetes"
)

func InitConfig(vConfig *config.VirtualClusterConfig) error {
	var err error

	// set global vCluster name
	translate.VClusterName = vConfig.Name

	// set workload namespace
	err = os.Setenv("NAMESPACE", vConfig.WorkloadNamespace)
	if err != nil {
		return fmt.Errorf("set NAMESPACE env var: %w", err)
	}

	// get host cluster client
	vConfig.ControlPlaneClient, err = kubernetes.NewForConfig(vConfig.ControlPlaneConfig)
	if err != nil {
		return err
	}

	// get workload client
	vConfig.WorkloadClient, err = kubernetes.NewForConfig(vConfig.WorkloadConfig)
	if err != nil {
		return err
	}

	// get workload target namespace
	if vConfig.Experimental.MultiNamespaceMode.Enabled {
		translate.Default = translate.NewMultiNamespaceTranslator(vConfig.WorkloadNamespace)
	} else {
		// ensure target namespace
		vConfig.WorkloadTargetNamespace = vConfig.Experimental.SyncSettings.TargetNamespace
		if vConfig.WorkloadTargetNamespace == "" {
			vConfig.WorkloadTargetNamespace = vConfig.WorkloadNamespace
		}

		translate.Default = translate.NewSingleNamespaceTranslator(vConfig.WorkloadTargetNamespace)
	}

	// check if previously we were using k3s as a default and now have switched to a different distro
	if vConfig.Distro() != vclusterconfig.K3SDistro {
		_, err := os.Stat(k3s.TokenPath)
		if err == nil {
			return fmt.Errorf("seems like you were using k3s as a distro before and now have switched to %s, please make sure to not switch between vCluster distros", vConfig.Distro())
		}
	}

	// check if previously we were using k0s as distro
	if vConfig.Distro() != vclusterconfig.K0SDistro {
		_, err = os.Stat("/data/k0s")
		if err == nil {
			return fmt.Errorf("seems like you were using k0s as a distro before and now have switched to %s, please make sure to not switch between vCluster distros", vConfig.Distro())
		}
	}

	return nil
}
