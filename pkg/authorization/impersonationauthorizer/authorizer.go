package impersonationauthorizer

import (
	"context"

	delegatingauthorizer "github.com/loft-sh/vcluster/pkg/authorization/delegatingauthorizer"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(client client.Client) authorizer.Authorizer {
	return &impersonationAuthorizer{
		client: client,

		cache: delegatingauthorizer.NewCache(),
	}
}

type impersonationAuthorizer struct {
	client client.Client

	cache *delegatingauthorizer.Cache
}

func (i *impersonationAuthorizer) Authorize(ctx context.Context, a authorizer.Attributes) (authorized authorizer.Decision, reason string, err error) {
	if a.GetVerb() != "impersonate" || !a.IsResourceRequest() {
		return authorizer.DecisionNoOpinion, "", nil
	}

	// check if in cache
	authorized, reason, exists := i.cache.Get(a)
	if exists {
		return authorized, reason, nil
	}

	// check if request is allowed in the target cluster
	accessReview := &authorizationv1.SubjectAccessReview{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: authorizationv1.SubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace:   a.GetNamespace(),
				Verb:        a.GetVerb(),
				Group:       a.GetAPIGroup(),
				Version:     a.GetAPIVersion(),
				Resource:    a.GetResource(),
				Subresource: a.GetSubresource(),
				Name:        a.GetName(),
			},
			User:   a.GetUser().GetName(),
			UID:    a.GetUser().GetUID(),
			Groups: a.GetUser().GetGroups(),
			Extra:  clienthelper.ConvertExtra(a.GetUser().GetExtra()),
		},
	}
	err = i.client.Create(ctx, accessReview)
	if err != nil {
		return authorizer.DecisionDeny, "", err
	} else if accessReview.Status.Allowed && !accessReview.Status.Denied {
		i.cache.Set(a, authorizer.DecisionAllow, "")
		return authorizer.DecisionAllow, "", nil
	}

	return authorizer.DecisionDeny, accessReview.Status.Reason, nil
}
