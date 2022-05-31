package k8sdefaultendpoint

import (
	controllercontext "github.com/loft-sh/vcluster/cmd/vcluster/context"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/discovery"
	"k8s.io/klog"
)

func Register(ctx *controllercontext.ControllerContext) error {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(ctx.VirtualManager.GetConfig())
	if err != nil {
		return err
	}

	useLegacy, err := ShouldUseLegacy(discoveryClient)
	if err != nil {
		return err
	}

	var provider provider
	if useLegacy {
		klog.Infof("Registering legacy discovery endpoint for k8s.io/api/discovery/v1beta1")
		provider = &v1BetaProvider{}
	} else {
		provider = &v1Provider{}
	}
	return NewEndpointController(ctx, provider).Register(ctx.LocalManager)
}

func ShouldUseLegacy(discoveryClient discovery.DiscoveryInterface) (bool, error) {
	resources, err := discoveryClient.ServerResourcesForGroupVersion("discovery.k8s.io/v1")
	if err != nil {
		if kerrors.IsNotFound(err) {
			return true, nil
		}

		return false, err
	}

	for _, r := range resources.APIResources {
		if r.Kind == "EndpointSlice" {
			return false, nil
		}
	}

	return true, nil
}
