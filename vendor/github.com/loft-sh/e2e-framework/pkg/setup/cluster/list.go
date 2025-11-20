package cluster

import (
	"context"
	"slices"

	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/support"
)

type listKey int

const (
	listContextKey listKey = iota
)

func From(ctx context.Context, name string) support.E2EClusterProvider {
	if k, ok := envfuncs.GetClusterFromContext(ctx, name); ok {
		return k
	}
	return nil
}

func List(ctx context.Context) []string {
	if l := ctx.Value(listContextKey); l != nil {
		return l.([]string)
	}

	return nil
}

func Add(ctx context.Context, name string) context.Context {
	list := List(ctx)
	list = append(list, name)
	slices.Sort(list)
	return context.WithValue(ctx, listContextKey, slices.Compact(list))
}

func Remove(ctx context.Context, name string) context.Context {
	var newList []string
	list := List(ctx)
	for _, item := range list {
		if item == name {
			continue
		}

		newList = append(newList, item)
	}
	return context.WithValue(ctx, listContextKey, newList)
}
