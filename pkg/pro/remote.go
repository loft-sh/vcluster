package pro

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/loft-sh/admin-apis/pkg/licenseapi"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var GetRemoteClient = func(vConfig *config.VirtualClusterConfig) (*rest.Config, string, string, *rest.Config, string, string, error) {
	inClusterConfig := ctrl.GetConfigOrDie()

	// Get QPS from environment variable or default to 40
	qpsStr := os.Getenv("K8S_CLIENT_QPS")
	qps, err := strconv.ParseFloat(qpsStr, 32)
	if err != nil || qpsStr == "" {
		qps = 40
	}
	inClusterConfig.QPS = float32(qps)

	// Get Burst from environment variable or default to 80
	burstStr := os.Getenv("K8S_CLIENT_BURST")
	burst, err := strconv.Atoi(burstStr)
	if err != nil || burstStr == "" {
		burst = 80
	}
	inClusterConfig.Burst = burst

	// Get Timeout from environment variable or default to 0
	timeoutStr := os.Getenv("K8S_CLIENT_TIMEOUT")
	timeout, err := strconv.Atoi(timeoutStr)
	if err != nil || timeoutStr == "" {
		timeout = 0
	}
	inClusterConfig.Timeout = time.Duration(timeout) * time.Second

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

var ExchangeControlPlaneClient = func(controllerCtx *synccontext.ControllerContext) (client.Client, error) {
	return controllerCtx.WorkloadNamespaceClient, nil
}

var SyncRemoteEndpoints = func(_ context.Context, _ types.NamespacedName, _ client.Client, _ types.NamespacedName, _ client.Client) error {
	return NewFeatureError(string(licenseapi.VirtualClusterProDistroIsolatedControlPlane))
}
