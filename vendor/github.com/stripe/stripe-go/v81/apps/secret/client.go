//
//
// File generated from our OpenAPI spec
//
//

// Package secret provides the /apps/secrets APIs
package secret

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /apps/secrets APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Create or replace a secret in the secret store.
func New(params *stripe.AppsSecretParams) (*stripe.AppsSecret, error) {
	return getC().New(params)
}

// Create or replace a secret in the secret store.
func (c Client) New(params *stripe.AppsSecretParams) (*stripe.AppsSecret, error) {
	secret := &stripe.AppsSecret{}
	err := c.B.Call(http.MethodPost, "/v1/apps/secrets", c.Key, params, secret)
	return secret, err
}

// Deletes a secret from the secret store by name and scope.
func DeleteWhere(params *stripe.AppsSecretDeleteWhereParams) (*stripe.AppsSecret, error) {
	return getC().DeleteWhere(params)
}

// Deletes a secret from the secret store by name and scope.
func (c Client) DeleteWhere(params *stripe.AppsSecretDeleteWhereParams) (*stripe.AppsSecret, error) {
	secret := &stripe.AppsSecret{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/apps/secrets/delete",
		c.Key,
		params,
		secret,
	)
	return secret, err
}

// Finds a secret in the secret store by name and scope.
func Find(params *stripe.AppsSecretFindParams) (*stripe.AppsSecret, error) {
	return getC().Find(params)
}

// Finds a secret in the secret store by name and scope.
func (c Client) Find(params *stripe.AppsSecretFindParams) (*stripe.AppsSecret, error) {
	secret := &stripe.AppsSecret{}
	err := c.B.Call(
		http.MethodGet,
		"/v1/apps/secrets/find",
		c.Key,
		params,
		secret,
	)
	return secret, err
}

// List all secrets stored on the given scope.
func List(params *stripe.AppsSecretListParams) *Iter {
	return getC().List(params)
}

// List all secrets stored on the given scope.
func (c Client) List(listParams *stripe.AppsSecretListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.AppsSecretList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/apps/secrets", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for apps secrets.
type Iter struct {
	*stripe.Iter
}

// AppsSecret returns the apps secret which the iterator is currently pointing to.
func (i *Iter) AppsSecret() *stripe.AppsSecret {
	return i.Current().(*stripe.AppsSecret)
}

// AppsSecretList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) AppsSecretList() *stripe.AppsSecretList {
	return i.List().(*stripe.AppsSecretList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
