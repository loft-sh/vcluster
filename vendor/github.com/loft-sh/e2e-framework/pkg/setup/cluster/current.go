package cluster

import (
	"context"
	"fmt"

	"github.com/loft-sh/e2e-framework/pkg/setup"
	"k8s.io/client-go/kubernetes"
	clientpkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/types"
)

type currentKey int

const (
	currentClusterContextKey currentKey = iota
	currentClusterNameContextKey
)

func WithCurrentClusterName(ctx context.Context, clusterName string) context.Context {
	return context.WithValue(ctx, currentClusterNameContextKey, clusterName)
}

func CurrentClusterNameFrom(ctx context.Context) string {
	if value := ctx.Value(currentClusterNameContextKey); value != nil {
		return value.(string)
	}
	return ""
}

func WithCurrentCluster(ctx context.Context, cluster types.E2EClusterProvider) context.Context {
	return context.WithValue(ctx, currentClusterContextKey, cluster)
}

func CurrentClusterFrom(ctx context.Context) types.E2EClusterProvider {
	if value := ctx.Value(currentClusterContextKey); value != nil {
		return value.(types.E2EClusterProvider)
	}
	return nil
}

func CurrentClusterClientFrom(ctx context.Context) clientpkg.Client {
	currentCluster := CurrentClusterNameFrom(ctx)
	return ControllerRuntimeClientFrom(ctx, currentCluster)
}

func CurrentKubeClientFrom(ctx context.Context) kubernetes.Interface {
	currentCluster := CurrentClusterNameFrom(ctx)
	return KubeClientFrom(ctx, currentCluster)
}

func SetupClients(clusterName string) setup.Func {
	return func(ctx context.Context) (context.Context, error) {
		if ControllerRuntimeClientFrom(ctx, clusterName) == nil {
			var err error
			if ctx, err = SetupControllerRuntimeClient(WithCluster(clusterName))(ctx); err != nil {
				return ctx, err
			}
		}

		if KubeClientFrom(ctx, clusterName) == nil {
			var err error
			if ctx, err = SetupKubeClient(clusterName)(ctx); err != nil {
				return ctx, err
			}
		}
		return ctx, nil
	}
}

func UseCluster(clusterName string) setup.Func {
	return func(ctx context.Context) (context.Context, error) {
		clusterVal := From(ctx, clusterName)
		if clusterVal == nil {
			return ctx, fmt.Errorf("cluster not found in context")
		}

		ctx = WithCurrentClusterName(ctx, clusterName)
		ctx = WithCurrentCluster(ctx, clusterVal)
		return SetupClients(clusterName)(ctx)
	}
}
