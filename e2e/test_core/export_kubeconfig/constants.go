package export_kubeconfig

// Exported constants describing the ExportKubeConfig vCluster fixture.
// The vCluster lifecycle lives in suite_export_kubeconfig_test.go; these
// constants live here (the single test_* package that uses them) per the
// e2e-conventions.md rule "test-specific constants belong in the test
// package". The suite file imports them from here.

const (
	VClusterName = "export-kubeconfig-vcluster"
	TargetNS     = "export-kubeconfig-target"

	// Same-namespace additional secret config (must match YAML).
	SameNSSecretName = "export-kubeconfig-same-ns"
	SameNSServer     = "https://export-kubeconfig-vcluster.vcluster-export-kubeconfig-vcluster.svc:443"
	SameNSContext    = "same-ns-context"

	// Cross-namespace additional secret config (must match YAML).
	CrossNSSecretName = "export-kubeconfig-cross-ns"
	CrossNSServer     = "https://export-kubeconfig-vcluster.export-kubeconfig-target.svc:443"
	CrossNSContext    = "cross-ns-context"
)
