package cluster

import (
	"context"
	"fmt"

	"github.com/loft-sh/e2e-framework/pkg/setup"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
)

func Destroy(clusterName string) setup.Func {
	return func(ctx context.Context) (context.Context, error) {
		if clusterName == "" {
			return ctx, fmt.Errorf("cluster name is required")
		}

		var err error
		if ctx, err = envfuncs.DestroyCluster(clusterName)(ctx, nil); err != nil {
			return ctx, err
		}

		return Remove(ctx, clusterName), nil
	}
}

func DestroyAll() setup.Func {
	return func(ctx context.Context) (context.Context, error) {
		clusters := List(ctx)
		for _, c := range clusters {
			var err error
			ctx, err = Destroy(c)(ctx)
			if err != nil {
				return ctx, err
			}
		}
		return ctx, nil
	}
}
