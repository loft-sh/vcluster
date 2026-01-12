//
//
// File generated from our OpenAPI spec
//
//

// Package setupintent provides the /setup_intents APIs
package setupintent

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /setup_intents APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Creates a SetupIntent object.
//
// After you create the SetupIntent, attach a payment method and [confirm](https://stripe.com/docs/api/setup_intents/confirm)
// it to collect any required permissions to charge the payment method later.
func New(params *stripe.SetupIntentParams) (*stripe.SetupIntent, error) {
	return getC().New(params)
}

// Creates a SetupIntent object.
//
// After you create the SetupIntent, attach a payment method and [confirm](https://stripe.com/docs/api/setup_intents/confirm)
// it to collect any required permissions to charge the payment method later.
func (c Client) New(params *stripe.SetupIntentParams) (*stripe.SetupIntent, error) {
	setupintent := &stripe.SetupIntent{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/setup_intents",
		c.Key,
		params,
		setupintent,
	)
	return setupintent, err
}

// Retrieves the details of a SetupIntent that has previously been created.
//
// Client-side retrieval using a publishable key is allowed when the client_secret is provided in the query string.
//
// When retrieved with a publishable key, only a subset of properties will be returned. Please refer to the [SetupIntent](https://stripe.com/docs/api#setup_intent_object) object reference for more details.
func Get(id string, params *stripe.SetupIntentParams) (*stripe.SetupIntent, error) {
	return getC().Get(id, params)
}

// Retrieves the details of a SetupIntent that has previously been created.
//
// Client-side retrieval using a publishable key is allowed when the client_secret is provided in the query string.
//
// When retrieved with a publishable key, only a subset of properties will be returned. Please refer to the [SetupIntent](https://stripe.com/docs/api#setup_intent_object) object reference for more details.
func (c Client) Get(id string, params *stripe.SetupIntentParams) (*stripe.SetupIntent, error) {
	path := stripe.FormatURLPath("/v1/setup_intents/%s", id)
	setupintent := &stripe.SetupIntent{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, setupintent)
	return setupintent, err
}

// Updates a SetupIntent object.
func Update(id string, params *stripe.SetupIntentParams) (*stripe.SetupIntent, error) {
	return getC().Update(id, params)
}

// Updates a SetupIntent object.
func (c Client) Update(id string, params *stripe.SetupIntentParams) (*stripe.SetupIntent, error) {
	path := stripe.FormatURLPath("/v1/setup_intents/%s", id)
	setupintent := &stripe.SetupIntent{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, setupintent)
	return setupintent, err
}

// You can cancel a SetupIntent object when it's in one of these statuses: requires_payment_method, requires_confirmation, or requires_action.
//
// After you cancel it, setup is abandoned and any operations on the SetupIntent fail with an error. You can't cancel the SetupIntent for a Checkout Session. [Expire the Checkout Session](https://stripe.com/docs/api/checkout/sessions/expire) instead.
func Cancel(id string, params *stripe.SetupIntentCancelParams) (*stripe.SetupIntent, error) {
	return getC().Cancel(id, params)
}

// You can cancel a SetupIntent object when it's in one of these statuses: requires_payment_method, requires_confirmation, or requires_action.
//
// After you cancel it, setup is abandoned and any operations on the SetupIntent fail with an error. You can't cancel the SetupIntent for a Checkout Session. [Expire the Checkout Session](https://stripe.com/docs/api/checkout/sessions/expire) instead.
func (c Client) Cancel(id string, params *stripe.SetupIntentCancelParams) (*stripe.SetupIntent, error) {
	path := stripe.FormatURLPath("/v1/setup_intents/%s/cancel", id)
	setupintent := &stripe.SetupIntent{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, setupintent)
	return setupintent, err
}

// Confirm that your customer intends to set up the current or
// provided payment method. For example, you would confirm a SetupIntent
// when a customer hits the “Save” button on a payment method management
// page on your website.
//
// If the selected payment method does not require any additional
// steps from the customer, the SetupIntent will transition to the
// succeeded status.
//
// Otherwise, it will transition to the requires_action status and
// suggest additional actions via next_action. If setup fails,
// the SetupIntent will transition to the
// requires_payment_method status or the canceled status if the
// confirmation limit is reached.
func Confirm(id string, params *stripe.SetupIntentConfirmParams) (*stripe.SetupIntent, error) {
	return getC().Confirm(id, params)
}

// Confirm that your customer intends to set up the current or
// provided payment method. For example, you would confirm a SetupIntent
// when a customer hits the “Save” button on a payment method management
// page on your website.
//
// If the selected payment method does not require any additional
// steps from the customer, the SetupIntent will transition to the
// succeeded status.
//
// Otherwise, it will transition to the requires_action status and
// suggest additional actions via next_action. If setup fails,
// the SetupIntent will transition to the
// requires_payment_method status or the canceled status if the
// confirmation limit is reached.
func (c Client) Confirm(id string, params *stripe.SetupIntentConfirmParams) (*stripe.SetupIntent, error) {
	path := stripe.FormatURLPath("/v1/setup_intents/%s/confirm", id)
	setupintent := &stripe.SetupIntent{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, setupintent)
	return setupintent, err
}

// Verifies microdeposits on a SetupIntent object.
func VerifyMicrodeposits(id string, params *stripe.SetupIntentVerifyMicrodepositsParams) (*stripe.SetupIntent, error) {
	return getC().VerifyMicrodeposits(id, params)
}

// Verifies microdeposits on a SetupIntent object.
func (c Client) VerifyMicrodeposits(id string, params *stripe.SetupIntentVerifyMicrodepositsParams) (*stripe.SetupIntent, error) {
	path := stripe.FormatURLPath("/v1/setup_intents/%s/verify_microdeposits", id)
	setupintent := &stripe.SetupIntent{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, setupintent)
	return setupintent, err
}

// Returns a list of SetupIntents.
func List(params *stripe.SetupIntentListParams) *Iter {
	return getC().List(params)
}

// Returns a list of SetupIntents.
func (c Client) List(listParams *stripe.SetupIntentListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.SetupIntentList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/setup_intents", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for setup intents.
type Iter struct {
	*stripe.Iter
}

// SetupIntent returns the setup intent which the iterator is currently pointing to.
func (i *Iter) SetupIntent() *stripe.SetupIntent {
	return i.Current().(*stripe.SetupIntent)
}

// SetupIntentList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) SetupIntentList() *stripe.SetupIntentList {
	return i.List().(*stripe.SetupIntentList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
