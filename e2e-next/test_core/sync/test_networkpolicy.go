package test_core

import (
	"context"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	networkingv1 "k8s.io/api/networking/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var _ = Describe("NetworkPolicy sync from vCluster to host",
	labels.Core,
	labels.Sync,
	labels.NetworkPolicies,
	cluster.Use(clusters.K8sDefaultEndpointVCluster),
	cluster.Use(clusters.HostCluster),
	func() {
		var (
			hostClient        kubernetes.Interface
			vClusterClient    kubernetes.Interface
			vClusterName      = clusters.K8sDefaultEndpointVClusterName
			vClusterNamespace = "vcluster-" + vClusterName
		)

		BeforeEach(func(ctx context.Context) {
			hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
			Expect(hostClient).NotTo(BeNil())
			vClusterClient = cluster.CurrentKubeClientFrom(ctx)
			Expect(vClusterClient).NotTo(BeNil())
		})

		It("should sync a NetworkPolicy from vCluster to the host cluster", func(ctx context.Context) {
			const vclusterNS = "default"
			suffix := random.String(6)
			policyName := "np-sync-test-" + suffix
			hostPolicyName := translate.SingleNamespaceHostName(policyName, vclusterNS, vClusterName)

			policy := &networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      policyName,
					Namespace: vclusterNS,
				},
				Spec: networkingv1.NetworkPolicySpec{
					PodSelector: metav1.LabelSelector{},
					PolicyTypes: []networkingv1.PolicyType{
						networkingv1.PolicyTypeIngress,
						networkingv1.PolicyTypeEgress,
					},
					Ingress: []networkingv1.NetworkPolicyIngressRule{{}},
					Egress:  []networkingv1.NetworkPolicyEgressRule{{}},
				},
			}
			_, err := vClusterClient.NetworkingV1().NetworkPolicies(vclusterNS).Create(ctx, policy, metav1.CreateOptions{})
			Expect(err).To(Succeed())

			DeferCleanup(func(ctx context.Context) {
				err := vClusterClient.NetworkingV1().NetworkPolicies(vclusterNS).Delete(ctx, policyName, metav1.DeleteOptions{})
				if !kerrors.IsNotFound(err) {
					Expect(err).To(Succeed())
				}
				err = hostClient.NetworkingV1().NetworkPolicies(vClusterNamespace).Delete(ctx, hostPolicyName, metav1.DeleteOptions{})
				if !kerrors.IsNotFound(err) {
					Expect(err).To(Succeed())
				}
			})

			By("Waiting for the NetworkPolicy to appear in the host cluster", func() {
				Eventually(func(g Gomega) {
					hostPolicy, err := hostClient.NetworkingV1().NetworkPolicies(vClusterNamespace).Get(ctx, hostPolicyName, metav1.GetOptions{})
					g.Expect(err).NotTo(HaveOccurred(),
						"NetworkPolicy %s/%s not yet synced to host", vClusterNamespace, hostPolicyName)
					g.Expect(hostPolicy.Spec.PolicyTypes).To(ContainElements(
						networkingv1.PolicyTypeIngress,
						networkingv1.PolicyTypeEgress,
					), "synced policy should preserve policy types")
				}).
					WithPolling(constants.PollingInterval).
					WithTimeout(constants.PollingTimeout).
					Should(Succeed())
			})
		})
	},
)
