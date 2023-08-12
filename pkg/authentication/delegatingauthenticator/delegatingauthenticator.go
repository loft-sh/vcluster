package delegatingauthenticator

import (
	"context"
	"errors"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	authenticationv1 "k8s.io/api/authentication/v1"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/request/bearertoken"
	"k8s.io/apiserver/pkg/authentication/user"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(client client.Client) authenticator.Request {
	cache, _ := lru.New[string, cacheEntry](256)
	return bearertoken.New(&delegatingAuthenticator{
		client: client,
		cache:  cache,
	})
}

type delegatingAuthenticator struct {
	client client.Client
	cache  *lru.Cache[string, cacheEntry]
}

type cacheEntry struct {
	response *authenticator.Response
	exp      time.Time
}

func (d *delegatingAuthenticator) AuthenticateToken(ctx context.Context, token string) (*authenticator.Response, bool, error) {
	now := time.Now()

	// check if in cache
	entry, ok := d.cache.Get(token)
	if ok && entry.exp.After(now) {
		return entry.response, true, nil
	}

	tokReview := &authenticationv1.TokenReview{
		Spec: authenticationv1.TokenReviewSpec{
			Token: token,
		},
	}
	err := d.client.Create(ctx, tokReview)
	if err != nil {
		return nil, false, err
	} else if !tokReview.Status.Authenticated {
		return nil, false, errors.New(tokReview.Status.Error)
	}

	response := &authenticator.Response{
		Audiences: tokReview.Status.Audiences,
		User: &user.DefaultInfo{
			Name:   tokReview.Status.User.Username,
			UID:    tokReview.Status.User.UID,
			Groups: tokReview.Status.User.Groups,
			Extra:  clienthelper.ConvertExtraFrom(tokReview.Status.User.Extra),
		},
	}
	d.cache.Add(token, cacheEntry{
		response: response,
		exp:      now.Add(time.Second * 5),
	})
	return response, true, nil
}
