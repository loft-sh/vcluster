package deploy

import (
	"fmt"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/kubeconfig"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"k8s.io/klog/v2"
)

func RegisterInitManifestsController(controllerCtx *synccontext.ControllerContext) error {
	vConfig, err := kubeconfig.ConvertRestConfigToClientConfig(controllerCtx.VirtualManager.GetConfig())
	if err != nil {
		return err
	}

	vConfigRaw, err := vConfig.RawConfig()
	if err != nil {
		return err
	}

	deployer := &Deployer{
		Log:            loghelper.New("init-manifests-controller"),
		VirtualManager: controllerCtx.VirtualManager,

		HelmClient: helm.NewClient(&vConfigRaw, log.GetInstance(), constants.HelmBinary),
	}

	// deploy manifests
	err = deployer.DeployInitManifests(controllerCtx, controllerCtx.Config)
	if err != nil {
		return fmt.Errorf("error deploying experimental.deploy.vCluster.manifests: %w", err)
	}

	// deploy helm charts
	go func() {
		for {
			// deploy helm charts
			err := deployer.DeployHelmCharts(controllerCtx, controllerCtx.Config)
			if err != nil {
				klog.Errorf("Error deploying experimental.deploy.vCluster.helm: %v", err)
				time.Sleep(time.Second * 10)
				continue
			}

			// exit loop
			break
		}
	}()

	return nil
}
