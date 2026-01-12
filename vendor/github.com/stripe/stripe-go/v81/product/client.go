//
//
// File generated from our OpenAPI spec
//
//

// Package product provides the /products APIs
package product

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /products APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Creates a new product object.
func New(params *stripe.ProductParams) (*stripe.Product, error) {
	return getC().New(params)
}

// Creates a new product object.
func (c Client) New(params *stripe.ProductParams) (*stripe.Product, error) {
	product := &stripe.Product{}
	err := c.B.Call(http.MethodPost, "/v1/products", c.Key, params, product)
	return product, err
}

// Retrieves the details of an existing product. Supply the unique product ID from either a product creation request or the product list, and Stripe will return the corresponding product information.
func Get(id string, params *stripe.ProductParams) (*stripe.Product, error) {
	return getC().Get(id, params)
}

// Retrieves the details of an existing product. Supply the unique product ID from either a product creation request or the product list, and Stripe will return the corresponding product information.
func (c Client) Get(id string, params *stripe.ProductParams) (*stripe.Product, error) {
	path := stripe.FormatURLPath("/v1/products/%s", id)
	product := &stripe.Product{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, product)
	return product, err
}

// Updates the specific product by setting the values of the parameters passed. Any parameters not provided will be left unchanged.
func Update(id string, params *stripe.ProductParams) (*stripe.Product, error) {
	return getC().Update(id, params)
}

// Updates the specific product by setting the values of the parameters passed. Any parameters not provided will be left unchanged.
func (c Client) Update(id string, params *stripe.ProductParams) (*stripe.Product, error) {
	path := stripe.FormatURLPath("/v1/products/%s", id)
	product := &stripe.Product{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, product)
	return product, err
}

// Delete a product. Deleting a product is only possible if it has no prices associated with it. Additionally, deleting a product with type=good is only possible if it has no SKUs associated with it.
func Del(id string, params *stripe.ProductParams) (*stripe.Product, error) {
	return getC().Del(id, params)
}

// Delete a product. Deleting a product is only possible if it has no prices associated with it. Additionally, deleting a product with type=good is only possible if it has no SKUs associated with it.
func (c Client) Del(id string, params *stripe.ProductParams) (*stripe.Product, error) {
	path := stripe.FormatURLPath("/v1/products/%s", id)
	product := &stripe.Product{}
	err := c.B.Call(http.MethodDelete, path, c.Key, params, product)
	return product, err
}

// Returns a list of your products. The products are returned sorted by creation date, with the most recently created products appearing first.
func List(params *stripe.ProductListParams) *Iter {
	return getC().List(params)
}

// Returns a list of your products. The products are returned sorted by creation date, with the most recently created products appearing first.
func (c Client) List(listParams *stripe.ProductListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.ProductList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/products", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for products.
type Iter struct {
	*stripe.Iter
}

// Product returns the product which the iterator is currently pointing to.
func (i *Iter) Product() *stripe.Product {
	return i.Current().(*stripe.Product)
}

// ProductList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) ProductList() *stripe.ProductList {
	return i.List().(*stripe.ProductList)
}

// Search for products you've previously created using Stripe's [Search Query Language](https://stripe.com/docs/search#search-query-language).
// Don't use search in read-after-write flows where strict consistency is necessary. Under normal operating
// conditions, data is searchable in less than a minute. Occasionally, propagation of new or updated data can be up
// to an hour behind during outages. Search functionality is not available to merchants in India.
func Search(params *stripe.ProductSearchParams) *SearchIter {
	return getC().Search(params)
}

// Search for products you've previously created using Stripe's [Search Query Language](https://stripe.com/docs/search#search-query-language).
// Don't use search in read-after-write flows where strict consistency is necessary. Under normal operating
// conditions, data is searchable in less than a minute. Occasionally, propagation of new or updated data can be up
// to an hour behind during outages. Search functionality is not available to merchants in India.
func (c Client) Search(params *stripe.ProductSearchParams) *SearchIter {
	return &SearchIter{
		SearchIter: stripe.GetSearchIter(params, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.SearchContainer, error) {
			list := &stripe.ProductSearchResult{}
			err := c.B.CallRaw(http.MethodGet, "/v1/products/search", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// SearchIter is an iterator for products.
type SearchIter struct {
	*stripe.SearchIter
}

// Product returns the product which the iterator is currently pointing to.
func (i *SearchIter) Product() *stripe.Product {
	return i.Current().(*stripe.Product)
}

// ProductSearchResult returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *SearchIter) ProductSearchResult() *stripe.ProductSearchResult {
	return i.SearchResult().(*stripe.ProductSearchResult)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
