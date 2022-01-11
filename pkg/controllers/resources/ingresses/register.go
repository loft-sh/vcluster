package ingresses

import (
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/ingresses/legacy"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
)

func Register(ctx *context2.ControllerContext, eventBroadcaster record.EventBroadcaster) error {
	useLegacy, err := ShouldUseLegacy(ctx.LocalManager.GetConfig())
	if err != nil {
		return err
	}

	if useLegacy {
		klog.Infof("Registering legacy ingress syncer for networking.k8s.io/v1beta1")
		return legacy.RegisterSyncer(ctx, eventBroadcaster)
	}
	return RegisterSyncer(ctx, eventBroadcaster)
}

func ShouldUseLegacy(config *rest.Config) (bool, error) {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return false, err
	}

	resources, err := discoveryClient.ServerResourcesForGroupVersion("networking.k8s.io/v1")
	if err != nil {
		if kerrors.IsNotFound(err) {
			return true, nil
		}

		return false, err
	}

	for _, r := range resources.APIResources {
		if r.Kind == "Ingress" {
			return false, nil
		}
	}

	return true, nil
}
