//
//
// File generated from our OpenAPI spec
//
//

// Package taxrate provides the /tax_rates APIs
package taxrate

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /tax_rates APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Creates a new tax rate.
func New(params *stripe.TaxRateParams) (*stripe.TaxRate, error) {
	return getC().New(params)
}

// Creates a new tax rate.
func (c Client) New(params *stripe.TaxRateParams) (*stripe.TaxRate, error) {
	taxrate := &stripe.TaxRate{}
	err := c.B.Call(http.MethodPost, "/v1/tax_rates", c.Key, params, taxrate)
	return taxrate, err
}

// Retrieves a tax rate with the given ID
func Get(id string, params *stripe.TaxRateParams) (*stripe.TaxRate, error) {
	return getC().Get(id, params)
}

// Retrieves a tax rate with the given ID
func (c Client) Get(id string, params *stripe.TaxRateParams) (*stripe.TaxRate, error) {
	path := stripe.FormatURLPath("/v1/tax_rates/%s", id)
	taxrate := &stripe.TaxRate{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, taxrate)
	return taxrate, err
}

// Updates an existing tax rate.
func Update(id string, params *stripe.TaxRateParams) (*stripe.TaxRate, error) {
	return getC().Update(id, params)
}

// Updates an existing tax rate.
func (c Client) Update(id string, params *stripe.TaxRateParams) (*stripe.TaxRate, error) {
	path := stripe.FormatURLPath("/v1/tax_rates/%s", id)
	taxrate := &stripe.TaxRate{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, taxrate)
	return taxrate, err
}

// Returns a list of your tax rates. Tax rates are returned sorted by creation date, with the most recently created tax rates appearing first.
func List(params *stripe.TaxRateListParams) *Iter {
	return getC().List(params)
}

// Returns a list of your tax rates. Tax rates are returned sorted by creation date, with the most recently created tax rates appearing first.
func (c Client) List(listParams *stripe.TaxRateListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.TaxRateList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/tax_rates", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for tax rates.
type Iter struct {
	*stripe.Iter
}

// TaxRate returns the tax rate which the iterator is currently pointing to.
func (i *Iter) TaxRate() *stripe.TaxRate {
	return i.Current().(*stripe.TaxRate)
}

// TaxRateList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) TaxRateList() *stripe.TaxRateList {
	return i.List().(*stripe.TaxRateList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
