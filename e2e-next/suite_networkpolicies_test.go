// Suite: networkpolicies-vcluster
// vCluster: CommonVCluster config (reuses default YAML)
// Run:      just run-e2e 'networkpolicies'
// Prereq:   Kind cluster with Calico CNI (disableDefaultCNI + calico.yaml)
//
// NetworkPolicy enforcement tests require a CNI that actually enforces policies.
// Standard Kind uses kindnet which does NOT enforce NetworkPolicies.
// This suite runs as a separate CI job with Calico installed.
//
// Local setup:
//
//	cat <<EOF | kind create cluster --name kind-cluster --config -
//	kind: Cluster
//	apiVersion: kind.x-k8s.io/v1alpha4
//	networking:
//	  disableDefaultCNI: true
//	  podSubnet: 192.168.0.0/16
//	EOF
//	kubectl apply -f https://raw.githubusercontent.com/projectcalico/calico/v3.29.3/manifests/calico.yaml
//	kubectl -n kube-system wait --for=condition=Ready pod -l k8s-app=calico-node --timeout=120s
//	just run-e2e 'networkpolicies'
package e2e_next

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	test_core "github.com/loft-sh/vcluster/e2e-next/test_core/sync"
	. "github.com/onsi/ginkgo/v2"
)

func init() {
	suiteNetworkPoliciesVCluster()
}

func suiteNetworkPoliciesVCluster() {
	Describe("networkpolicies-vcluster",
		cluster.Use(clusters.CommonVCluster),
		cluster.Use(clusters.HostCluster),
		func() {
			test_core.NetworkPolicyEnforcementSpec()
		},
	)
}
