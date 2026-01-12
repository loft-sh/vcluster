//
//
// File generated from our OpenAPI spec
//
//

// Package paymentmethodconfiguration provides the /payment_method_configurations APIs
package paymentmethodconfiguration

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /payment_method_configurations APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Creates a payment method configuration
func New(params *stripe.PaymentMethodConfigurationParams) (*stripe.PaymentMethodConfiguration, error) {
	return getC().New(params)
}

// Creates a payment method configuration
func (c Client) New(params *stripe.PaymentMethodConfigurationParams) (*stripe.PaymentMethodConfiguration, error) {
	paymentmethodconfiguration := &stripe.PaymentMethodConfiguration{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/payment_method_configurations",
		c.Key,
		params,
		paymentmethodconfiguration,
	)
	return paymentmethodconfiguration, err
}

// Retrieve payment method configuration
func Get(id string, params *stripe.PaymentMethodConfigurationParams) (*stripe.PaymentMethodConfiguration, error) {
	return getC().Get(id, params)
}

// Retrieve payment method configuration
func (c Client) Get(id string, params *stripe.PaymentMethodConfigurationParams) (*stripe.PaymentMethodConfiguration, error) {
	path := stripe.FormatURLPath("/v1/payment_method_configurations/%s", id)
	paymentmethodconfiguration := &stripe.PaymentMethodConfiguration{}
	err := c.B.Call(
		http.MethodGet,
		path,
		c.Key,
		params,
		paymentmethodconfiguration,
	)
	return paymentmethodconfiguration, err
}

// Update payment method configuration
func Update(id string, params *stripe.PaymentMethodConfigurationParams) (*stripe.PaymentMethodConfiguration, error) {
	return getC().Update(id, params)
}

// Update payment method configuration
func (c Client) Update(id string, params *stripe.PaymentMethodConfigurationParams) (*stripe.PaymentMethodConfiguration, error) {
	path := stripe.FormatURLPath("/v1/payment_method_configurations/%s", id)
	paymentmethodconfiguration := &stripe.PaymentMethodConfiguration{}
	err := c.B.Call(
		http.MethodPost,
		path,
		c.Key,
		params,
		paymentmethodconfiguration,
	)
	return paymentmethodconfiguration, err
}

// List payment method configurations
func List(params *stripe.PaymentMethodConfigurationListParams) *Iter {
	return getC().List(params)
}

// List payment method configurations
func (c Client) List(listParams *stripe.PaymentMethodConfigurationListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.PaymentMethodConfigurationList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/payment_method_configurations", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for payment method configurations.
type Iter struct {
	*stripe.Iter
}

// PaymentMethodConfiguration returns the payment method configuration which the iterator is currently pointing to.
func (i *Iter) PaymentMethodConfiguration() *stripe.PaymentMethodConfiguration {
	return i.Current().(*stripe.PaymentMethodConfiguration)
}

// PaymentMethodConfigurationList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) PaymentMethodConfigurationList() *stripe.PaymentMethodConfigurationList {
	return i.List().(*stripe.PaymentMethodConfigurationList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
