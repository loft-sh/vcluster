package rootless

import (
	"context"
	"strings"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/util/podhelper"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var _ = Describe("Rootless mode",
	labels.Security,
	cluster.Use(clusters.RootlessVCluster),
	cluster.Use(clusters.HostCluster),
	func() {
		var (
			hostClient        kubernetes.Interface
			hostRestConfig    *rest.Config
			vClusterNamespace = "vcluster-" + clusters.RootlessVClusterName
		)

		BeforeEach(func(ctx context.Context) {
			hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
			Expect(hostClient).NotTo(BeNil())
			hostRestConfig = cluster.From(ctx, constants.GetHostClusterName()).KubernetesRestConfig()
			Expect(hostRestConfig).NotTo(BeNil())
		})

		It("verifies the syncer container runs as non-root user", func(ctx context.Context) {
			By("Listing vcluster pods in the rootless vcluster namespace", func() {
				pods, err := hostClient.CoreV1().Pods(vClusterNamespace).List(ctx, metav1.ListOptions{
					LabelSelector: "app=vcluster",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(pods.Items).NotTo(BeEmpty(), "expected at least one vcluster pod")

				By("Executing id -u in the syncer container", func() {
					vclusterPod := pods.Items[0].Name
					cmd := []string{"/bin/sh", "-c", "id -u"}
					stdout, stderr, err := podhelper.ExecBuffered(
						ctx,
						hostRestConfig,
						vClusterNamespace,
						vclusterPod,
						"syncer",
						cmd,
						nil,
					)
					Expect(err).NotTo(HaveOccurred())
					Expect(stderr).To(BeEmpty(), "expected no stderr output")
					Expect(strings.TrimSpace(string(stdout))).To(Equal("12345"), "expected syncer to run as UID 12345")
				})
			})
		})
	},
)
