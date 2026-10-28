package export_kubeconfig

import (
	"context"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e/constants"
	"github.com/loft-sh/vcluster/e2e/labels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	authorizationv1 "k8s.io/api/authorization/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func ExportPermissionsSpec() {
	Describe("kubeconfig export permissions", labels.Core, func() {
		It("does not grant access to unrelated resources in the destination namespace", func(ctx context.Context) {
			hostClient := cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
			Expect(hostClient).NotTo(BeNil())
			vClusterName := cluster.CurrentClusterNameFrom(ctx)
			user := "system:serviceaccount:vcluster-" + vClusterName + ":vc-" + vClusterName
			groups := []string{"system:serviceaccounts", "system:serviceaccounts:vcluster-" + vClusterName, "system:authenticated"}

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
							Groups: groups,
							ResourceAttributes: &authorizationv1.ResourceAttributes{
								Namespace: TargetNS, Verb: check.verb, Resource: check.resource, Name: check.name,
							},
						},
					}, metav1.CreateOptions{})
					g.Expect(err).NotTo(HaveOccurred(), "review exporter permission to %s %s/%s in namespace %s", check.verb, check.resource, check.name, TargetNS)
					g.Expect(review.Status.Allowed).To(Equal(check.allowed), "%s %s/%s", check.verb, check.resource, check.name)
				}).WithContext(ctx).WithPolling(constants.PollingInterval).
					WithTimeout(constants.PollingTimeoutShort).
					Should(Succeed())
			}

			By("listing and watching the destination Secret with a name field selector")
			hostConfig := rest.CopyConfig(cluster.From(ctx, constants.GetHostClusterName()).KubernetesRestConfig())
			hostConfig.Impersonate = rest.ImpersonationConfig{UserName: user, Groups: groups}
			hostConfig.Timeout = constants.PollingTimeoutShort
			exporter, err := kubernetes.NewForConfig(hostConfig)
			Expect(err).NotTo(HaveOccurred(), "create host client impersonating exporter %s", user)
			secrets := exporter.CoreV1().Secrets(TargetNS)
			options := metav1.ListOptions{FieldSelector: fields.OneTermEqualSelector("metadata.name", ServiceAccountSecretName).String()}
			Eventually(func(g Gomega) {
				listed, err := secrets.List(ctx, options)
				g.Expect(err).NotTo(HaveOccurred(), "list destination Secret %s/%s with a name field selector", TargetNS, ServiceAccountSecretName)
				names := make([]string, 0, len(listed.Items))
				for _, secret := range listed.Items {
					names = append(names, secret.Name)
				}
				g.Expect(names).To(ConsistOf(ServiceAccountSecretName), "name-filtered list in namespace %s must return only the destination Secret", TargetNS)
				stream, err := secrets.Watch(ctx, options)
				g.Expect(err).NotTo(HaveOccurred(), "watch destination Secret %s/%s with a name field selector", TargetNS, ServiceAccountSecretName)
				stream.Stop()
			}).WithContext(ctx).WithPolling(constants.PollingInterval).
				WithTimeout(constants.PollingTimeoutShort).
				Should(Succeed())

			By("rejecting unfiltered requests and selectors for an unrelated Secret")
			for _, selector := range []string{"", fields.OneTermEqualSelector("metadata.name", "unrelated").String()} {
				_, err := secrets.List(ctx, metav1.ListOptions{FieldSelector: selector})
				Expect(kerrors.IsForbidden(err)).To(BeTrue(), "list Secrets in namespace %s with selector %q must be forbidden, got %v", TargetNS, selector, err)
				stream, err := secrets.Watch(ctx, metav1.ListOptions{FieldSelector: selector})
				if stream != nil {
					stream.Stop()
				}
				Expect(kerrors.IsForbidden(err)).To(BeTrue(), "watch Secrets in namespace %s with selector %q must be forbidden, got %v", TargetNS, selector, err)
			}
		})
	})
}
