/*
Copyright 2021 The Kubernetes Authors.

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
	"fmt"

	"sigs.k8s.io/e2e-framework/pkg/utils"

	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/support"
)

var LoadDockerImageToCluster = LoadImageToCluster

// GetClusterFromContext helps extract the E2EClusterProvider object from the context.
// This can be used to setup and run tests of multi cluster e2e Prioviders.
func GetClusterFromContext(ctx context.Context, clusterName string) (support.E2EClusterProvider, bool) {
	c := ctx.Value(support.ClusterNameContextKey(clusterName))
	if c == nil {
		return nil, false
	}
	cluster, ok := c.(support.E2EClusterProvider)
	return cluster, ok
}

// CreateCluster returns an env.Func that is used to
// create an E2E provider cluster that is then injected in the context
// using the name as a key.
//
// NOTE: the returned function will update its env config with the
// kubeconfig file for the config client.
func CreateCluster(p support.E2EClusterProvider, clusterName string) env.Func {
	return CreateClusterWithOpts(p, clusterName)
}

// CreateClusterWithOpts returns an env.Func that is used to
// create an E2E provider cluster that is then injected in the context
// using the name as a key. This can be provided with additional opts to extend the create
// workflow of the cluster.
//
// NOTE: the returned function will update its env config with the
// kubeconfig file for the config client.
func CreateClusterWithOpts(p support.E2EClusterProvider, clusterName string, opts ...support.ClusterOpts) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		k := p.SetDefaults().WithName(clusterName).WithOpts(opts...)
		kubecfg, err := k.Create(ctx)
		if err != nil {
			return ctx, err
		}

		// update envconfig  with kubeconfig
		cfg.WithKubeconfigFile(kubecfg)

		// stall, wait for pods initializations
		if err := k.WaitForControlPlane(ctx, cfg.Client()); err != nil {
			return ctx, err
		}

		// store entire cluster value in ctx for future access using the cluster name
		return context.WithValue(ctx, support.ClusterNameContextKey(clusterName), k), nil
	}
}

// CreateClusterWithConfig returns an env.Func that is used to
// create a e2e provider cluster that is then injected in the context
// using the name as a key.
//
// NOTE: the returned function will update its env config with the
// kubeconfig file for the config client.
func CreateClusterWithConfig(p support.E2EClusterProvider, clusterName, configFilePath string, opts ...support.ClusterOpts) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		k := p.SetDefaults().WithName(clusterName).WithOpts(opts...)
		kubecfg, err := k.CreateWithConfig(ctx, configFilePath)
		if err != nil {
			return ctx, err
		}

		// update envconfig  with kubeconfig
		cfg.WithKubeconfigFile(kubecfg)

		// stall, wait for pods initializations
		if err := k.WaitForControlPlane(ctx, cfg.Client()); err != nil {
			return ctx, err
		}

		// store entire cluster value in ctx for future access using the cluster name
		return context.WithValue(ctx, support.ClusterNameContextKey(clusterName), k), nil
	}
}

// DestroyCluster returns an EnvFunc that
// retrieves a previously saved e2e provider Cluster in the context (using the name), then deletes it.
//
// NOTE: this should be used in a Environment.Finish step.
func DestroyCluster(name string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		clusterVal := ctx.Value(support.ClusterNameContextKey(name))
		if clusterVal == nil {
			return ctx, fmt.Errorf("destroy e2e provider cluster func: context cluster is nil")
		}

		cluster, ok := clusterVal.(support.E2EClusterProvider)
		if !ok {
			return ctx, fmt.Errorf("destroy e2e provider cluster func: unexpected type for cluster value")
		}

		if err := cluster.Destroy(ctx); err != nil {
			return ctx, fmt.Errorf("destroy e2e provider cluster: %w", err)
		}

		return ctx, nil
	}
}

// LoadImageToCluster returns an EnvFunc that
// retrieves a previously saved e2e provider Cluster in the context (using the name), and then loads a container image
// from the host into the cluster.
func LoadImageToCluster(name, image string, args ...string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
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

// LoadImageArchiveToCluster returns an EnvFunc that
// retrieves a previously saved e2e provider Cluster in the context (using the name), and then loads a container image TAR archive
// from the host into the cluster.
func LoadImageArchiveToCluster(name, imageArchive string, args ...string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		clusterVal := ctx.Value(support.ClusterNameContextKey(name))
		if clusterVal == nil {
			return ctx, fmt.Errorf("load image archive func: context cluster is nil")
		}

		cluster, ok := clusterVal.(support.E2EClusterProviderWithImageLoader)
		if !ok {
			return ctx, fmt.Errorf("load image archive func: cluster provider does not support LoadImageArchive helper")
		}

		if err := cluster.LoadImageArchive(ctx, imageArchive, args...); err != nil {
			return ctx, fmt.Errorf("load image archive: %w", err)
		}

		return ctx, nil
	}
}

// ExportClusterLogs returns an EnvFunc that
// retrieves a previously saved e2e provider Cluster in the context (using the name), and then export cluster logs
// in the provided destination.
func ExportClusterLogs(name, dest string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		clusterVal := ctx.Value(support.ClusterNameContextKey(name))
		if clusterVal == nil {
			return ctx, fmt.Errorf("export e2e provider cluster logs: context cluster is nil")
		}

		cluster, ok := clusterVal.(support.E2EClusterProvider)
		if !ok {
			return ctx, fmt.Errorf("export e2e provider cluster logs: unexpected type for cluster value")
		}

		if err := cluster.ExportLogs(ctx, dest); err != nil {
			return ctx, fmt.Errorf("load image archive: %w", err)
		}

		return ctx, nil
	}
}

// PerformNodeOperation returns an EnvFunc that can be used to perform some node lifecycle operations.
// This can be used to add/remove/start/stop nodes in the cluster.
func PerformNodeOperation(clusterName string, action support.NodeOperation, node *support.Node, args ...string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		err := utils.PerformNodeLifecycleOperation(ctx, action, node, args...)
		return ctx, err
	}
}
