package k8sdefaultendpoint

import (
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/discovery"
	"k8s.io/klog/v2"
)

func Register(ctx *synccontext.ControllerContext) error {
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
		provider = &EndpointsV1BetaProvider{}
	} else {
		provider = &EndpointsV1Provider{}
	}
	return NewEndpointController(ctx, provider).Register(ctx.HostManager)
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
