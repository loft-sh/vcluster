package vcluster

import (
	"context"

	"github.com/loft-sh/e2e-framework/pkg/e2e"
	providervcluster "github.com/loft-sh/e2e-framework/pkg/provider/vcluster"
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/e2e-framework/pkg/setup/suite"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/e2e-framework/support"
)

// newProvider is swapped in tests via SetProviderForTesting.
var newProvider = func() support.E2EClusterProvider {
	return providervcluster.NewProvider()
}

// Spec describes a single per-Describe vCluster to provision from a
// BeforeAll. On spec failure the vCluster is kept alive and diagnostics
// are attached to the spec report; on pass it is destroyed.
type Spec struct {
	// Name of the vCluster. Must be unique per Ginkgo process.
	Name string

	// ConfigFile is the pre-rendered vcluster.yaml path. Caller owns
	// rendering and cleanup of the file.
	ConfigFile string

	// ClusterOpts are appended to the provider opts. The helper also
	// adds WithHostKubeConfig for the currently-active host cluster.
	ClusterOpts []support.ClusterOpts

	// HostClusterDep is registered via cluster.WithDependencies. Optional.
	HostClusterDep suite.Dependency

	// HostClusterName is looked up in ctx to drive the Go-client
	// diagnostics dump on failure. Empty means no diagnostics.
	HostClusterName string

	// PreSetup runs once before the vCluster is created (e.g. host-side
	// CRDs or PVCs). An error aborts the BeforeAll.
	PreSetup func(ctx context.Context) error
}

// Create provisions the vCluster, registers failure-aware teardown, and
// makes it the current cluster in ctx. Call from a BeforeAll in a
// ginkgo.Ordered Describe.
func Create(ctx context.Context, spec Spec) context.Context {
	GinkgoHelper()

	hostCluster := cluster.CurrentClusterFrom(ctx)
	Expect(hostCluster).NotTo(BeNil(),
		"vcluster %q: host cluster must be current in ctx - add cluster.Use(HostCluster) to the Describe", spec.Name)
	if hostCluster == nil {
		return ctx // only reachable under InterceptGomegaFailures; real runs abort above
	}

	// cluster.Create silently returns when a cluster with this name is
	// already in ctx; our teardown would then destroy someone else's
	// cluster. Fail fast instead.
	existing := cluster.From(ctx, spec.Name)
	Expect(existing).To(BeNil(),
		"vcluster %q: already registered in ctx - names must be unique per process", spec.Name)
	if existing != nil {
		return ctx
	}

	if spec.PreSetup != nil {
		Expect(spec.PreSetup(ctx)).To(Succeed(), "vcluster %q preSetup", spec.Name)
	}

	vcOpts := append([]support.ClusterOpts{}, spec.ClusterOpts...)
	vcOpts = append(vcOpts, providervcluster.WithHostKubeConfig(hostCluster.GetKubeconfig()))

	createOpts := []cluster.Options{
		cluster.WithName(spec.Name),
		cluster.WithConfigFile(spec.ConfigFile),
		cluster.WithProvider(newProvider()),
		cluster.WithOptions(vcOpts...),
	}
	if spec.HostClusterDep != nil {
		createOpts = append(createOpts, cluster.WithDependencies(spec.HostClusterDep))
	}

	var err error
	//nolint:defercleanupcluster // teardown is conditional on spec failure; see DeferCleanupCtx below.
	ctx, err = cluster.Create(createOpts...)(ctx)
	Expect(err).To(Succeed(), "vcluster %q create", spec.Name)

	name, hostName, configFile := spec.Name, spec.HostClusterName, spec.ConfigFile
	e2e.DeferCleanupCtx(ctx, func(ctx context.Context) (context.Context, error) {
		if CurrentSpecReport().Failed() {
			DumpDiagnostics(ctx, hostName, name, configFile)
			return ctx, nil
		}
		return cluster.Destroy(name)(ctx)
	})

	ctx, err = cluster.UseCluster(spec.Name)(ctx)
	Expect(err).To(Succeed(), "vcluster %q use", spec.Name)
	return ctx
}
