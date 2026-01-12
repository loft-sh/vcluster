//
//
// File generated from our OpenAPI spec
//
//

// Package paymentmethoddomain provides the /payment_method_domains APIs
package paymentmethoddomain

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /payment_method_domains APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Creates a payment method domain.
func New(params *stripe.PaymentMethodDomainParams) (*stripe.PaymentMethodDomain, error) {
	return getC().New(params)
}

// Creates a payment method domain.
func (c Client) New(params *stripe.PaymentMethodDomainParams) (*stripe.PaymentMethodDomain, error) {
	paymentmethoddomain := &stripe.PaymentMethodDomain{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/payment_method_domains",
		c.Key,
		params,
		paymentmethoddomain,
	)
	return paymentmethoddomain, err
}

// Retrieves the details of an existing payment method domain.
func Get(id string, params *stripe.PaymentMethodDomainParams) (*stripe.PaymentMethodDomain, error) {
	return getC().Get(id, params)
}

// Retrieves the details of an existing payment method domain.
func (c Client) Get(id string, params *stripe.PaymentMethodDomainParams) (*stripe.PaymentMethodDomain, error) {
	path := stripe.FormatURLPath("/v1/payment_method_domains/%s", id)
	paymentmethoddomain := &stripe.PaymentMethodDomain{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, paymentmethoddomain)
	return paymentmethoddomain, err
}

// Updates an existing payment method domain.
func Update(id string, params *stripe.PaymentMethodDomainParams) (*stripe.PaymentMethodDomain, error) {
	return getC().Update(id, params)
}

// Updates an existing payment method domain.
func (c Client) Update(id string, params *stripe.PaymentMethodDomainParams) (*stripe.PaymentMethodDomain, error) {
	path := stripe.FormatURLPath("/v1/payment_method_domains/%s", id)
	paymentmethoddomain := &stripe.PaymentMethodDomain{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, paymentmethoddomain)
	return paymentmethoddomain, err
}

// Some payment methods such as Apple Pay require additional steps to verify a domain. If the requirements weren't satisfied when the domain was created, the payment method will be inactive on the domain.
// The payment method doesn't appear in Elements for this domain until it is active.
//
// To activate a payment method on an existing payment method domain, complete the required validation steps specific to the payment method, and then validate the payment method domain with this endpoint.
//
// Related guides: [Payment method domains](https://stripe.com/docs/payments/payment-methods/pmd-registration).
func Validate(id string, params *stripe.PaymentMethodDomainValidateParams) (*stripe.PaymentMethodDomain, error) {
	return getC().Validate(id, params)
}

// Some payment methods such as Apple Pay require additional steps to verify a domain. If the requirements weren't satisfied when the domain was created, the payment method will be inactive on the domain.
// The payment method doesn't appear in Elements for this domain until it is active.
//
// To activate a payment method on an existing payment method domain, complete the required validation steps specific to the payment method, and then validate the payment method domain with this endpoint.
//
// Related guides: [Payment method domains](https://stripe.com/docs/payments/payment-methods/pmd-registration).
func (c Client) Validate(id string, params *stripe.PaymentMethodDomainValidateParams) (*stripe.PaymentMethodDomain, error) {
	path := stripe.FormatURLPath("/v1/payment_method_domains/%s/validate", id)
	paymentmethoddomain := &stripe.PaymentMethodDomain{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, paymentmethoddomain)
	return paymentmethoddomain, err
}

// Lists the details of existing payment method domains.
func List(params *stripe.PaymentMethodDomainListParams) *Iter {
	return getC().List(params)
}

// Lists the details of existing payment method domains.
func (c Client) List(listParams *stripe.PaymentMethodDomainListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.PaymentMethodDomainList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/payment_method_domains", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for payment method domains.
type Iter struct {
	*stripe.Iter
}

// PaymentMethodDomain returns the payment method domain which the iterator is currently pointing to.
func (i *Iter) PaymentMethodDomain() *stripe.PaymentMethodDomain {
	return i.Current().(*stripe.PaymentMethodDomain)
}

// PaymentMethodDomainList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) PaymentMethodDomainList() *stripe.PaymentMethodDomainList {
	return i.List().(*stripe.PaymentMethodDomainList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
