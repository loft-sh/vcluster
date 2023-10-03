package kubeletauthorizer

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/server/filters"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PathVerb struct {
	Path string
	Verb string
}

func New(uncachedVirtualClient client.Client) authorizer.Authorizer {
	return &kubeletAuthorizer{
		uncachedVirtualClient: uncachedVirtualClient,
	}
}

type kubeletAuthorizer struct {
	uncachedVirtualClient client.Client
}

func (l *kubeletAuthorizer) Authorize(ctx context.Context, a authorizer.Attributes) (authorized authorizer.Decision, reason string, err error) { // get node name
	nodeName, ok := filters.NodeNameFrom(ctx)
	if !ok {
		return authorizer.DecisionNoOpinion, "", nil
	} else if a.IsResourceRequest() {
		return authorizer.DecisionDeny, "forbidden", nil
	}

	// check if request is allowed in the target cluster
	accessReview := &authorizationv1.SubjectAccessReview{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: authorizationv1.SubjectAccessReviewSpec{
			User:   a.GetUser().GetName(),
			UID:    a.GetUser().GetUID(),
			Groups: a.GetUser().GetGroups(),
			Extra:  clienthelper.ConvertExtra(a.GetUser().GetExtra()),
		},
	}

	// check what kind of request it is
	if filters.IsKubeletStats(a.GetPath()) {
		accessReview.Spec.ResourceAttributes = &authorizationv1.ResourceAttributes{
			Verb:        "get",
			Group:       corev1.SchemeGroupVersion.Group,
			Version:     corev1.SchemeGroupVersion.Version,
			Resource:    "nodes",
			Subresource: "stats",
			Name:        nodeName,
		}
	} else if filters.IsKubeletMetrics(a.GetPath()) {
		accessReview.Spec.ResourceAttributes = &authorizationv1.ResourceAttributes{
			Verb:        "get",
			Group:       corev1.SchemeGroupVersion.Group,
			Version:     corev1.SchemeGroupVersion.Version,
			Resource:    "nodes",
			Subresource: "metrics",
			Name:        nodeName,
		}
	} else {
		accessReview.Spec.NonResourceAttributes = &authorizationv1.NonResourceAttributes{
			Path: a.GetPath(),
			Verb: a.GetVerb(),
		}
	}

	err = l.uncachedVirtualClient.Create(ctx, accessReview)
	if err != nil {
		return authorizer.DecisionDeny, "", err
	} else if accessReview.Status.Allowed && !accessReview.Status.Denied {
		return authorizer.DecisionAllow, "", nil
	}

	return authorizer.DecisionDeny, accessReview.Status.Reason, nil
}
