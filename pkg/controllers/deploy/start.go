package deploy

import (
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/loft-sh/vcluster/pkg/util/helmdownloader"
	"github.com/loft-sh/vcluster/pkg/util/kubeconfig"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"k8s.io/klog/v2"
)

func RegisterInitManifestsController(controllerCtx *config.ControllerContext) error {
	vConfig, err := kubeconfig.ConvertRestConfigToClientConfig(controllerCtx.VirtualManager.GetConfig())
	if err != nil {
		return err
	}

	vConfigRaw, err := vConfig.RawConfig()
	if err != nil {
		return err
	}

	helmBinaryPath, err := helmdownloader.GetHelmBinaryPath(controllerCtx.Context, log.GetInstance())
	if err != nil {
		return err
	}

	controller := &Deployer{
		Log:            loghelper.New("init-manifests-controller"),
		VirtualManager: controllerCtx.VirtualManager,

		HelmClient: helm.NewClient(&vConfigRaw, log.GetInstance(), helmBinaryPath),
	}

	go func() {
		for {
			result, err := controller.Apply(controllerCtx.Context, controllerCtx.Config)
			if err != nil {
				klog.Errorf("Error deploying manifests: %v", err)
				time.Sleep(time.Second * 10)
			} else if !result.Requeue {
				break
			}
		}
	}()

	return nil
}
