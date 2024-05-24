package pro

import (
	"context"

	"github.com/loft-sh/admin-apis/pkg/licenseapi"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var GetRemoteClient = func(vConfig *config.VirtualClusterConfig) (*rest.Config, string, string, *rest.Config, string, string, error) {
	inClusterConfig := ctrl.GetConfigOrDie()
	inClusterConfig.QPS = 40
	inClusterConfig.Burst = 80
	inClusterConfig.Timeout = 0

	// get current namespace
	currentNamespace, err := clienthelper.CurrentNamespace()
	if err != nil {
		return nil, "", "", nil, "", "", err
	}

	// check if remote cluster
	if vConfig.Experimental.IsolatedControlPlane.Enabled {
		return nil, "", "", nil, "", "", NewFeatureError(string(licenseapi.VirtualClusterProDistroIsolatedControlPlane))
	}

	return inClusterConfig, currentNamespace, vConfig.ControlPlaneService, inClusterConfig, currentNamespace, vConfig.ControlPlaneService, nil
}

var AddRemoteNodePortSANs = func(_ context.Context, _, _ string, _ kubernetes.Interface) error {
	return nil
}

var ExchangeControlPlaneClient = func(controllerCtx *config.ControllerContext) (client.Client, error) {
	return controllerCtx.WorkloadNamespaceClient, nil
}

var SyncRemoteEndpoints = func(_ context.Context, _ types.NamespacedName, _ client.Client, _ types.NamespacedName, _ client.Client) error {
	return NewFeatureError(string(licenseapi.VirtualClusterProDistroIsolatedControlPlane))
}
