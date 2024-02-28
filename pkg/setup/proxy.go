package setup

import (
	"github.com/loft-sh/vcluster/pkg/options"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/server"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

func StartProxy(ctx *options.ControllerContext, controlPlaneNamespace, controlPlaneService string, controlPlaneClient kubernetes.Interface) error {
	// add remote node port sans
	err := pro.AddRemoteNodePortSANs(ctx.Context, controlPlaneNamespace, controlPlaneService, controlPlaneClient)
	if err != nil {
		return err
	}

	// start the proxy
	proxyServer, err := server.NewServer(ctx, ctx.Options.RequestHeaderCaCert, ctx.Options.ClientCaCert)
	if err != nil {
		return err
	}

	// start the proxy server in secure mode
	go func() {
		err = proxyServer.ServeOnListenerTLS(ctx.Options.BindAddress, ctx.Options.Port, ctx.StopChan)
		if err != nil {
			klog.Fatalf("Error serving: %v", err)
		}
	}()

	return nil
}
