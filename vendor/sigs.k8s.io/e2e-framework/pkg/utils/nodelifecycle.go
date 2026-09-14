/*
Copyright 2024 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"context"
	"fmt"

	"sigs.k8s.io/e2e-framework/pkg/types"
)

// PerformNodeLifecycleOperation performs a node operation on a cluster. These operations can range from Add/Remove/Start/Stop.
// This helper is re-used in both node lifecycle handler used as types.StepFunc or env.Func
func PerformNodeLifecycleOperation(ctx context.Context, action types.NodeOperation, node *types.Node, args ...string) error {
	clusterVal := ctx.Value(types.ClusterNameContextKey(node.Cluster))
	if clusterVal == nil {
		return fmt.Errorf("%s node to cluster: context cluster is nil", action)
	}

	clusterProvider, ok := clusterVal.(types.E2EClusterProviderWithLifeCycle)
	if !ok {
		return fmt.Errorf("cluster provider %s doesn't support node lifecycle operations", node.Cluster)
	}

	switch action {
	case types.AddNode:
		return clusterProvider.AddNode(ctx, node, args...)
	case types.RemoveNode:
		return clusterProvider.RemoveNode(ctx, node, args...)
	case types.StartNode:
		return clusterProvider.StartNode(ctx, node, args...)
	case types.StopNode:
		return clusterProvider.StopNode(ctx, node, args...)
	default:
		return fmt.Errorf("unknown node operation: %s", action)
	}
}
