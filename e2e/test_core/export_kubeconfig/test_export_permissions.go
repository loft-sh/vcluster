package export_kubeconfig

import (
	"context"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e/constants"
	"github.com/loft-sh/vcluster/e2e/labels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ExportPermissionsSpec() {
	Describe("kubeconfig export permissions", labels.Core, func() {
		It("does not grant access to unrelated resources in the destination namespace", func(ctx context.Context) {
			hostClient := cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
			Expect(hostClient).NotTo(BeNil())
			vClusterName := cluster.CurrentClusterNameFrom(ctx)
			user := "system:serviceaccount:vcluster-" + vClusterName + ":vc-" + vClusterName

			By("checking the exporter permissions in the destination namespace")
			for _, check := range []struct {
				verb, resource, name string
				allowed              bool
			}{
				{verb: "get", resource: "secrets", name: CrossNSSecretName, allowed: true},
				{verb: "patch", resource: "secrets", name: CrossNSSecretName, allowed: true},
				{verb: "list", resource: "secrets", name: ServiceAccountSecretName, allowed: true},
				{verb: "watch", resource: "secrets", name: ServiceAccountSecretName, allowed: true},
				{verb: "get", resource: "secrets", name: "unrelated"},
				{verb: "patch", resource: "secrets", name: "unrelated"},
				{verb: "list", resource: "secrets", name: "unrelated"},
				{verb: "watch", resource: "secrets", name: "unrelated"},
				{verb: "create", resource: "secrets", name: CrossNSSecretName},
				{verb: "list", resource: "secrets"},
				{verb: "watch", resource: "secrets"},
				{verb: "list", resource: "pods"},
				{verb: "watch", resource: "services"},
			} {
				Eventually(func(g Gomega) {
					review, err := hostClient.AuthorizationV1().SubjectAccessReviews().Create(ctx, &authorizationv1.SubjectAccessReview{
						Spec: authorizationv1.SubjectAccessReviewSpec{
							User:   user,
							Groups: []string{"system:serviceaccounts", "system:serviceaccounts:vcluster-" + vClusterName, "system:authenticated"},
							ResourceAttributes: &authorizationv1.ResourceAttributes{
								Namespace: TargetNS, Verb: check.verb, Resource: check.resource, Name: check.name,
							},
						},
					}, metav1.CreateOptions{})
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(review.Status.Allowed).To(Equal(check.allowed), "%s %s/%s", check.verb, check.resource, check.name)
				}).WithPolling(constants.PollingInterval).
					WithTimeout(constants.PollingTimeoutShort).
					Should(Succeed())
			}
		})
	})
}
