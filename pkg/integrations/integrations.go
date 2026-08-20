package integrations

import (
	"github.com/loft-sh/vcluster/pkg/integrations/metricsserver"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
)

type Integration func(ctx *synccontext.ControllerContext) error

var Integrations = []Integration{
	metricsserver.Register,
}

func StartIntegrations(ctx *synccontext.ControllerContext) error {
	for _, integration := range Integrations {
		err := integration(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}
