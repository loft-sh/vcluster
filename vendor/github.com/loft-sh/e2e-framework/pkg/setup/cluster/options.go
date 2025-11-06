package cluster

import (
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/support"
)

type cluster struct {
	name       string
	provider   support.E2EClusterProvider
	configFile string
	envCfg     *envconf.Config
	opts       []support.ClusterOpts
}

type Options func(c *cluster)

func WithProvider(p support.E2EClusterProvider) Options {
	return func(c *cluster) {
		c.provider = p
	}
}

func WithName(name string) Options {
	return func(c *cluster) {
		c.name = name
	}
}

func WithConfigFile(path string) Options {
	return func(c *cluster) {
		c.configFile = path
	}
}

func WithEnvConfig(envCfg *envconf.Config) Options {
	return func(c *cluster) {
		c.envCfg = envCfg
	}
}

func WithOptions(opts ...support.ClusterOpts) Options {
	return func(c *cluster) {
		c.opts = opts
	}
}
