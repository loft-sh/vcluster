package delegatingauthorizer

import (
	"context"
	"strings"

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

		cache: NewCache(),
	}
}

type delegatingAuthorizer struct {
	delegatingClient client.Client

	nonResources []PathVerb
	resources    []GroupVersionResourceVerb

	cache *Cache
}

func (l *delegatingAuthorizer) Authorize(ctx context.Context, a authorizer.Attributes) (authorized authorizer.Decision, reason string, err error) {
	if !applies(a, l.resources, l.nonResources) {
		return authorizer.DecisionNoOpinion, "", nil
	}

	// check if in cache
	authorized, reason, exists := l.cache.Get(a)
	if exists {
		return authorized, reason, nil
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
		l.cache.Set(a, authorizer.DecisionAllow, "")
		return authorizer.DecisionAllow, "", nil
	}

	return authorizer.DecisionDeny, accessReview.Status.Reason, nil
}

func applies(a authorizer.Attributes, resources []GroupVersionResourceVerb, nonResources []PathVerb) bool {
	if a.IsResourceRequest() {
		for _, gv := range resources {
			if (gv.Group == "*" || gv.Group == a.GetAPIGroup()) && (gv.Version == "*" || gv.Version == a.GetAPIVersion()) && (gv.Resource == "*" || gv.Resource == a.GetResource()) && verbMatches(gv.Verb, a.GetVerb()) && (gv.SubResource == "*" || gv.SubResource == a.GetSubresource()) {
				return true
			}
		}
	} else {
		for _, p := range nonResources {
			if p.Path == a.GetPath() && verbMatches(p.Verb, a.GetVerb()) {
				return true
			} else if strings.HasSuffix(p.Path, "*") && strings.HasPrefix(a.GetPath(), strings.TrimSuffix(p.Path, "*")) && verbMatches(p.Verb, a.GetVerb()) {
				return true
			}
		}
	}

	return false
}

func verbMatches(ruleVerb, requestVerb string) bool {
	// the logic is that if the ruleVerb is a list of verbs, and the requestVerb is in the list, then it matches
	// if the ruleVerb is a single verb, and the requestVerb is the same, then it matches
	// if the ruleVerb is a single verb, and the requestVerb is not the same, then it does not match
	// if the ruleVerb is a list of verbs, and the requestVerb is not in the list, then it does not match
	// if the ruleVerb is a single verb, and the requestVerb is not the same, then it does not match
	negate := false
	if strings.HasPrefix(ruleVerb, "!") {
		negate = true
		ruleVerb = strings.TrimPrefix(ruleVerb, "!")
	}

	verbs := strings.Split(ruleVerb, ",")
	for _, verb := range verbs {
		if verb == "*" || verb == requestVerb {
			return !negate
		}
	}

	return negate
}
