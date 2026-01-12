//
//
// File generated from our OpenAPI spec
//
//

// Package shippingrate provides the /shipping_rates APIs
package shippingrate

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /shipping_rates APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Creates a new shipping rate object.
func New(params *stripe.ShippingRateParams) (*stripe.ShippingRate, error) {
	return getC().New(params)
}

// Creates a new shipping rate object.
func (c Client) New(params *stripe.ShippingRateParams) (*stripe.ShippingRate, error) {
	shippingrate := &stripe.ShippingRate{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/shipping_rates",
		c.Key,
		params,
		shippingrate,
	)
	return shippingrate, err
}

// Returns the shipping rate object with the given ID.
func Get(id string, params *stripe.ShippingRateParams) (*stripe.ShippingRate, error) {
	return getC().Get(id, params)
}

// Returns the shipping rate object with the given ID.
func (c Client) Get(id string, params *stripe.ShippingRateParams) (*stripe.ShippingRate, error) {
	path := stripe.FormatURLPath("/v1/shipping_rates/%s", id)
	shippingrate := &stripe.ShippingRate{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, shippingrate)
	return shippingrate, err
}

// Updates an existing shipping rate object.
func Update(id string, params *stripe.ShippingRateParams) (*stripe.ShippingRate, error) {
	return getC().Update(id, params)
}

// Updates an existing shipping rate object.
func (c Client) Update(id string, params *stripe.ShippingRateParams) (*stripe.ShippingRate, error) {
	path := stripe.FormatURLPath("/v1/shipping_rates/%s", id)
	shippingrate := &stripe.ShippingRate{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, shippingrate)
	return shippingrate, err
}

// Returns a list of your shipping rates.
func List(params *stripe.ShippingRateListParams) *Iter {
	return getC().List(params)
}

// Returns a list of your shipping rates.
func (c Client) List(listParams *stripe.ShippingRateListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.ShippingRateList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/shipping_rates", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for shipping rates.
type Iter struct {
	*stripe.Iter
}

// ShippingRate returns the shipping rate which the iterator is currently pointing to.
func (i *Iter) ShippingRate() *stripe.ShippingRate {
	return i.Current().(*stripe.ShippingRate)
}

// ShippingRateList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) ShippingRateList() *stripe.ShippingRateList {
	return i.List().(*stripe.ShippingRateList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
