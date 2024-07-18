package networkpolicies

import (
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/util/retry"
)

var _ = ginkgo.Describe("NetworkPolicies are created as expected", func() {
	var (
		f            *framework.Framework
		iteration    int
		nsA          *corev1.Namespace
		nsB          *corev1.Namespace
		curlPod      *corev1.Pod
		nginxService *corev1.Service
		nginxPod     *corev1.Pod
	)

	ginkgo.JustBeforeEach(func() {
		// use default framework
		f = framework.DefaultFramework
		iteration++
		nsNameA := fmt.Sprintf("e2e-syncer-networkpolicies-a-%d-%s", iteration, random.String(5))
		nsNameB := fmt.Sprintf("e2e-syncer-networkpolicies-b-%d-%s", iteration, random.String(5))

		// create test namespaces with different labels
		var err error
		nsA, err = f.VClusterClient.CoreV1().Namespaces().Create(f.Context, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Name:   nsNameA,
			Labels: map[string]string{"key-a": fmt.Sprintf("e2e-syncer-networkpolicies-aaa-%d", iteration)},
		}}, metav1.CreateOptions{})
		framework.ExpectNoError(err)
		nsB, err = f.VClusterClient.CoreV1().Namespaces().Create(f.Context, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Name:   nsNameB,
			Labels: map[string]string{"key-b": fmt.Sprintf("e2e-syncer-networkpolicies-bbb-%d", iteration)},
		}}, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		curlPod, err = f.CreateCurlPod(nsNameA)
		framework.ExpectNoError(err)

		nginxPod, nginxService, err = f.CreateNginxPodAndService(nsNameB)
		framework.ExpectNoError(err)
		err = f.WaitForPodRunning(nginxPod.GetName(), nginxPod.GetNamespace())
		framework.ExpectNoError(err, "A pod created in the vcluster is expected to be in the Running phase eventually.")

		err = f.WaitForPodRunning(curlPod.GetName(), curlPod.GetNamespace())
		framework.ExpectNoError(err, "A pod created in the vcluster is expected to be in the Running phase eventually.")
	})

	ginkgo.AfterEach(func() {
		// delete test namespace
		err := f.DeleteTestNamespace(nsA.GetName(), true)
		framework.ExpectNoError(err)
		err = f.DeleteTestNamespace(nsB.GetName(), true)
		framework.ExpectNoError(err)
	})

	ginkgo.It("Test Egress NetworkPolicy works as expected", func() {
		// no NetworkPolicy yet - verify communication to test service
		framework.DefaultFramework.TestServiceIsEventuallyReachable(curlPod, nginxService)

		// create a policy that will allow Egress to the coreDNS so we can use .svc urls
		_, err := f.CreateEgressNetworkPolicyForDNS(f.Context, nsA.GetName())
		framework.ExpectNoError(err)

		f.Log.Info("deny all Egress from the Namespace that hosts curl pod")
		networkPolicy, err := f.VClusterClient.NetworkingV1().NetworkPolicies(nsA.GetName()).Create(f.Context, &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{Namespace: nsA.GetName(), Name: "my-egress-policy"},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{},
				PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
			},
		}, metav1.CreateOptions{})
		framework.ExpectNoError(err)
		// sleep to reduce the rate of pod/exec calls made when checking if service is reacheable
		time.Sleep(time.Second * 10)
		framework.DefaultFramework.TestServiceIsEventuallyUnreachable(curlPod, nginxService)

		f.Log.Info("allow Egress to the testService Namespace")
		err = updateNetworkPolicyWithRetryOnConflict(f, networkPolicy, func(np *networkingv1.NetworkPolicy) {
			np.Spec.Egress = []networkingv1.NetworkPolicyEgressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{{Port: &intstr.IntOrString{Type: intstr.Int, IntVal: nginxService.Spec.Ports[0].Port}}},
					To: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: nsB.GetLabels(),
							},
						},
					},
				},
			}
		})
		framework.ExpectNoError(err)
		// sleep to reduce the rate of pod/exec calls made when checking if service is reacheable
		time.Sleep(time.Second * 10)
		framework.DefaultFramework.TestServiceIsEventuallyReachable(curlPod, nginxService)

		f.Log.Info("deny Egress to the testService by using non matching pod selector")
		err = updateNetworkPolicyWithRetryOnConflict(f, networkPolicy, func(np *networkingv1.NetworkPolicy) {
			np.Spec.Egress = []networkingv1.NetworkPolicyEgressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{{Port: &intstr.IntOrString{Type: intstr.Int, IntVal: nginxService.Spec.Ports[0].Port}}},
					To: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: nsB.GetLabels(),
							},
							PodSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"wrong": "label"}},
						},
					},
				},
			}
		})
		framework.ExpectNoError(err)
		// sleep to reduce the rate of pod/exec calls made when checking if service is reacheable
		time.Sleep(time.Second * 10)
		framework.DefaultFramework.TestServiceIsEventuallyUnreachable(curlPod, nginxService)

		f.Log.Info("allow Egress to the testService Namespace and nginx pod")
		err = updateNetworkPolicyWithRetryOnConflict(f, networkPolicy, func(np *networkingv1.NetworkPolicy) {
			np.Spec.Egress = []networkingv1.NetworkPolicyEgressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{{Port: &intstr.IntOrString{Type: intstr.Int, IntVal: nginxService.Spec.Ports[0].Port}}},
					To: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: nsB.GetLabels(),
							},
							PodSelector: &metav1.LabelSelector{MatchLabels: nginxPod.GetLabels()},
						},
					},
				},
			}
		})
		framework.ExpectNoError(err)
		// sleep to reduce the rate of pod/exec calls made when checking if service is reacheable
		time.Sleep(time.Second * 10)
		framework.DefaultFramework.TestServiceIsEventuallyReachable(curlPod, nginxService)

		f.Log.Info("deny Egress on the nginx port (by allowing Egress only on a different pod)")
		err = updateNetworkPolicyWithRetryOnConflict(f, networkPolicy, func(np *networkingv1.NetworkPolicy) {
			np.Spec.Egress = []networkingv1.NetworkPolicyEgressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{{Port: &intstr.IntOrString{Type: intstr.Int, IntVal: 1}}},
					To: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: nsB.GetLabels(),
							},
							PodSelector: &metav1.LabelSelector{MatchLabels: nginxPod.GetLabels()},
						},
					},
				},
			}
		})
		framework.ExpectNoError(err)
		// sleep to reduce the rate of pod/exec calls made when checking if service is reacheable
		time.Sleep(time.Second * 10)
		framework.DefaultFramework.TestServiceIsEventuallyUnreachable(curlPod, nginxService)
	})

})

func updateNetworkPolicyWithRetryOnConflict(f *framework.Framework, networkPolicy *networkingv1.NetworkPolicy, mutator func(np *networkingv1.NetworkPolicy)) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var err error
		networkPolicy, err = f.VClusterClient.NetworkingV1().NetworkPolicies(networkPolicy.GetNamespace()).Get(f.Context, networkPolicy.GetName(), metav1.GetOptions{})
		if err != nil {
			return err
		}

		mutator(networkPolicy)

		networkPolicy, err = f.VClusterClient.NetworkingV1().NetworkPolicies(networkPolicy.GetNamespace()).Update(f.Context, networkPolicy, metav1.UpdateOptions{})
		return err
	})
}
