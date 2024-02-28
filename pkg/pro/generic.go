package pro

import (
	"github.com/loft-sh/vcluster/pkg/options"
	"github.com/loft-sh/vcluster/pkg/util/blockingcacheclient"
	"github.com/loft-sh/vcluster/pkg/util/pluginhookclient"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var InitProControllerContext = func(_ *options.ControllerContext) error {
	return nil
}

var NewPhysicalClient = func(_ *options.VirtualClusterOptions) client.NewClientFunc {
	return pluginhookclient.NewPhysicalPluginClientFactory(blockingcacheclient.NewCacheClient)
}

var NewVirtualClient = func(_ *options.VirtualClusterOptions) client.NewClientFunc {
	return pluginhookclient.NewVirtualPluginClientFactory(blockingcacheclient.NewCacheClient)
}
