/*
Copyright 2023 The Kubernetes Authors.

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

package envfuncs

import (
	"context"

	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/support/kind"
)

// Deprecated: This handler has been deprecated in favor of GetClusterFromContext
func GetKindClusterFromContext(ctx context.Context, clusterName string) (*kind.Cluster, bool) {
	provider, ok := GetClusterFromContext(ctx, clusterName)
	if ok {
		return provider.(*kind.Cluster), ok // nolint: errcheck
	}
	return nil, ok
}

// Deprecated: This handler has been deprecated in favor of CreateCluster which can now accept
// support.ClusterProvider type as input in order to setup the cluster using right providers
func CreateKindCluster(clusterName string) env.Func {
	return CreateCluster(kind.NewProvider(), clusterName)
}

// Deprecated: This handler has been deprecated in favor of CreateClusterWithConfig which can now accept
// support.ClusterProvider type as input in order to setup the cluster using right providers
func CreateKindClusterWithConfig(clusterName, image, configFilePath string) env.Func {
	return CreateClusterWithConfig(kind.NewProvider(), clusterName, configFilePath, kind.WithImage(image))
}

// Deprecated: This handler has been deprecated in favor of DestroyCluster
func DestroyKindCluster(name string) env.Func {
	return DestroyCluster(name)
}

// Deprecated: This handler has been deprecated in favor of ExportClusterLogs
func ExportKindClusterLogs(name, dest string) env.Func {
	return ExportClusterLogs(name, dest)
}
