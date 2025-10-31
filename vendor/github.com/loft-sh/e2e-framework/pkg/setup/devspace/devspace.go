package devspace

import (
	"context"
	"maps"

	"github.com/loft-sh/e2e-framework/pkg/setup"
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	corev1 "k8s.io/api/core/v1"
	clientpkg "sigs.k8s.io/controller-runtime/pkg/client"
)

type key int

const devSpaceKey key = iota

func With(ctx context.Context, devspace bool) context.Context {
	return context.WithValue(ctx, devSpaceKey, devspace)
}

func From(ctx context.Context) bool {
	if devSpace, ok := ctx.Value(devSpaceKey).(bool); ok {
		return devSpace
	}
	return false
}

func HasReplacedPod(namespace string, labels map[string]string) setup.Func {
	return func(ctx context.Context) (context.Context, error) {
		listLabels := maps.Clone(labels)
		listLabels["devspace.sh/replaced"] = "true"

		client := cluster.CurrentClusterClientFrom(ctx)
		podList := &corev1.PodList{}
		client.List(ctx, podList, clientpkg.InNamespace(namespace), clientpkg.MatchingLabels(listLabels))
		return With(ctx, len(podList.Items) > 0), nil
	}
}
