package cluster

import (
	"context"

	"github.com/loft-sh/e2e-framework/pkg/setup"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
)

func Create(options ...Options) setup.Func {
	return func(ctx context.Context) (context.Context, error) {
		c := &cluster{}
		for _, o := range options {
			o(c)
		}

		// Check if cluster already exists in context
		if existingCluster := From(ctx, c.name); existingCluster != nil {
			return ctx, nil
		}

		if c.envCfg == nil {
			c.envCfg = envconf.New()
		}

		if c.configFile == "" {
			// Likely an existing cluster
			var err error
			ctx, err = envfuncs.CreateClusterWithOpts(c.provider, c.name, c.opts...)(ctx, c.envCfg)
			if err != nil {
				return ctx, err
			}
		} else {
			var err error
			ctx, err = envfuncs.CreateClusterWithConfig(c.provider, c.name, c.configFile, c.opts...)(ctx, c.envCfg)
			if err != nil {
				return ctx, err
			}
		}

		return Add(ctx, c.name), nil
	}
}
