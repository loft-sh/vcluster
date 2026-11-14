package isolation

import (
	"context"
	"fmt"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e/constants"
	"github.com/loft-sh/vcluster/e2e/labels"
	pkgconstants "github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/util/podhelper"
	"github.com/loft-sh/vcluster/pkg/util/random"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
)

// ControlPlaneDNSSpec registers a test that the control plane NetworkPolicy lets the
// control plane resolve names through CoreDNS labeled k8s-app=vcluster-kube-dns.
// The tenant cluster must have policies.networkPolicy enabled. Register it only in
// suites that install the chart from this tree, because it tests a rule that chart renders.
func ControlPlaneDNSSpec() {
	Describe("Control plane DNS egress",
		labels.PR,
		labels.Security,
		func() {
			var (
				hostClient        kubernetes.Interface
				hostConfig        *rest.Config
				vClusterName      string
				vClusterNamespace string
			)

			BeforeEach(func(ctx context.Context) {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				hostConfig = cluster.From(ctx, constants.GetHostClusterName()).KubernetesRestConfig()
				Expect(hostConfig).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
				// vcluster create installs into "vcluster-<name>" unless told otherwise.
				vClusterNamespace = "vcluster-" + vClusterName
			})

			It("lets the control plane resolve names through CoreDNS labeled vcluster-kube-dns", func(ctx context.Context) {
				// vCluster Standalone runs its own CoreDNS in kube-system, labeled
				// k8s-app=vcluster-kube-dns instead of kube-dns. Run one next to the kind
				// CoreDNS and resolve through it from the control plane pod, whose egress
				// the vc-cp NetworkPolicy controls.
				const (
					recordName  = "netpol-dns.vcluster-e2e.test"
					recordIP    = "192.0.2.53"
					selectorKey = "e2e.vcluster.loft.sh/netpol-dns"
				)
				suffix := random.String(6)
				dnsName := "netpol-dns-" + suffix

				var serviceIP, controlPlanePod string
				lookup := func(ctx context.Context) (string, error) {
					execCtx, cancel := context.WithTimeout(ctx, constants.PollingTimeoutShort)
					defer cancel()
					stdout, _, err := podhelper.ExecBuffered(execCtx, hostConfig, vClusterNamespace, controlPlanePod, "syncer",
						[]string{"nslookup", recordName, serviceIP}, nil)
					return string(stdout), err
				}

				By("Running a CoreDNS server labeled vcluster-kube-dns in the control plane cluster's kube-system namespace", func() {
					hostCoreDNS, err := hostClient.AppsV1().Deployments(metav1.NamespaceSystem).Get(ctx, "coredns", metav1.GetOptions{})
					Expect(err).To(Succeed(), "failed to get the kind CoreDNS deployment to reuse its image")
					Expect(hostCoreDNS.Spec.Template.Spec.Containers).NotTo(BeEmpty())

					_, err = hostClient.CoreV1().ConfigMaps(metav1.NamespaceSystem).Create(ctx, &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{Name: dnsName},
						Data: map[string]string{
							"Corefile": fmt.Sprintf(".:1053 {\n    errors\n    hosts {\n        %s %s\n    }\n}\n", recordIP, recordName),
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					DeferCleanup(func(ctx context.Context) {
						err := hostClient.CoreV1().ConfigMaps(metav1.NamespaceSystem).Delete(ctx, dnsName, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).To(Succeed())
						}
					})

					_, err = hostClient.CoreV1().Pods(metav1.NamespaceSystem).Create(ctx, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: dnsName,
							Labels: map[string]string{
								pkgconstants.CoreDNSLabelKey: pkgconstants.CoreDNSLabelValue,
								selectorKey:                  suffix,
							},
						},
						Spec: corev1.PodSpec{
							TerminationGracePeriodSeconds: ptr.To(int64(1)),
							Containers: []corev1.Container{
								{
									Name:            "coredns",
									Image:           hostCoreDNS.Spec.Template.Spec.Containers[0].Image,
									ImagePullPolicy: corev1.PullIfNotPresent,
									Args:            []string{"-conf", "/etc/coredns/Corefile"},
									ReadinessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt32(1053)},
										},
									},
									VolumeMounts: []corev1.VolumeMount{{Name: "config", MountPath: "/etc/coredns", ReadOnly: true}},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "config",
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: dnsName}},
									},
								},
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					DeferCleanup(func(ctx context.Context) {
						err := hostClient.CoreV1().Pods(metav1.NamespaceSystem).Delete(ctx, dnsName, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).To(Succeed())
						}
					})

					svc, err := hostClient.CoreV1().Services(metav1.NamespaceSystem).Create(ctx, &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{Name: dnsName},
						Spec: corev1.ServiceSpec{
							Selector: map[string]string{selectorKey: suffix},
							Ports: []corev1.ServicePort{
								{Name: "dns", Protocol: corev1.ProtocolUDP, Port: 53, TargetPort: intstr.FromInt32(1053)},
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					DeferCleanup(func(ctx context.Context) {
						err := hostClient.CoreV1().Services(metav1.NamespaceSystem).Delete(ctx, dnsName, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).To(Succeed())
						}
					})
					serviceIP = svc.Spec.ClusterIP

					Eventually(func(g Gomega) {
						pod, err := hostClient.CoreV1().Pods(metav1.NamespaceSystem).Get(ctx, dnsName, metav1.GetOptions{})
						g.Expect(err).To(Succeed(), "failed to get DNS pod %s: %v", dnsName, err)
						g.Expect(pod.Status.Conditions).To(ContainElement(And(
							HaveField("Type", corev1.PodReady),
							HaveField("Status", corev1.ConditionTrue),
						)), "DNS pod %s is not ready, phase: %s", dnsName, pod.Status.Phase)
					}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("Finding the control plane pod", func() {
					pods, err := hostClient.CoreV1().Pods(vClusterNamespace).List(ctx, metav1.ListOptions{
						LabelSelector: "app=vcluster,release=" + vClusterName,
					})
					Expect(err).To(Succeed())
					Expect(pods.Items).NotTo(BeEmpty(), "no control plane pod with release=%s in %s", vClusterName, vClusterNamespace)
					controlPlanePod = pods.Items[0].Name
				})

				By("Resolving a name through that server from the control plane pod", func() {
					Eventually(func(g Gomega) {
						out, err := lookup(ctx)
						g.Expect(err).To(Succeed(), "nslookup via %s failed: %s", serviceIP, out)
						g.Expect(out).To(ContainSubstring("Address: "+recordIP), "unexpected nslookup answer: %s", out)
					}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})

				// On a cluster whose CNI ignores NetworkPolicy the lookup above passes whatever
				// the selector says, so check the same server is blocked under another label.
				By("Checking the control plane can't reach the server once it carries another k8s-app label", func() {
					patch := fmt.Sprintf(`{"metadata":{"labels":{%q:"vcluster-e2e-not-dns"}}}`, pkgconstants.CoreDNSLabelKey)
					_, err := hostClient.CoreV1().Pods(metav1.NamespaceSystem).Patch(ctx, dnsName, types.MergePatchType, []byte(patch), metav1.PatchOptions{})
					Expect(err).To(Succeed())

					Eventually(func(g Gomega) {
						out, err := lookup(ctx)
						g.Expect(err).To(MatchError(ContainSubstring("exit code 1")), "nslookup via %s still resolved: %s", serviceIP, out)
						g.Expect(out).To(ContainSubstring("connection timed out"), "nslookup failed for another reason: %s", out)
					}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})
		},
	)
}
