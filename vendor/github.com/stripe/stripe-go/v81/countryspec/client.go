//
//
// File generated from our OpenAPI spec
//
//

// Package countryspec provides the /country_specs APIs
package countryspec

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /country_specs APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Returns a Country Spec for a given Country code.
func Get(id string, params *stripe.CountrySpecParams) (*stripe.CountrySpec, error) {
	return getC().Get(id, params)
}

// Returns a Country Spec for a given Country code.
func (c Client) Get(id string, params *stripe.CountrySpecParams) (*stripe.CountrySpec, error) {
	path := stripe.FormatURLPath("/v1/country_specs/%s", id)
	countryspec := &stripe.CountrySpec{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, countryspec)
	return countryspec, err
}

// Lists all Country Spec objects available in the API.
func List(params *stripe.CountrySpecListParams) *Iter {
	return getC().List(params)
}

// Lists all Country Spec objects available in the API.
func (c Client) List(listParams *stripe.CountrySpecListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.CountrySpecList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/country_specs", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for country specs.
type Iter struct {
	*stripe.Iter
}

// CountrySpec returns the country spec which the iterator is currently pointing to.
func (i *Iter) CountrySpec() *stripe.CountrySpec {
	return i.Current().(*stripe.CountrySpec)
}

// CountrySpecList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) CountrySpecList() *stripe.CountrySpecList {
	return i.List().(*stripe.CountrySpecList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
