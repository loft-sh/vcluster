//
//
// File generated from our OpenAPI spec
//
//

// Package creditgrant provides the /billing/credit_grants APIs
package creditgrant

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /billing/credit_grants APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Creates a credit grant.
func New(params *stripe.BillingCreditGrantParams) (*stripe.BillingCreditGrant, error) {
	return getC().New(params)
}

// Creates a credit grant.
func (c Client) New(params *stripe.BillingCreditGrantParams) (*stripe.BillingCreditGrant, error) {
	creditgrant := &stripe.BillingCreditGrant{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/billing/credit_grants",
		c.Key,
		params,
		creditgrant,
	)
	return creditgrant, err
}

// Retrieves a credit grant.
func Get(id string, params *stripe.BillingCreditGrantParams) (*stripe.BillingCreditGrant, error) {
	return getC().Get(id, params)
}

// Retrieves a credit grant.
func (c Client) Get(id string, params *stripe.BillingCreditGrantParams) (*stripe.BillingCreditGrant, error) {
	path := stripe.FormatURLPath("/v1/billing/credit_grants/%s", id)
	creditgrant := &stripe.BillingCreditGrant{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, creditgrant)
	return creditgrant, err
}

// Updates a credit grant.
func Update(id string, params *stripe.BillingCreditGrantParams) (*stripe.BillingCreditGrant, error) {
	return getC().Update(id, params)
}

// Updates a credit grant.
func (c Client) Update(id string, params *stripe.BillingCreditGrantParams) (*stripe.BillingCreditGrant, error) {
	path := stripe.FormatURLPath("/v1/billing/credit_grants/%s", id)
	creditgrant := &stripe.BillingCreditGrant{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, creditgrant)
	return creditgrant, err
}

// Expires a credit grant.
func Expire(id string, params *stripe.BillingCreditGrantExpireParams) (*stripe.BillingCreditGrant, error) {
	return getC().Expire(id, params)
}

// Expires a credit grant.
func (c Client) Expire(id string, params *stripe.BillingCreditGrantExpireParams) (*stripe.BillingCreditGrant, error) {
	path := stripe.FormatURLPath("/v1/billing/credit_grants/%s/expire", id)
	creditgrant := &stripe.BillingCreditGrant{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, creditgrant)
	return creditgrant, err
}

// Voids a credit grant.
func VoidGrant(id string, params *stripe.BillingCreditGrantVoidGrantParams) (*stripe.BillingCreditGrant, error) {
	return getC().VoidGrant(id, params)
}

// Voids a credit grant.
func (c Client) VoidGrant(id string, params *stripe.BillingCreditGrantVoidGrantParams) (*stripe.BillingCreditGrant, error) {
	path := stripe.FormatURLPath("/v1/billing/credit_grants/%s/void", id)
	creditgrant := &stripe.BillingCreditGrant{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, creditgrant)
	return creditgrant, err
}

// Retrieve a list of credit grants.
func List(params *stripe.BillingCreditGrantListParams) *Iter {
	return getC().List(params)
}

// Retrieve a list of credit grants.
func (c Client) List(listParams *stripe.BillingCreditGrantListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.BillingCreditGrantList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/billing/credit_grants", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for billing credit grants.
type Iter struct {
	*stripe.Iter
}

// BillingCreditGrant returns the billing credit grant which the iterator is currently pointing to.
func (i *Iter) BillingCreditGrant() *stripe.BillingCreditGrant {
	return i.Current().(*stripe.BillingCreditGrant)
}

// BillingCreditGrantList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) BillingCreditGrantList() *stripe.BillingCreditGrantList {
	return i.List().(*stripe.BillingCreditGrantList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
