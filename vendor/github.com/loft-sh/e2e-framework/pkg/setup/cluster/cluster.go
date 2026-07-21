package cluster

import (
	"context"

	"github.com/loft-sh/e2e-framework/pkg/setup"
	"github.com/loft-sh/e2e-framework/pkg/setup/suite"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func Define(options ...Options) suite.Dependency {
	c := &cluster{}
	for _, o := range options {
		o(c)
	}

	before := func(c *cluster) suite.SetupContextCallback {
		return func(ctx context.Context) context.Context {
			var err error

			ctx, err = UseCluster(c.name)(ctx)
			Expect(err).NotTo(HaveOccurred())

			return ctx
		}
	}(c)

	return suite.Define(
		suite.WithLabel(Label(c.name)),
		suite.WithDependencies(c.dependencies...),
		suite.WithSetup(Create(options...)),
		suite.WithImport(Import(options...)),
		suite.WithTeardown(Destroy(c.name)),
		suite.WithBeforeEach(before),
		suite.WithBeforeAll(before),
	)
}

func Use(dependency suite.Dependency) ginkgo.Labels {
	return suite.Use(dependency)
}

func Import(options ...Options) setup.Func {
	return Create(options...)
}

func Label(clusterName string) string {
	return "cluster: " + clusterName
}
