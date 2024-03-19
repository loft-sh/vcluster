package setup

import (
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/server"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

func StartProxy(ctx *config.ControllerContext, controlPlaneNamespace, controlPlaneService string, controlPlaneClient kubernetes.Interface) error {
	// add remote node port sans
	err := pro.AddRemoteNodePortSANs(ctx.Context, controlPlaneNamespace, controlPlaneService, controlPlaneClient)
	if err != nil {
		return err
	}

	// start the proxy
	proxyServer, err := server.NewServer(ctx, ctx.Config.VirtualClusterKubeConfig().RequestHeaderCACert, ctx.Config.VirtualClusterKubeConfig().ClientCACert)
	if err != nil {
		return err
	}

	// start the proxy server in secure mode
	go func() {
		err = proxyServer.ServeOnListenerTLS(ctx.Config.ControlPlane.Proxy.BindAddress, ctx.Config.ControlPlane.Proxy.Port, ctx.StopChan)
		if err != nil {
			klog.Fatalf("Error serving: %v", err)
		}
	}()

	return nil
}
