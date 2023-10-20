package ingresses

import (
	"github.com/loft-sh/vcluster/pkg/controllers/resources/ingresses/legacy"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	syncer "github.com/loft-sh/vcluster/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

func New(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	useLegacy, err := ShouldUseLegacy(ctx.PhysicalManager.GetConfig())
	if err != nil {
		return nil, err
	}

	if useLegacy {
		klog.Infof("Registering legacy ingress syncer for networking.k8s.io/v1beta1")
		return legacy.NewSyncer(ctx)
	}
	return NewSyncer(ctx)
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
