package test_vind

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"

	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/util/random"
)

var _ = Describe("Docker driver", labels.Vind, func() {
	var (
		clusterName string
		ctx         context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		clusterName = "e2e-vind-" + random.String(6)

		DeferCleanup(func() {
			cleanupCtx := context.Background()
			_, _ = runVCluster(cleanupCtx, "delete", clusterName, "--driver", "docker", "--ignore-not-found", "--delete-context")
		})
	})

	It("should create, connect, and delete a vcluster", func() {
		var (
			cpContainer  string
			network      string
			volumePrefix string
		)

		By("Creating vcluster with docker driver", func() {
			_, err := runVCluster(ctx, "create", clusterName, "--driver", "docker", "--connect=false", "--chart-version", getChartVersion())
			Expect(err).To(Succeed())
		})

		By("Verifying control plane container is running", func() {
			cpContainer = controlPlaneContainerName(clusterName)
			Expect(dockerContainerRunning(ctx, cpContainer)).To(BeTrue(), "control plane container should be running")
		})

		By("Verifying Docker network exists", func() {
			network = networkName(clusterName)
			Expect(dockerNetworkExists(ctx, network)).To(BeTrue(), "docker network should exist")
		})

		By("Verifying control plane volumes exist", func() {
			volumePrefix = controlPlaneVolumePrefix(clusterName)
			Expect(dockerVolumesExist(ctx, volumePrefix)).To(BeTrue(), "control plane volumes should exist")
		})

		By("Connecting to vcluster and verifying API server is reachable", func() {
			_, err := runVCluster(ctx, "connect", clusterName, "--driver", "docker", "--update-current")
			Expect(err).To(Succeed())

			kubeContext := "vcluster-docker_" + clusterName
			Eventually(func(g Gomega) {
				client, err := kubeClientForContext(kubeContext)
				g.Expect(err).To(Succeed(), "failed to create kube client")
				nsList, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
				g.Expect(err).To(Succeed(), "list namespaces failed")
				names := make([]string, 0, len(nsList.Items))
				for _, ns := range nsList.Items {
					names = append(names, ns.Name)
				}
				g.Expect(names).To(ContainElement("default"))
			}).
				WithPolling(constants.PollingInterval).
				WithTimeout(constants.PollingTimeoutLong).
				Should(Succeed())
		})

		By("Disconnecting from vcluster", func() {
			_, err := runVCluster(ctx, "disconnect")
			Expect(err).To(Succeed())
		})

		By("Deleting vcluster", func() {
			_, err := runVCluster(ctx, "delete", clusterName, "--driver", "docker", "--delete-context")
			Expect(err).To(Succeed())
		})

		By("Verifying control plane container is gone", func() {
			Expect(dockerContainerExists(ctx, cpContainer)).To(BeFalse(), "control plane container should be removed")
		})

		By("Verifying Docker network is gone", func() {
			Expect(dockerNetworkExists(ctx, network)).To(BeFalse(), "docker network should be removed")
		})

		By("Verifying volumes are gone", func() {
			Expect(dockerVolumesExist(ctx, volumePrefix)).To(BeFalse(), "control plane volumes should be removed")
		})

		By("Verifying no containers with cluster name prefix remain", func() {
			out, err := exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", "name=^vcluster.cp."+clusterName+"$", "--format", "{{.ID}}").Output()
			Expect(err).To(Succeed())
			Expect(strings.TrimSpace(string(out))).To(BeEmpty(), "no control plane containers should remain")

			out, err = exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", "name=^vcluster.node."+clusterName+"\\.", "--format", "{{.ID}}").Output()
			Expect(err).To(Succeed())
			Expect(strings.TrimSpace(string(out))).To(BeEmpty(), "no worker node containers should remain")

			out, err = exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", "name=^vcluster.lb."+clusterName+"\\.", "--format", "{{.ID}}").Output()
			Expect(err).To(Succeed())
			Expect(strings.TrimSpace(string(out))).To(BeEmpty(), "no load balancer containers should remain")
		})
	})

	It("should assign an external IP to a LoadBalancer service and serve HTTP traffic", func() {
		clusterName = "e2e-vind-lb-" + random.String(6)
		kubeContext := "vcluster-docker_" + clusterName
		var client kubernetes.Interface

		By("Creating vcluster with docker driver", func() {
			_, err := runVCluster(ctx, "create", clusterName, "--driver", "docker", "--connect=false", "--chart-version", getChartVersion())
			Expect(err).To(Succeed())
		})

		By("Connecting to vcluster and waiting for API server", func() {
			_, err := runVCluster(ctx, "connect", clusterName, "--driver", "docker", "--update-current")
			Expect(err).To(Succeed())

			Eventually(func(g Gomega) {
				var clientErr error
				client, clientErr = kubeClientForContext(kubeContext)
				g.Expect(clientErr).To(Succeed(), "failed to create kube client")
				nsList, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
				g.Expect(err).To(Succeed(), "list namespaces failed")
				names := make([]string, 0, len(nsList.Items))
				for _, ns := range nsList.Items {
					names = append(names, ns.Name)
				}
				g.Expect(names).To(ContainElement("default"))
			}).
				WithPolling(constants.PollingInterval).
				WithTimeout(constants.PollingTimeoutLong).
				Should(Succeed())
		})

		By("Deploying nginx pod", func() {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "nginx",
					Labels: map[string]string{"run": "nginx"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "nginx",
						Image: "nginx:stable",
						Ports: []corev1.ContainerPort{{ContainerPort: 80}},
					}},
				},
			}
			_, err := client.CoreV1().Pods("default").Create(ctx, pod, metav1.CreateOptions{})
			Expect(err).To(Succeed(), "create nginx pod failed")
		})

		By("Exposing nginx pod as LoadBalancer service", func() {
			svc := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Name: "nginx-lb"},
				Spec: corev1.ServiceSpec{
					Type:     corev1.ServiceTypeLoadBalancer,
					Selector: map[string]string{"run": "nginx"},
					Ports: []corev1.ServicePort{{
						Port:       80,
						TargetPort: intstr.FromInt32(80),
					}},
				},
			}
			_, err := client.CoreV1().Services("default").Create(ctx, svc, metav1.CreateOptions{})
			Expect(err).To(Succeed(), "create LoadBalancer service failed")
		})

		var externalIP string

		By("Waiting for LoadBalancer external IP to be assigned", func() {
			Eventually(func(g Gomega) {
				svc, err := client.CoreV1().Services("default").Get(ctx, "nginx-lb", metav1.GetOptions{})
				g.Expect(err).To(Succeed())
				ingress := svc.Status.LoadBalancer.Ingress
				g.Expect(ingress).NotTo(BeEmpty(), "LoadBalancer external IP should be assigned")
				externalIP = ingress[0].IP
				g.Expect(externalIP).NotTo(BeEmpty(), "LoadBalancer external IP should not be empty")
			}).
				WithPolling(constants.PollingInterval).
				WithTimeout(constants.PollingTimeoutLong).
				Should(Succeed())
		})

		By("Verifying nginx is reachable via HTTP from outside the cluster", func() {
			url := fmt.Sprintf("http://%s", externalIP)
			httpClient := &http.Client{Timeout: 5 * time.Second}

			Eventually(func(g Gomega) {
				resp, err := httpClient.Get(url) //nolint:gosec // test code, IP from trusted LB
				g.Expect(err).To(Succeed(), "HTTP GET to %s failed", url)
				defer resp.Body.Close() //nolint:errcheck // test code
				g.Expect(resp.StatusCode).To(Equal(http.StatusOK), "nginx should return HTTP 200")
			}).
				WithPolling(constants.PollingInterval).
				WithTimeout(constants.PollingTimeoutLong).
				Should(Succeed())
		})
	})
})
