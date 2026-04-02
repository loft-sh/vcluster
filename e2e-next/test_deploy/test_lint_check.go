package test_deploy

import (
	"context"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// This test intentionally uses the auto-registration pattern (var _ = Describe)
// with cluster.Use, which violates the describefunc linter rule.
// The linter requires wrapping this in an exported function like:
//
//	func DescribeLintCheck(vcluster suite.Dependency) bool { return Describe(...) }
var _ = Describe("Lint check - should fail describefunc linter",
	labels.Deploy,
	cluster.Use(clusters.CommonVCluster),
	func() {
		var vClusterClient kubernetes.Interface

		BeforeEach(func(ctx context.Context) {
			vClusterClient = cluster.CurrentKubeClientFrom(ctx)
			Expect(vClusterClient).NotTo(BeNil())
		})

		It("reads the default namespace", func(ctx context.Context) {
			ns, err := vClusterClient.CoreV1().Namespaces().Get(ctx, "default", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(ns.Name).To(Equal("default"))
		})
	},
)
