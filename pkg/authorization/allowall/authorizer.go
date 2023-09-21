package allowall

import (
	"context"

	"k8s.io/apiserver/pkg/authorization/authorizer"
)

func New() authorizer.Authorizer {
	return &allowAllAuthorizer{}
}

type allowAllAuthorizer struct{}

func (i *allowAllAuthorizer) Authorize(context.Context, authorizer.Attributes) (authorized authorizer.Decision, reason string, err error) {
	return authorizer.DecisionAllow, "", nil
}
