package test_core

import (
	"context"
	"fmt"
	"sort"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

var _ = Describe("Service replication and sync",
	Ordered,
	labels.Core,
	labels.Sync,
	labels.PR,
	cluster.Use(clusters.ServiceSyncVCluster),
	cluster.Use(clusters.HostCluster),
	func() {
		var (
			hostClient        kubernetes.Interface
			vClusterClient    kubernetes.Interface
			vClusterName      = clusters.ServiceSyncVClusterName
			vClusterNamespace = "vcluster-" + vClusterName
		)

		BeforeAll(func(ctx context.Context) {
			hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
			Expect(hostClient).NotTo(BeNil())
			vClusterClient = cluster.CurrentKubeClientFrom(ctx)
			Expect(vClusterClient).NotTo(BeNil())
		})

		It("replicates service and endpoints from host namespace into vcluster", func(ctx context.Context) {
			const (
				fromNS   = "test"
				fromName = "test"
				toNS     = "default"
				toName   = "test"
			)

			By("creating source namespace on host")
			_, err := hostClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: fromNS},
			}, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func(ctx context.Context) {
				Expect(hostClient.CoreV1().Namespaces().Delete(ctx, fromNS, metav1.DeleteOptions{})).To(Succeed())
				Eventually(func(g Gomega) {
					_, err := hostClient.CoreV1().Namespaces().Get(ctx, fromNS, metav1.GetOptions{})
					g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
			})

			By("creating source service on host")
			fromService, err := hostClient.CoreV1().Services(fromNS).Create(ctx, &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Name: fromName, Namespace: fromNS},
				Spec: corev1.ServiceSpec{
					Selector: map[string]string{"test": "test"},
					Ports: []corev1.ServicePort{
						{Name: "custom", Port: 8080, Protocol: corev1.ProtocolTCP},
					},
				},
			}, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("waiting for the replicated service to appear in vcluster")
			var toService *corev1.Service
			Eventually(func(g Gomega) {
				toService, err = vClusterClient.CoreV1().Services(toNS).Get(ctx, toName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

			By("asserting the replicated service has correct ports")
			Expect(toService.Spec.Ports).To(HaveLen(1))
			Expect(toService.Spec.Ports[0].Name).To(Equal("custom"))
			Expect(toService.Spec.Ports[0].Port).To(Equal(int32(8080)))

			By("waiting for the replicated endpoint to appear in vcluster")
			//nolint:staticcheck
			var toEndpoints *corev1.Endpoints
			Eventually(func(g Gomega) {
				//nolint:staticcheck
				toEndpoints, err = vClusterClient.CoreV1().Endpoints(toNS).Get(ctx, toName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(toEndpoints.Subsets).To(HaveLen(1))
				g.Expect(toEndpoints.Subsets[0].Addresses).To(HaveLen(1))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

			By("asserting endpoint IP equals source service ClusterIP")
			Expect(toEndpoints.Subsets[0].Addresses[0].IP).To(Equal(fromService.Spec.ClusterIP))

			By("deleting the source service on host")
			Expect(hostClient.CoreV1().Services(fromNS).Delete(ctx, fromName, metav1.DeleteOptions{})).To(Succeed())

			By("waiting for the replicated service to be removed from vcluster")
			Eventually(func(g Gomega) {
				_, err := vClusterClient.CoreV1().Services(toNS).Get(ctx, toName, metav1.GetOptions{})
				g.Expect(kerrors.IsNotFound(err)).To(BeTrue(), "replicated service should be deleted after source is gone")
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

			By("waiting for the replicated endpoint to be removed from vcluster")
			Eventually(func(g Gomega) {
				//nolint:staticcheck
				_, err := vClusterClient.CoreV1().Endpoints(toNS).Get(ctx, toName, metav1.GetOptions{})
				g.Expect(kerrors.IsNotFound(err)).To(BeTrue(), "replicated endpoint should be deleted after source is gone")
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
		})

		It("maps service from vcluster namespace to host with translated selector", func(ctx context.Context) {
			const (
				fromNS   = "test"
				fromName = "test"
				toName   = "test"
			)

			By("creating source namespace in vcluster")
			_, err := vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: fromNS},
			}, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func(ctx context.Context) {
				Expect(vClusterClient.CoreV1().Namespaces().Delete(ctx, fromNS, metav1.DeleteOptions{})).To(Succeed())
				Eventually(func(g Gomega) {
					_, err := vClusterClient.CoreV1().Namespaces().Get(ctx, fromNS, metav1.GetOptions{})
					g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
			})

			By("creating source service in vcluster")
			_, err = vClusterClient.CoreV1().Services(fromNS).Create(ctx, &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Name: fromName, Namespace: fromNS},
				Spec: corev1.ServiceSpec{
					Selector: map[string]string{"test": "test"},
					Ports: []corev1.ServicePort{
						{Name: "custom", Port: 8080, Protocol: corev1.ProtocolTCP},
					},
				},
			}, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("waiting for the mapped service to appear on host")
			var toService *corev1.Service
			Eventually(func(g Gomega) {
				toService, err = hostClient.CoreV1().Services(vClusterNamespace).Get(ctx, toName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

			By("asserting the mapped service has correct ports")
			Expect(toService.Spec.Ports).To(HaveLen(1))
			Expect(toService.Spec.Ports[0].Name).To(Equal("custom"))
			Expect(toService.Spec.Ports[0].Port).To(Equal(int32(8080)))

			By("asserting the mapped service selector is translated to host labels")
			Expect(toService.Spec.Selector).To(HaveLen(3))
			Expect(toService.Spec.Selector[translate.NamespaceLabel]).To(Equal(fromNS))
			// vClusterName holds the actual vcluster name used at deploy time;
			// translate.VClusterName is only valid inside the vcluster process itself.
			Expect(toService.Spec.Selector[translate.MarkerLabel]).To(Equal(vClusterName))
			Expect(toService.Spec.Selector[translate.HostLabel("test")]).To(Equal("test"))

			By("deleting the source service in vcluster")
			Expect(vClusterClient.CoreV1().Services(fromNS).Delete(ctx, fromName, metav1.DeleteOptions{})).To(Succeed())

			By("waiting for the mapped service to be removed from host")
			Eventually(func(g Gomega) {
				_, err := hostClient.CoreV1().Services(vClusterNamespace).Get(ctx, toName, metav1.GetOptions{})
				g.Expect(kerrors.IsNotFound(err)).To(BeTrue(), "mapped service should be deleted after source is gone")
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
		})

		It("syncs endpoints of headless service replicated from host into vcluster", func(ctx context.Context) {
			const (
				fromNS   = "test"
				fromName = "nginx"
				toNS     = "default"
				toName   = "nginx"
			)
			var two int32 = 2
			var zero int32

			By("creating source namespace on host")
			_, err := hostClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: fromNS},
			}, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func(ctx context.Context) {
				Expect(hostClient.CoreV1().Namespaces().Delete(ctx, fromNS, metav1.DeleteOptions{})).To(Succeed())
				Eventually(func(g Gomega) {
					_, err := hostClient.CoreV1().Namespaces().Get(ctx, fromNS, metav1.GetOptions{})
					g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
			})

			By("creating an nginx deployment on host")
			_, err = hostClient.AppsV1().Deployments(fromNS).Create(ctx, &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: fromName, Namespace: fromNS},
				Spec: appsv1.DeploymentSpec{
					Replicas: &two,
					Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "nginx"}},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "nginx"}},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{Name: "nginx", Image: "nginx"}},
						},
					},
				},
			}, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("creating a headless service targeting the deployment on host")
			_, err = hostClient.CoreV1().Services(fromNS).Create(ctx, &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Name: fromName, Namespace: fromNS},
				Spec: corev1.ServiceSpec{
					Selector:  map[string]string{"app": "nginx"},
					ClusterIP: corev1.ClusterIPNone,
				},
			}, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("waiting for the replicated service to appear in vcluster")
			Eventually(func(g Gomega) {
				_, err := vClusterClient.CoreV1().Services(toNS).Get(ctx, toName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

			By(fmt.Sprintf("waiting for %d endpoints on both sides to be populated", two))
			//nolint:staticcheck
			var fromEPs, toEPs *corev1.Endpoints
			Eventually(func(g Gomega) {
				//nolint:staticcheck
				fromEPs, err = hostClient.CoreV1().Endpoints(fromNS).Get(ctx, fromName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fromEPs.Subsets).To(HaveLen(1))
				g.Expect(fromEPs.Subsets[0].Addresses).To(HaveLen(int(two)))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())

			Eventually(func(g Gomega) {
				//nolint:staticcheck
				toEPs, err = vClusterClient.CoreV1().Endpoints(toNS).Get(ctx, toName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(toEPs.Subsets).To(HaveLen(1))
				g.Expect(toEPs.Subsets[0].Addresses).To(HaveLen(int(two)))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())

			By("asserting endpoint IPs match between host and vcluster (order-independent)")
			Expect(endpointIPs(fromEPs.Subsets[0].Addresses)).To(ConsistOf(endpointIPs(toEPs.Subsets[0].Addresses)))

			By("scaling the deployment to 0")
			Eventually(func(g Gomega) {
				dep, err := hostClient.AppsV1().Deployments(fromNS).Get(ctx, fromName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				dep.Spec.Replicas = &zero
				_, err = hostClient.AppsV1().Deployments(fromNS).Update(ctx, dep, metav1.UpdateOptions{})
				g.Expect(err).NotTo(HaveOccurred())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			By("waiting for host endpoints to be cleared")
			Eventually(func(g Gomega) {
				//nolint:staticcheck
				ep, err := hostClient.CoreV1().Endpoints(fromNS).Get(ctx, fromName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(ep.Subsets).To(BeNil())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

			By("waiting for vcluster endpoints to also be cleared")
			Eventually(func(g Gomega) {
				//nolint:staticcheck
				ep, err := vClusterClient.CoreV1().Endpoints(toNS).Get(ctx, toName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(ep.Subsets).To(BeNil())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

			By(fmt.Sprintf("scaling the deployment back to %d", two))
			Eventually(func(g Gomega) {
				dep, err := hostClient.AppsV1().Deployments(fromNS).Get(ctx, fromName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				dep.Spec.Replicas = &two
				_, err = hostClient.AppsV1().Deployments(fromNS).Update(ctx, dep, metav1.UpdateOptions{})
				g.Expect(err).NotTo(HaveOccurred())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			By(fmt.Sprintf("waiting for %d endpoints to be repopulated on both sides", two))
			Eventually(func(g Gomega) {
				//nolint:staticcheck
				fromEPs, err = hostClient.CoreV1().Endpoints(fromNS).Get(ctx, fromName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fromEPs.Subsets).To(HaveLen(1))
				g.Expect(fromEPs.Subsets[0].Addresses).To(HaveLen(int(two)))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())

			Eventually(func(g Gomega) {
				//nolint:staticcheck
				toEPs, err = vClusterClient.CoreV1().Endpoints(toNS).Get(ctx, toName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(toEPs.Subsets).To(HaveLen(1))
				g.Expect(toEPs.Subsets[0].Addresses).To(HaveLen(int(two)))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())

			By("asserting endpoint IPs still match after scale-up")
			Expect(endpointIPs(fromEPs.Subsets[0].Addresses)).To(ConsistOf(endpointIPs(toEPs.Subsets[0].Addresses)))
		})

		It("syncs endpoints of headless service mapped from vcluster to host", func(ctx context.Context) {
			const (
				fromNS   = "test"
				fromName = "nginx"
				toName   = "nginx"
			)
			var two int32 = 2
			var zero int32

			By("creating source namespace in vcluster")
			_, err := vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: fromNS},
			}, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func(ctx context.Context) {
				Expect(vClusterClient.CoreV1().Namespaces().Delete(ctx, fromNS, metav1.DeleteOptions{})).To(Succeed())
				Eventually(func(g Gomega) {
					_, err := vClusterClient.CoreV1().Namespaces().Get(ctx, fromNS, metav1.GetOptions{})
					g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
			})

			By("creating an nginx deployment in vcluster")
			_, err = vClusterClient.AppsV1().Deployments(fromNS).Create(ctx, &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: fromName, Namespace: fromNS},
				Spec: appsv1.DeploymentSpec{
					Replicas: &two,
					Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "nginx"}},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "nginx"}},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{Name: "nginx", Image: "nginx"}},
						},
					},
				},
			}, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("creating a headless service targeting the deployment in vcluster")
			_, err = vClusterClient.CoreV1().Services(fromNS).Create(ctx, &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Name: fromName, Namespace: fromNS},
				Spec: corev1.ServiceSpec{
					Selector:  map[string]string{"app": "nginx"},
					ClusterIP: corev1.ClusterIPNone,
				},
			}, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("waiting for the mapped service to appear on host")
			Eventually(func(g Gomega) {
				_, err := hostClient.CoreV1().Services(vClusterNamespace).Get(ctx, toName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

			By(fmt.Sprintf("waiting for %d endpoints on both sides to be populated", two))
			//nolint:staticcheck
			var fromEPs, toEPs *corev1.Endpoints
			Eventually(func(g Gomega) {
				//nolint:staticcheck
				fromEPs, err = vClusterClient.CoreV1().Endpoints(fromNS).Get(ctx, fromName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fromEPs.Subsets).To(HaveLen(1))
				g.Expect(fromEPs.Subsets[0].Addresses).To(HaveLen(int(two)))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())

			Eventually(func(g Gomega) {
				//nolint:staticcheck
				toEPs, err = hostClient.CoreV1().Endpoints(vClusterNamespace).Get(ctx, toName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(toEPs.Subsets).To(HaveLen(1))
				g.Expect(toEPs.Subsets[0].Addresses).To(HaveLen(int(two)))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())

			By("asserting endpoint IPs match between vcluster and host (order-independent)")
			Expect(endpointIPs(fromEPs.Subsets[0].Addresses)).To(ConsistOf(endpointIPs(toEPs.Subsets[0].Addresses)))

			By("scaling the deployment to 0")
			Eventually(func(g Gomega) {
				dep, err := vClusterClient.AppsV1().Deployments(fromNS).Get(ctx, fromName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				dep.Spec.Replicas = &zero
				_, err = vClusterClient.AppsV1().Deployments(fromNS).Update(ctx, dep, metav1.UpdateOptions{})
				g.Expect(err).NotTo(HaveOccurred())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			By("waiting for vcluster endpoints to be cleared")
			Eventually(func(g Gomega) {
				//nolint:staticcheck
				ep, err := vClusterClient.CoreV1().Endpoints(fromNS).Get(ctx, fromName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(ep.Subsets).To(BeNil())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

			By("waiting for host endpoints to also be cleared")
			Eventually(func(g Gomega) {
				//nolint:staticcheck
				ep, err := hostClient.CoreV1().Endpoints(vClusterNamespace).Get(ctx, toName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(ep.Subsets).To(BeNil())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

			By(fmt.Sprintf("scaling the deployment back to %d", two))
			Eventually(func(g Gomega) {
				dep, err := vClusterClient.AppsV1().Deployments(fromNS).Get(ctx, fromName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				dep.Spec.Replicas = &two
				_, err = vClusterClient.AppsV1().Deployments(fromNS).Update(ctx, dep, metav1.UpdateOptions{})
				g.Expect(err).NotTo(HaveOccurred())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			By(fmt.Sprintf("waiting for %d endpoints to be repopulated on both sides", two))
			Eventually(func(g Gomega) {
				//nolint:staticcheck
				fromEPs, err = vClusterClient.CoreV1().Endpoints(fromNS).Get(ctx, fromName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fromEPs.Subsets).To(HaveLen(1))
				g.Expect(fromEPs.Subsets[0].Addresses).To(HaveLen(int(two)))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())

			Eventually(func(g Gomega) {
				//nolint:staticcheck
				toEPs, err = hostClient.CoreV1().Endpoints(vClusterNamespace).Get(ctx, toName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(toEPs.Subsets).To(HaveLen(1))
				g.Expect(toEPs.Subsets[0].Addresses).To(HaveLen(int(two)))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())

			By("asserting endpoint IPs still match after scale-up")
			Expect(endpointIPs(fromEPs.Subsets[0].Addresses)).To(ConsistOf(endpointIPs(toEPs.Subsets[0].Addresses)))
		})

		It("syncs Service, Endpoints, and EndpointSlice to host when Endpoint is created before Service", func(ctx context.Context) {
			svcNS := "default"
			svcName := fmt.Sprintf("test-svc-sync-%d", GinkgoRandomSeed())
			translatedName := translate.SingleNamespaceHostName(svcName, svcNS, vClusterName)

			By("creating Endpoint in vcluster before the Service exists")
			//nolint:staticcheck
			_, err := vClusterClient.CoreV1().Endpoints(svcNS).Create(ctx, &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{Name: svcName, Namespace: svcNS},
				//nolint:staticcheck
				Subsets: []corev1.EndpointSubset{
					{
						Addresses: []corev1.EndpointAddress{{IP: "1.1.1.1"}},
						Ports:     []corev1.EndpointPort{{Port: 5000}},
					},
				},
			}, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func(ctx context.Context) {
				//nolint:staticcheck
				err := vClusterClient.CoreV1().Endpoints(svcNS).Delete(ctx, svcName, metav1.DeleteOptions{})
				if !kerrors.IsNotFound(err) {
					Expect(err).NotTo(HaveOccurred())
				}
			})

			By("creating headless Service in vcluster")
			_, err = vClusterClient.CoreV1().Services(svcNS).Create(ctx, &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Name: svcName, Namespace: svcNS},
				Spec: corev1.ServiceSpec{
					ClusterIP: corev1.ClusterIPNone,
					Ports: []corev1.ServicePort{
						{Name: "custom-port", Port: 8080, Protocol: corev1.ProtocolTCP, TargetPort: intstr.FromInt(5000)},
					},
				},
			}, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func(ctx context.Context) {
				err := vClusterClient.CoreV1().Services(svcNS).Delete(ctx, svcName, metav1.DeleteOptions{})
				if !kerrors.IsNotFound(err) {
					Expect(err).NotTo(HaveOccurred())
				}
			})

			By("waiting for Service to appear on host with correct ports")
			Eventually(func(g Gomega) {
				hostSvc, err := hostClient.CoreV1().Services(vClusterNamespace).Get(ctx, translatedName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(hostSvc.Spec.Ports).To(HaveLen(1))
				g.Expect(hostSvc.Spec.Ports[0].Name).To(Equal("custom-port"))
				g.Expect(hostSvc.Spec.Ports[0].Port).To(Equal(int32(8080)))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

			By("waiting for Endpoint to appear on host with correct address and port")
			Eventually(func(g Gomega) {
				//nolint:staticcheck
				hostEP, err := hostClient.CoreV1().Endpoints(vClusterNamespace).Get(ctx, translatedName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(hostEP.Subsets).To(HaveLen(1))
				g.Expect(hostEP.Subsets[0].Addresses).To(HaveLen(1))
				g.Expect(hostEP.Subsets[0].Addresses[0].IP).To(Equal("1.1.1.1"))
				g.Expect(hostEP.Subsets[0].Ports).To(HaveLen(1))
				g.Expect(hostEP.Subsets[0].Ports[0].Port).To(Equal(int32(5000)))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

			By("waiting for EndpointSlice to appear on host with correct content")
			Eventually(func(g Gomega) {
				slices, err := hostClient.DiscoveryV1().EndpointSlices(vClusterNamespace).List(ctx, metav1.ListOptions{
					LabelSelector: fmt.Sprintf("kubernetes.io/service-name=%s", translatedName),
				})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(slices.Items).To(HaveLen(1))
				g.Expect(slices.Items[0].Endpoints).NotTo(BeEmpty())
				g.Expect(slices.Items[0].Endpoints[0].Addresses).To(ConsistOf("1.1.1.1"))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

			By("deleting Service in vcluster and verifying host resources are cleaned up")
			Expect(vClusterClient.CoreV1().Services(svcNS).Delete(ctx, svcName, metav1.DeleteOptions{})).To(Succeed())

			Eventually(func(g Gomega) {
				_, err := hostClient.CoreV1().Services(vClusterNamespace).Get(ctx, translatedName, metav1.GetOptions{})
				g.Expect(kerrors.IsNotFound(err)).To(BeTrue(), "host service should be deleted")
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

			Eventually(func(g Gomega) {
				//nolint:staticcheck
				_, err := hostClient.CoreV1().Endpoints(vClusterNamespace).Get(ctx, translatedName, metav1.GetOptions{})
				g.Expect(kerrors.IsNotFound(err)).To(BeTrue(), "host endpoint should be deleted")
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

			Eventually(func(g Gomega) {
				slices, err := hostClient.DiscoveryV1().EndpointSlices(vClusterNamespace).List(ctx, metav1.ListOptions{
					LabelSelector: fmt.Sprintf("kubernetes.io/service-name=%s", translatedName),
				})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(slices.Items).To(BeEmpty(), "host EndpointSlice should be deleted")
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
		})

		// -----------------------------------------------------------------------
		// Stale replicated service cleanup on vcluster start
		// -----------------------------------------------------------------------
		It("removes previously-replicated service not present in replication config on startup", func(ctx context.Context) {
			// The service "test-replicated-service-cleanup" is pre-deployed in the vcluster's
			// "default" namespace via experimental.deploy.vcluster.manifests in vcluster-servicesync.yaml.
			// It carries the "vcluster.loft.sh/controlled-by: vcluster" label but is NOT listed in
			// networking.replicateServices.fromHost, so vcluster should delete it on startup.

			By("confirming the stale service is removed from vcluster default namespace")
			Eventually(func(g Gomega) {
				_, err := vClusterClient.CoreV1().Services("default").Get(ctx, "test-replicated-service-cleanup", metav1.GetOptions{})
				g.Expect(kerrors.IsNotFound(err)).To(BeTrue(), "stale replicated service should have been cleaned up")
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
		})
	},
)

// endpointIPs extracts IP addresses from a slice of EndpointAddresses.
func endpointIPs(addrs []corev1.EndpointAddress) []string {
	GinkgoHelper()
	ips := make([]string, 0, len(addrs))
	for _, a := range addrs {
		ips = append(ips, a.IP)
	}
	sort.Strings(ips)
	return ips
}
