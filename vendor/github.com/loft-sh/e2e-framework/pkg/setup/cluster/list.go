package cluster

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/loft-sh/e2e-framework/pkg/provider"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/support"
)

type listKey int

const (
	listContextKey listKey = iota
)

var (
	clusterList []string
)

func From(ctx context.Context, name string) support.E2EClusterProvider {
	if k, ok := envfuncs.GetClusterFromContext(ctx, name); ok {
		return k
	}
	return nil
}

func With(ctx context.Context, clusterName string, provider support.E2EClusterProvider) context.Context {
	return context.WithValue(ctx, support.ClusterNameContextKey(clusterName), provider)
}

func List(_ context.Context) []string {
	// Clone to avoid accidental modification
	return slices.Clone(clusterList)
}

func Add(ctx context.Context, name string) context.Context {
	if name == "" {
		panic("cluster name is required")
	}

	clusterList = append(clusterList, name)
	slices.Sort(clusterList)
	clusterList = slices.Compact(clusterList)

	// Clone to avoid accidental modification
	return context.WithValue(ctx, listContextKey, slices.Clone(clusterList))
}

func Remove(ctx context.Context, name string) context.Context {
	if name == "" {
		panic("cluster name is required")
	}

	clusterList = slices.DeleteFunc(clusterList, func(item string) bool {
		return item == name
	})

	// Clone to avoid accidental modification
	return context.WithValue(ctx, listContextKey, slices.Clone(clusterList))
}

func ExportAll(ctx context.Context) ([]byte, error) {
	var clusters []support.E2EClusterProvider
	for _, clusterName := range List(ctx) {
		cluster := From(ctx, clusterName)
		clusters = append(clusters, cluster)
	}
	return json.Marshal(clusters)
}

func ImportAll(ctx context.Context, data []byte) (context.Context, error) {
	clusters, err := provider.LoadFromBytes(data)
	if err != nil {
		return ctx, err
	}

	for name, c := range clusters {
		ctx = Add(ctx, name)
		ctx = With(ctx, name, c)
	}
	return ctx, nil
}
