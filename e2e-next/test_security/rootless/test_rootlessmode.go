// Package rootless contains rootless mode tests.
package rootless

import (
	"context"
	"strings"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/util/podhelper"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// RootlessModeSpec registers rootless mode tests.
func RootlessModeSpec() {
	Describe("vCluster rootless mode",
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
				vClusterNamespace = "vcluster-" + vClusterName
			})

			It("should run the syncer container as the non-root user configured in the security context", func(ctx context.Context) {
				var vclusterPodName string

				By("Finding a running vCluster pod", func() {
					Eventually(func(g Gomega) {
						pods, err := hostClient.CoreV1().Pods(vClusterNamespace).List(ctx, metav1.ListOptions{
							LabelSelector: "app=vcluster",
						})
						g.Expect(err).To(Succeed(), "failed to list vCluster pods in namespace %s: %v", vClusterNamespace, err)
						g.Expect(pods.Items).NotTo(BeEmpty(), "no vCluster pods found in namespace %s", vClusterNamespace)
						vclusterPodName = pods.Items[0].Name
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("Executing 'id -u' in the syncer container and asserting the UID matches the configured runAsUser", func() {
					cmd := []string{"/bin/sh", "-c", "id -u"}
					stdout, stderr, err := podhelper.ExecBuffered(
						ctx,
						hostConfig,
						vClusterNamespace,
						vclusterPodName,
						"syncer",
						cmd,
						nil,
					)
					Expect(err).To(Succeed(), "exec into pod %s/%s failed: %v", vClusterNamespace, vclusterPodName, err)
					Expect(stderr).To(BeEmpty(), "unexpected stderr output from 'id -u': %s", string(stderr))
					Expect(strings.TrimSuffix(string(stdout), "\n")).To(Equal("12345"),
						"expected syncer to run as UID 12345 (rootless mode), got: %s", string(stdout))
				})
			})
		},
	)
}
