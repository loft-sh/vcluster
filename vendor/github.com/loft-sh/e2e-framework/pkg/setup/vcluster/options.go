package vcluster

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/suite"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/support"
)

type vCluster struct {
	name         string
	version      string
	hostCluster  string
	vClusterYAML string
	envCfg       *envconf.Config
	opts         []support.ClusterOpts
	dependencies []suite.Dependency
}

type Options func(c *vCluster)

func WithName(name string) Options {
	return func(c *vCluster) {
		c.name = name
	}
}

func WithVersion(version string) Options {
	return func(c *vCluster) {
		c.version = version
	}
}

func WithHostCluster(hostCluster string) Options {
	return func(c *vCluster) {
		c.hostCluster = hostCluster
	}
}

func WithVClusterYAML(vClusterYAML string) Options {
	return func(c *vCluster) {
		c.vClusterYAML = vClusterYAML
	}
}

func WithOptions(opts ...support.ClusterOpts) Options {
	return func(c *vCluster) {
		c.opts = opts
	}
}

func WithDependencies(dependencies ...suite.Dependency) Options {
	return func(c *vCluster) {
		c.dependencies = dependencies
	}
}
