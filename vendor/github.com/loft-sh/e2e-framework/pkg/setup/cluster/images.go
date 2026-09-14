package cluster

import (
	"context"
	"fmt"
	"slices"

	"github.com/loft-sh/e2e-framework/pkg/setup"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/support"
)

func LoadImage(name, image string, args ...string) setup.Func {
	return func(ctx context.Context) (context.Context, error) {
		if hasImage(ctx, name, image) {
			return ctx, nil
		}

		clusterVal := ctx.Value(support.ClusterNameContextKey(name))
		if clusterVal == nil {
			return ctx, fmt.Errorf("load image func: context cluster is nil")
		}

		cluster, ok := clusterVal.(support.E2EClusterProviderWithImageLoader)
		if !ok {
			return ctx, fmt.Errorf("load image archive func: cluster provider does not support LoadImage helper")
		}

		if err := cluster.LoadImage(ctx, image, args...); err != nil {
			return ctx, fmt.Errorf("load image: %w", err)
		}

		return ctx, nil
	}
}

func hasImage(ctx context.Context, clusterName, image string) bool {
	client := ControllerRuntimeClientFrom(ctx, clusterName)
	if client == nil {
		return false
	}

	nodeList := &corev1.NodeList{}
	if err := client.List(ctx, nodeList); err != nil {
		return false
	}

	found := map[string]bool{}
	for _, node := range nodeList.Items {
		found[node.Name] = false
	imageLoop:
		for _, images := range node.Status.Images {
			if slices.Contains(images.Names, image) {
				found[node.Name] = true
				break imageLoop
			}
		}
	}

	for _, v := range found {
		if !v {
			return false
		}
	}

	return true
}
