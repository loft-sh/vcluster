//
//
// File generated from our OpenAPI spec
//
//

// Package configuration provides the /billing_portal/configurations APIs
package configuration

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /billing_portal/configurations APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Creates a configuration that describes the functionality and behavior of a PortalSession
func New(params *stripe.BillingPortalConfigurationParams) (*stripe.BillingPortalConfiguration, error) {
	return getC().New(params)
}

// Creates a configuration that describes the functionality and behavior of a PortalSession
func (c Client) New(params *stripe.BillingPortalConfigurationParams) (*stripe.BillingPortalConfiguration, error) {
	configuration := &stripe.BillingPortalConfiguration{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/billing_portal/configurations",
		c.Key,
		params,
		configuration,
	)
	return configuration, err
}

// Retrieves a configuration that describes the functionality of the customer portal.
func Get(id string, params *stripe.BillingPortalConfigurationParams) (*stripe.BillingPortalConfiguration, error) {
	return getC().Get(id, params)
}

// Retrieves a configuration that describes the functionality of the customer portal.
func (c Client) Get(id string, params *stripe.BillingPortalConfigurationParams) (*stripe.BillingPortalConfiguration, error) {
	path := stripe.FormatURLPath("/v1/billing_portal/configurations/%s", id)
	configuration := &stripe.BillingPortalConfiguration{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, configuration)
	return configuration, err
}

// Updates a configuration that describes the functionality of the customer portal.
func Update(id string, params *stripe.BillingPortalConfigurationParams) (*stripe.BillingPortalConfiguration, error) {
	return getC().Update(id, params)
}

// Updates a configuration that describes the functionality of the customer portal.
func (c Client) Update(id string, params *stripe.BillingPortalConfigurationParams) (*stripe.BillingPortalConfiguration, error) {
	path := stripe.FormatURLPath("/v1/billing_portal/configurations/%s", id)
	configuration := &stripe.BillingPortalConfiguration{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, configuration)
	return configuration, err
}

// Returns a list of configurations that describe the functionality of the customer portal.
func List(params *stripe.BillingPortalConfigurationListParams) *Iter {
	return getC().List(params)
}

// Returns a list of configurations that describe the functionality of the customer portal.
func (c Client) List(listParams *stripe.BillingPortalConfigurationListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.BillingPortalConfigurationList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/billing_portal/configurations", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for billing portal configurations.
type Iter struct {
	*stripe.Iter
}

// BillingPortalConfiguration returns the billing portal configuration which the iterator is currently pointing to.
func (i *Iter) BillingPortalConfiguration() *stripe.BillingPortalConfiguration {
	return i.Current().(*stripe.BillingPortalConfiguration)
}

// BillingPortalConfigurationList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) BillingPortalConfigurationList() *stripe.BillingPortalConfigurationList {
	return i.List().(*stripe.BillingPortalConfigurationList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
