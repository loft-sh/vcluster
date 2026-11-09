package resources

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func ensureHostGatewayAPIKind(ctx *synccontext.RegisterContext, gvk schema.GroupVersionKind, configPath string) error {
	if ctx.HostManager == nil || ctx.HostManager.GetConfig() == nil {
		return fmt.Errorf("cannot check host cluster for Gateway API resource %s: host manager is not configured", gvk.String())
	}

	exists, err := util.KindExists(ctx.HostManager.GetConfig(), gvk)
	if err != nil {
		return fmt.Errorf("check host cluster for Gateway API resource %s: %w", gvk.String(), err)
	}
	if !exists {
		return fmt.Errorf("host cluster does not advertise Gateway API resource %s; install Gateway API CRDs that serve this version on the host cluster or disable %s", gvk.String(), configPath)
	}

	return nil
}
