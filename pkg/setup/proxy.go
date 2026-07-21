package setup

import (
	"github.com/loft-sh/vcluster/pkg/server"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/osutil"
	"k8s.io/klog/v2"
)

func StartProxy(ctx *synccontext.ControllerContext) error {
	// start the proxy
	proxyServer, err := server.NewServer(ctx)
	if err != nil {
		return err
	}

	// start the proxy server in secure mode
	go func() {
		err = proxyServer.ServeOnListenerTLS(ctx)
		if err != nil {
			klog.Errorf("Error serving: %v", err)
			osutil.Exit(1)
		}
	}()

	return nil
}
