package lifecycle

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/util/random"
)

var _ = Describe("Tenant cluster lifecycle", labels.Core, labels.PR, func() {
	Context("create, list and delete tenant cluster", Ordered, func() {
		// Ordered because each spec operates on the tenant cluster
		// created by the first spec, and the last spec deletes it.
		var (
			suffix      string
			clusterName string
			namespace   string
			hostClient  kubernetes.Interface
		)

		BeforeAll(func(ctx context.Context) {
			suffix = random.String(6)
			clusterName = "e2e-cld-" + suffix
			namespace = clusterName
			hostClient = hostKubeClient()
		})

		AfterAll(func(ctx context.Context) {
			_, err := runVClusterCmd(ctx, "delete", clusterName, "-n", namespace, "--delete-namespace", "--ignore-not-found")
			Expect(err).To(Succeed())
		})

		It("should create a tenant cluster", func(ctx context.Context) {
			By("Creating a tenant cluster", func() {
				_, err := runVClusterCmd(ctx, createArgs(clusterName, namespace)...)
				Expect(err).To(Succeed())
			})

			By("Waiting for tenant cluster to be ready", func() {
				waitForVClusterReady(ctx, hostClient, clusterName, namespace)
			})
		})

		It("should list the tenant cluster as Running", func(ctx context.Context) {
			Eventually(func(g Gomega, ctx context.Context) {
				entries, err := listVClusters(ctx, namespace)
				g.Expect(err).To(Succeed())
				found := findByName(entries, clusterName)
				g.Expect(found).NotTo(BeNil(), "tenant cluster %s not found in list", clusterName)
				g.Expect(found.Status).To(Equal(string(find.StatusRunning)),
					"tenant cluster %s has status %s, expected %s", clusterName, found.Status, find.StatusRunning)
			}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
		})

		It("should delete a running tenant cluster", func(ctx context.Context) {
			By("Deleting the running tenant cluster", func() {
				_, err := runVClusterCmd(ctx, "delete", clusterName,
					"-n", namespace, "--delete-namespace")
				Expect(err).To(Succeed())
			})

			By("Verifying namespace is gone", func() {
				Eventually(func(g Gomega, ctx context.Context) {
					_, err := hostClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
					g.Expect(kerrors.IsNotFound(err)).To(BeTrue(),
						"namespace %s should be deleted", namespace)
				}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
			})
		})
	})

	Context("list and delete a scaled-down tenant cluster", Ordered, func() {
		// Ordered because each spec operates on the tenant cluster
		// created by the first spec. The second spec scales it down,
		// and subsequent specs depend on it being scaled down.
		var (
			suffix      string
			clusterName string
			namespace   string
			hostClient  kubernetes.Interface
		)

		BeforeAll(func(ctx context.Context) {
			suffix = random.String(6)
			clusterName = "e2e-csld-" + suffix
			namespace = clusterName
			hostClient = hostKubeClient()
		})

		AfterAll(func(ctx context.Context) {
			_, err := runVClusterCmd(ctx, "delete", clusterName, "-n", namespace, "--delete-namespace", "--ignore-not-found")
			Expect(err).To(Succeed())
		})

		It("should create a tenant cluster", func(ctx context.Context) {
			By("Creating a tenant cluster", func() {
				_, err := runVClusterCmd(ctx, createArgs(clusterName, namespace)...)
				Expect(err).To(Succeed())
			})

			By("Waiting for tenant cluster to be ready", func() {
				waitForVClusterReady(ctx, hostClient, clusterName, namespace)
			})
		})

		It("should list the tenant cluster as ScaledDown after scaling down", func(ctx context.Context) {
			By("Scaling down the tenant cluster StatefulSet to 0 replicas", func() {
				scaleDownVCluster(ctx, hostClient, clusterName, namespace)
			})

			By("Verifying it appears in list with ScaledDown status", func() {
				Eventually(func(g Gomega, ctx context.Context) {
					entries, err := listVClusters(ctx, namespace)
					g.Expect(err).To(Succeed())
					found := findByName(entries, clusterName)
					g.Expect(found).NotTo(BeNil(), "tenant cluster %s not found in list", clusterName)
					g.Expect(found.Status).To(Equal(string(find.StatusScaledDown)),
						"tenant cluster %s has status %s, expected %s", clusterName, found.Status, find.StatusScaledDown)
				}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
			})
		})

		It("should delete a scaled-down tenant cluster", func(ctx context.Context) {
			By("Deleting the scaled-down tenant cluster", func() {
				_, err := runVClusterCmd(ctx, "delete", clusterName,
					"-n", namespace, "--delete-namespace")
				Expect(err).To(Succeed())
			})

			By("Verifying namespace is gone", func() {
				Eventually(func(g Gomega, ctx context.Context) {
					_, err := hostClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
					g.Expect(kerrors.IsNotFound(err)).To(BeTrue(),
						"namespace %s should be deleted", namespace)
				}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
			})
		})
	})
})
