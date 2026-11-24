package fromhost

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// vClusterCanList asks the host API server whether the vCluster service account
// may list the given resource in a host namespace. Use metav1.NamespaceAll to
// ask about all namespaces.
func vClusterCanList(ctx context.Context, hostClient kubernetes.Interface, vClusterHostNS, vClusterName, resource, namespace string) bool {
	GinkgoHelper()
	review, err := hostClient.AuthorizationV1().SubjectAccessReviews().Create(ctx, &authorizationv1.SubjectAccessReview{
		Spec: authorizationv1.SubjectAccessReviewSpec{
			User:   "system:serviceaccount:" + vClusterHostNS + ":vc-" + vClusterName,
			Groups: []string{"system:serviceaccounts", "system:serviceaccounts:" + vClusterHostNS, "system:authenticated"},
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace: namespace,
				Verb:      "list",
				Resource:  resource,
			},
		},
	}, metav1.CreateOptions{})
	Expect(err).To(Succeed())
	return review.Status.Allowed
}
