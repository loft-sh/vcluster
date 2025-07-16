package tohost

import (
	"time"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("Test sync NetworkPolicy from vCluster to host", ginkgo.Ordered, func() {
	var (
		f                 *framework.Framework
		vclusterNamespace = "default"
		hostNamespace     = "vcluster"
		policyName        = "allow-all"
	)

	ginkgo.BeforeAll(func() {
		f = framework.DefaultFramework

		ginkgo.By("Creating NetworkPolicy in vCluster")
		policy := &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      policyName,
				Namespace: vclusterNamespace,
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
		_, err := f.VClusterClient.NetworkingV1().NetworkPolicies(vclusterNamespace).Create(f.Context, policy, metav1.CreateOptions{})
		framework.ExpectNoError(err)
	})

	ginkgo.AfterAll(func() {
		_ = f.VClusterClient.NetworkingV1().NetworkPolicies(vclusterNamespace).Delete(f.Context, policyName, metav1.DeleteOptions{})
		_ = f.HostClient.NetworkingV1().NetworkPolicies(hostNamespace).Delete(f.Context, policyName+"-x-"+vclusterNamespace+"-x-"+hostNamespace, metav1.DeleteOptions{})
	})

	ginkgo.It("should sync NetworkPolicy from vCluster to host", func() {
		gomega.Eventually(func() bool {
			_, err := f.VClusterClient.NetworkingV1().NetworkPolicies(vclusterNamespace).Get(f.Context, policyName, metav1.GetOptions{})
			return err == nil
		}).
			WithTimeout(time.Minute).
			WithPolling(time.Second).
			Should(gomega.BeTrue(), "Timed out waiting for NetworkPolicy to sync to host")
		ginkgo.By("Waiting for NetworkPolicy to appear in host cluster")
		gomega.Eventually(func() bool {
			_, err := f.HostClient.NetworkingV1().NetworkPolicies(hostNamespace).Get(f.Context, policyName+"-x-"+vclusterNamespace+"-x-"+hostNamespace, metav1.GetOptions{})
			return err == nil
		}).
			WithTimeout(time.Minute).
			WithPolling(time.Second).
			Should(gomega.BeTrue(), "Timed out waiting for NetworkPolicy to sync to host")
	})
})
