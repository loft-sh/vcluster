//
//
// File generated from our OpenAPI spec
//
//

// Package price provides the /prices APIs
package price

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /prices APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Creates a new price for an existing product. The price can be recurring or one-time.
func New(params *stripe.PriceParams) (*stripe.Price, error) {
	return getC().New(params)
}

// Creates a new price for an existing product. The price can be recurring or one-time.
func (c Client) New(params *stripe.PriceParams) (*stripe.Price, error) {
	price := &stripe.Price{}
	err := c.B.Call(http.MethodPost, "/v1/prices", c.Key, params, price)
	return price, err
}

// Retrieves the price with the given ID.
func Get(id string, params *stripe.PriceParams) (*stripe.Price, error) {
	return getC().Get(id, params)
}

// Retrieves the price with the given ID.
func (c Client) Get(id string, params *stripe.PriceParams) (*stripe.Price, error) {
	path := stripe.FormatURLPath("/v1/prices/%s", id)
	price := &stripe.Price{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, price)
	return price, err
}

// Updates the specified price by setting the values of the parameters passed. Any parameters not provided are left unchanged.
func Update(id string, params *stripe.PriceParams) (*stripe.Price, error) {
	return getC().Update(id, params)
}

// Updates the specified price by setting the values of the parameters passed. Any parameters not provided are left unchanged.
func (c Client) Update(id string, params *stripe.PriceParams) (*stripe.Price, error) {
	path := stripe.FormatURLPath("/v1/prices/%s", id)
	price := &stripe.Price{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, price)
	return price, err
}

// Returns a list of your active prices, excluding [inline prices](https://stripe.com/docs/products-prices/pricing-models#inline-pricing). For the list of inactive prices, set active to false.
func List(params *stripe.PriceListParams) *Iter {
	return getC().List(params)
}

// Returns a list of your active prices, excluding [inline prices](https://stripe.com/docs/products-prices/pricing-models#inline-pricing). For the list of inactive prices, set active to false.
func (c Client) List(listParams *stripe.PriceListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.PriceList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/prices", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for prices.
type Iter struct {
	*stripe.Iter
}

// Price returns the price which the iterator is currently pointing to.
func (i *Iter) Price() *stripe.Price {
	return i.Current().(*stripe.Price)
}

// PriceList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) PriceList() *stripe.PriceList {
	return i.List().(*stripe.PriceList)
}

// Search for prices you've previously created using Stripe's [Search Query Language](https://stripe.com/docs/search#search-query-language).
// Don't use search in read-after-write flows where strict consistency is necessary. Under normal operating
// conditions, data is searchable in less than a minute. Occasionally, propagation of new or updated data can be up
// to an hour behind during outages. Search functionality is not available to merchants in India.
func Search(params *stripe.PriceSearchParams) *SearchIter {
	return getC().Search(params)
}

// Search for prices you've previously created using Stripe's [Search Query Language](https://stripe.com/docs/search#search-query-language).
// Don't use search in read-after-write flows where strict consistency is necessary. Under normal operating
// conditions, data is searchable in less than a minute. Occasionally, propagation of new or updated data can be up
// to an hour behind during outages. Search functionality is not available to merchants in India.
func (c Client) Search(params *stripe.PriceSearchParams) *SearchIter {
	return &SearchIter{
		SearchIter: stripe.GetSearchIter(params, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.SearchContainer, error) {
			list := &stripe.PriceSearchResult{}
			err := c.B.CallRaw(http.MethodGet, "/v1/prices/search", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// SearchIter is an iterator for prices.
type SearchIter struct {
	*stripe.SearchIter
}

// Price returns the price which the iterator is currently pointing to.
func (i *SearchIter) Price() *stripe.Price {
	return i.Current().(*stripe.Price)
}

// PriceSearchResult returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *SearchIter) PriceSearchResult() *stripe.PriceSearchResult {
	return i.SearchResult().(*stripe.PriceSearchResult)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
