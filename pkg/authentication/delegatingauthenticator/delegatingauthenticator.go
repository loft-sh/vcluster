package delegatingauthenticator

import (
	"context"
	"errors"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	authenticationv1 "k8s.io/api/authentication/v1"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/request/bearertoken"
	"k8s.io/apiserver/pkg/authentication/user"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(client client.Client) authenticator.Request {
	return bearertoken.New(&delegatingAuthenticator{client: client})
}

type delegatingAuthenticator struct {
	client client.Client
}

func (d *delegatingAuthenticator) AuthenticateToken(ctx context.Context, token string) (*authenticator.Response, bool, error) {
	tokReview := &authenticationv1.TokenReview{
		Spec: authenticationv1.TokenReviewSpec{
			Token: token,
		},
	}
	err := d.client.Create(ctx, tokReview)
	if err != nil {
		return nil, false, err
	} else if tokReview.Status.Authenticated == false {
		return nil, false, errors.New(tokReview.Status.Error)
	}

	return &authenticator.Response{
		Audiences: tokReview.Status.Audiences,
		User: &user.DefaultInfo{
			Name:   tokReview.Status.User.Username,
			UID:    tokReview.Status.User.UID,
			Groups: tokReview.Status.User.Groups,
			Extra:  clienthelper.ConvertExtraFrom(tokReview.Status.User.Extra),
		},
	}, true, nil
}
