package delegatingauthorizer

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GroupVersionResourceVerb struct {
	schema.GroupVersionResource
	SubResource string
	Verb        string
}

type PathVerb struct {
	Path string
	Verb string
}

func New(delegatingClient client.Client, resources []GroupVersionResourceVerb, nonResources []PathVerb) authorizer.Authorizer {
	return &delegatingAuthorizer{
		delegatingClient: delegatingClient,

		nonResources: nonResources,
		resources:    resources,
	}
}

type delegatingAuthorizer struct {
	delegatingClient client.Client

	nonResources []PathVerb
	resources    []GroupVersionResourceVerb
}

func (l *delegatingAuthorizer) Authorize(ctx context.Context, a authorizer.Attributes) (authorized authorizer.Decision, reason string, err error) {
	if !applies(a, l.resources, l.nonResources) {
		return authorizer.DecisionNoOpinion, "", nil
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
	if a.IsResourceRequest() {
		accessReview.Spec.ResourceAttributes = &authorizationv1.ResourceAttributes{
			Namespace:   a.GetNamespace(),
			Verb:        a.GetVerb(),
			Group:       a.GetAPIGroup(),
			Version:     a.GetAPIVersion(),
			Resource:    a.GetResource(),
			Subresource: a.GetSubresource(),
			Name:        a.GetName(),
		}
	} else {
		accessReview.Spec.NonResourceAttributes = &authorizationv1.NonResourceAttributes{
			Path: a.GetPath(),
			Verb: a.GetVerb(),
		}
	}
	err = l.delegatingClient.Create(ctx, accessReview)
	if err != nil {
		return authorizer.DecisionDeny, "", err
	} else if accessReview.Status.Allowed && !accessReview.Status.Denied {
		return authorizer.DecisionAllow, "", nil
	}

	return authorizer.DecisionDeny, accessReview.Status.Reason, nil
}

func applies(a authorizer.Attributes, resources []GroupVersionResourceVerb, nonResources []PathVerb) bool {
	if a.IsResourceRequest() {
		for _, gv := range resources {
			if (gv.Group == "*" || gv.Group == a.GetAPIGroup()) && (gv.Version == "*" || gv.Version == a.GetAPIVersion()) && (gv.Resource == "*" || gv.Resource == a.GetResource()) && (gv.Verb == "*" || gv.Verb == a.GetVerb()) && (gv.SubResource == "*" || gv.SubResource == a.GetSubresource()) {
				return true
			}
		}
	} else {
		for _, p := range nonResources {
			if p.Path == a.GetPath() && (p.Verb == "*" || p.Verb == a.GetVerb()) {
				return true
			}
		}
	}

	return false
}
