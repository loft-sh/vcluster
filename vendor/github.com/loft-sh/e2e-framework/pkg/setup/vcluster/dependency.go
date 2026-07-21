package vcluster

import (
	"context"

	vcluster2 "github.com/loft-sh/e2e-framework/pkg/provider/vcluster"
	"github.com/loft-sh/e2e-framework/pkg/setup"
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/e2e-framework/pkg/setup/suite"
	. "github.com/onsi/gomega"
)

func Define(options ...Options) suite.Dependency {
	vc := &vCluster{}
	for _, option := range options {
		option(vc)
	}

	// Add host cluster as a dependency
	if vc.hostCluster != "" {
		hostCluster, ok := suite.Lookup(cluster.Label(vc.hostCluster))
		if ok {
			vc.dependencies = append(vc.dependencies, hostCluster)
		}
	}

	// Use same setup for create + import
	setupFn := func(vc *vCluster) setup.Func {
		return func(ctx context.Context) (context.Context, error) {
			hostCluster := cluster.CurrentClusterFrom(ctx)
			if vc.hostCluster != "" {
				hostCluster = cluster.From(ctx, vc.hostCluster)
			}

			vcOpts := vc.opts
			if hostCluster != nil {
				vcOpts = append(vcOpts,
					vcluster2.WithHostKubeConfig(hostCluster.GetKubeconfig()),
				)
			}

			clusterOpts := []cluster.Options{
				cluster.WithName(vc.name),
				cluster.WithConfigFile(vc.vClusterYAML),
				cluster.WithEnvConfig(vc.envCfg),
				cluster.WithProvider(
					vcluster2.NewProvider().WithVersion(vc.version),
				),
				cluster.WithOptions(vcOpts...),
				cluster.WithDependencies(vc.dependencies...),
			}

			return cluster.Create(clusterOpts...)(ctx)
		}
	}(vc)

	// Define before fn
	before := func(vc *vCluster) suite.SetupContextCallback {
		return func(ctx context.Context) context.Context {
			var err error

			ctx, err = cluster.UseCluster(vc.name)(ctx)
			Expect(err).NotTo(HaveOccurred())

			return ctx
		}
	}(vc)

	return suite.Define(
		suite.WithLabel(cluster.Label(vc.name)),
		suite.WithDependencies(vc.dependencies...),
		suite.WithSetup(setupFn),
		suite.WithImport(setupFn),
		suite.WithTeardown(cluster.Destroy(vc.name)),
		suite.WithBeforeEach(before),
		suite.WithBeforeAll(before),
	)
}
