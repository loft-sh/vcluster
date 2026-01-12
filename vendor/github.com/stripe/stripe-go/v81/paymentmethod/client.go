//
//
// File generated from our OpenAPI spec
//
//

// Package paymentmethod provides the /payment_methods APIs
package paymentmethod

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /payment_methods APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Creates a PaymentMethod object. Read the [Stripe.js reference](https://stripe.com/docs/stripe-js/reference#stripe-create-payment-method) to learn how to create PaymentMethods via Stripe.js.
//
// Instead of creating a PaymentMethod directly, we recommend using the [PaymentIntents API to accept a payment immediately or the <a href="/docs/payments/save-and-reuse">SetupIntent](https://stripe.com/docs/payments/accept-a-payment) API to collect payment method details ahead of a future payment.
func New(params *stripe.PaymentMethodParams) (*stripe.PaymentMethod, error) {
	return getC().New(params)
}

// Creates a PaymentMethod object. Read the [Stripe.js reference](https://stripe.com/docs/stripe-js/reference#stripe-create-payment-method) to learn how to create PaymentMethods via Stripe.js.
//
// Instead of creating a PaymentMethod directly, we recommend using the [PaymentIntents API to accept a payment immediately or the <a href="/docs/payments/save-and-reuse">SetupIntent](https://stripe.com/docs/payments/accept-a-payment) API to collect payment method details ahead of a future payment.
func (c Client) New(params *stripe.PaymentMethodParams) (*stripe.PaymentMethod, error) {
	paymentmethod := &stripe.PaymentMethod{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/payment_methods",
		c.Key,
		params,
		paymentmethod,
	)
	return paymentmethod, err
}

// Retrieves a PaymentMethod object attached to the StripeAccount. To retrieve a payment method attached to a Customer, you should use [Retrieve a Customer's PaymentMethods](https://stripe.com/docs/api/payment_methods/customer)
func Get(id string, params *stripe.PaymentMethodParams) (*stripe.PaymentMethod, error) {
	return getC().Get(id, params)
}

// Retrieves a PaymentMethod object attached to the StripeAccount. To retrieve a payment method attached to a Customer, you should use [Retrieve a Customer's PaymentMethods](https://stripe.com/docs/api/payment_methods/customer)
func (c Client) Get(id string, params *stripe.PaymentMethodParams) (*stripe.PaymentMethod, error) {
	path := stripe.FormatURLPath("/v1/payment_methods/%s", id)
	paymentmethod := &stripe.PaymentMethod{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, paymentmethod)
	return paymentmethod, err
}

// Updates a PaymentMethod object. A PaymentMethod must be attached a customer to be updated.
func Update(id string, params *stripe.PaymentMethodParams) (*stripe.PaymentMethod, error) {
	return getC().Update(id, params)
}

// Updates a PaymentMethod object. A PaymentMethod must be attached a customer to be updated.
func (c Client) Update(id string, params *stripe.PaymentMethodParams) (*stripe.PaymentMethod, error) {
	path := stripe.FormatURLPath("/v1/payment_methods/%s", id)
	paymentmethod := &stripe.PaymentMethod{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, paymentmethod)
	return paymentmethod, err
}

// Attaches a PaymentMethod object to a Customer.
//
// To attach a new PaymentMethod to a customer for future payments, we recommend you use a [SetupIntent](https://stripe.com/docs/api/setup_intents)
// or a PaymentIntent with [setup_future_usage](https://stripe.com/docs/api/payment_intents/create#create_payment_intent-setup_future_usage).
// These approaches will perform any necessary steps to set up the PaymentMethod for future payments. Using the /v1/payment_methods/:id/attach
// endpoint without first using a SetupIntent or PaymentIntent with setup_future_usage does not optimize the PaymentMethod for
// future use, which makes later declines and payment friction more likely.
// See [Optimizing cards for future payments](https://stripe.com/docs/payments/payment-intents#future-usage) for more information about setting up
// future payments.
//
// To use this PaymentMethod as the default for invoice or subscription payments,
// set [invoice_settings.default_payment_method](https://stripe.com/docs/api/customers/update#update_customer-invoice_settings-default_payment_method),
// on the Customer to the PaymentMethod's ID.
func Attach(id string, params *stripe.PaymentMethodAttachParams) (*stripe.PaymentMethod, error) {
	return getC().Attach(id, params)
}

// Attaches a PaymentMethod object to a Customer.
//
// To attach a new PaymentMethod to a customer for future payments, we recommend you use a [SetupIntent](https://stripe.com/docs/api/setup_intents)
// or a PaymentIntent with [setup_future_usage](https://stripe.com/docs/api/payment_intents/create#create_payment_intent-setup_future_usage).
// These approaches will perform any necessary steps to set up the PaymentMethod for future payments. Using the /v1/payment_methods/:id/attach
// endpoint without first using a SetupIntent or PaymentIntent with setup_future_usage does not optimize the PaymentMethod for
// future use, which makes later declines and payment friction more likely.
// See [Optimizing cards for future payments](https://stripe.com/docs/payments/payment-intents#future-usage) for more information about setting up
// future payments.
//
// To use this PaymentMethod as the default for invoice or subscription payments,
// set [invoice_settings.default_payment_method](https://stripe.com/docs/api/customers/update#update_customer-invoice_settings-default_payment_method),
// on the Customer to the PaymentMethod's ID.
func (c Client) Attach(id string, params *stripe.PaymentMethodAttachParams) (*stripe.PaymentMethod, error) {
	path := stripe.FormatURLPath("/v1/payment_methods/%s/attach", id)
	paymentmethod := &stripe.PaymentMethod{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, paymentmethod)
	return paymentmethod, err
}

// Detaches a PaymentMethod object from a Customer. After a PaymentMethod is detached, it can no longer be used for a payment or re-attached to a Customer.
func Detach(id string, params *stripe.PaymentMethodDetachParams) (*stripe.PaymentMethod, error) {
	return getC().Detach(id, params)
}

// Detaches a PaymentMethod object from a Customer. After a PaymentMethod is detached, it can no longer be used for a payment or re-attached to a Customer.
func (c Client) Detach(id string, params *stripe.PaymentMethodDetachParams) (*stripe.PaymentMethod, error) {
	path := stripe.FormatURLPath("/v1/payment_methods/%s/detach", id)
	paymentmethod := &stripe.PaymentMethod{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, paymentmethod)
	return paymentmethod, err
}

// Returns a list of PaymentMethods for Treasury flows. If you want to list the PaymentMethods attached to a Customer for payments, you should use the [List a Customer's PaymentMethods](https://stripe.com/docs/api/payment_methods/customer_list) API instead.
func List(params *stripe.PaymentMethodListParams) *Iter {
	return getC().List(params)
}

// Returns a list of PaymentMethods for Treasury flows. If you want to list the PaymentMethods attached to a Customer for payments, you should use the [List a Customer's PaymentMethods](https://stripe.com/docs/api/payment_methods/customer_list) API instead.
func (c Client) List(listParams *stripe.PaymentMethodListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.PaymentMethodList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/payment_methods", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for payment methods.
type Iter struct {
	*stripe.Iter
}

// PaymentMethod returns the payment method which the iterator is currently pointing to.
func (i *Iter) PaymentMethod() *stripe.PaymentMethod {
	return i.Current().(*stripe.PaymentMethod)
}

// PaymentMethodList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) PaymentMethodList() *stripe.PaymentMethodList {
	return i.List().(*stripe.PaymentMethodList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
