// Package oauth provides the OAuth APIs
package oauth

import (
	"fmt"
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /oauth and related APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// AuthorizeURL builds an OAuth authorize URL.
func AuthorizeURL(params *stripe.AuthorizeURLParams) string {
	return getC().AuthorizeURL(params)
}

// AuthorizeURL builds an OAuth authorize URL.
func (c Client) AuthorizeURL(params *stripe.AuthorizeURLParams) string {
	express := ""
	if stripe.BoolValue(params.Express) {
		express = "/express"
	}
	qs := &form.Values{}
	form.AppendTo(qs, params)
	return fmt.Sprintf(
		"%s%s/oauth/authorize?%s",
		stripe.ConnectURL,
		express,
		qs.Encode(),
	)
}

// New creates an OAuth token using a code after successful redirection back.
func New(params *stripe.OAuthTokenParams) (*stripe.OAuthToken, error) {
	return getC().New(params)
}

// New creates an OAuth token using a code after successful redirection back.
func (c Client) New(params *stripe.OAuthTokenParams) (*stripe.OAuthToken, error) {
	// client_secret is sent in the post body for this endpoint.
	if stripe.StringValue(params.ClientSecret) == "" {
		params.ClientSecret = stripe.String(stripe.Key)
	}

	oauthToken := &stripe.OAuthToken{}
	err := c.B.Call(http.MethodPost, "/oauth/token", c.Key, params, oauthToken)

	return oauthToken, err
}

// Del deauthorizes a connected account.
func Del(params *stripe.DeauthorizeParams) (*stripe.Deauthorize, error) {
	return getC().Del(params)
}

// Del deauthorizes a connected account.
func (c Client) Del(params *stripe.DeauthorizeParams) (*stripe.Deauthorize, error) {
	deauthorization := &stripe.Deauthorize{}
	err := c.B.Call(
		http.MethodPost,
		"/oauth/deauthorize",
		c.Key,
		params,
		deauthorization,
	)
	return deauthorization, err
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.ConnectBackend), stripe.Key}
}
