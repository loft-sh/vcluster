package integrations

import (
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/integrations/metricsserver"
)

type Integration func(ctx *config.ControllerContext) error

var Integrations = []Integration{
	metricsserver.Register,
}

func StartIntegrations(ctx *config.ControllerContext) error {
	for _, integration := range Integrations {
		err := integration(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}
