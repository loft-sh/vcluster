package setup

import (
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/server"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"k8s.io/klog/v2"
)

func StartProxy(ctx *synccontext.ControllerContext) error {
	// add remote node port sans
	if ctx.Config.Experimental.IsolatedControlPlane.Enabled {
		err := pro.AddRemoteNodePortSANs(ctx, ctx.Config.ControlPlaneNamespace, ctx.Config.ControlPlaneService, ctx.Config.ControlPlaneClient)
		if err != nil {
			return err
		}
	}

	// start the proxy
	proxyServer, err := server.NewServer(ctx)
	if err != nil {
		return err
	}

	// start the proxy server in secure mode
	go func() {
		err = proxyServer.ServeOnListenerTLS(ctx)
		if err != nil {
			klog.Fatalf("Error serving: %v", err)
		}
	}()

	return nil
}
