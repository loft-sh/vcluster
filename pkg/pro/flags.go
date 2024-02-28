package pro

import (
	"github.com/loft-sh/vcluster/pkg/options"
	"github.com/spf13/pflag"
)

func AddProFlags(flags *pflag.FlagSet, options *options.VirtualClusterOptions) {
	flags.StringVar(&options.ProOptions.ProLicenseSecret, "pro-license-secret", "", "If set, vCluster.Pro will try to find this secret to retrieve the vCluster.Pro license.")

	flags.StringVar(&options.ProOptions.RemoteKubeConfig, "remote-kube-config", "", "If set, will use the remote kube-config instead of the local in-cluster one. Expects a kube config to a headless vcluster installation")
	flags.StringVar(&options.ProOptions.RemoteNamespace, "remote-namespace", "", "If set, will use this as the remote namespace")
	flags.StringVar(&options.ProOptions.RemoteServiceName, "remote-service-name", "", "If set, will use this as the remote service name")

	flags.BoolVar(&options.ProOptions.IntegratedCoredns, "integrated-coredns", false, "If enabled vcluster will spin an in memory coreDNS inside the syncer container")
	flags.BoolVar(&options.ProOptions.UseCoreDNSPlugin, "use-coredns-plugin", false, "If enabled, the vcluster plugin for coredns will be used")
	flags.BoolVar(&options.ProOptions.NoopSyncer, "noop-syncer", false, "If enabled will setup a noop Syncer that filters and proxies requests to a specified remote cluster")
	flags.BoolVar(&options.ProOptions.SyncKubernetesService, "sync-k8s-service", false, "If enabled will sync the kubernetes service endpoints in the remote cluster with the load balancer ip of this cluster")

	flags.BoolVar(&options.ProOptions.EtcdEmbedded, "etcd-embedded", false, "If true, will start an embedded etcd within vCluster")
	flags.StringVar(&options.ProOptions.MigrateFrom, "migrate-from", "", "The url (including protocol) of the original database")
	flags.IntVar(&options.ProOptions.EtcdReplicas, "etcd-replicas", 0, "The amount of replicas the etcd has")

	flags.StringArrayVar(&options.ProOptions.EnforceValidatingHooks, "enforce-validating-hook", nil, "A validating hook configuration in yaml format encoded with base64. Can be used multiple times")
	flags.StringArrayVar(&options.ProOptions.EnforceMutatingHooks, "enforce-mutating-hook", nil, "A mutating hook configuration in yaml format encoded with base64. Can be used multiple times")
}
