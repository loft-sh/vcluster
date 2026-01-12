//
//
// File generated from our OpenAPI spec
//
//

// Package payout provides the /payouts APIs
package payout

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /payouts APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// To send funds to your own bank account, create a new payout object. Your [Stripe balance](https://stripe.com/docs/api#balance) must cover the payout amount. If it doesn't, you receive an “Insufficient Funds” error.
//
// If your API key is in test mode, money won't actually be sent, though every other action occurs as if you're in live mode.
//
// If you create a manual payout on a Stripe account that uses multiple payment source types, you need to specify the source type balance that the payout draws from. The [balance object](https://stripe.com/docs/api#balance_object) details available and pending amounts by source type.
func New(params *stripe.PayoutParams) (*stripe.Payout, error) {
	return getC().New(params)
}

// To send funds to your own bank account, create a new payout object. Your [Stripe balance](https://stripe.com/docs/api#balance) must cover the payout amount. If it doesn't, you receive an “Insufficient Funds” error.
//
// If your API key is in test mode, money won't actually be sent, though every other action occurs as if you're in live mode.
//
// If you create a manual payout on a Stripe account that uses multiple payment source types, you need to specify the source type balance that the payout draws from. The [balance object](https://stripe.com/docs/api#balance_object) details available and pending amounts by source type.
func (c Client) New(params *stripe.PayoutParams) (*stripe.Payout, error) {
	payout := &stripe.Payout{}
	err := c.B.Call(http.MethodPost, "/v1/payouts", c.Key, params, payout)
	return payout, err
}

// Retrieves the details of an existing payout. Supply the unique payout ID from either a payout creation request or the payout list. Stripe returns the corresponding payout information.
func Get(id string, params *stripe.PayoutParams) (*stripe.Payout, error) {
	return getC().Get(id, params)
}

// Retrieves the details of an existing payout. Supply the unique payout ID from either a payout creation request or the payout list. Stripe returns the corresponding payout information.
func (c Client) Get(id string, params *stripe.PayoutParams) (*stripe.Payout, error) {
	path := stripe.FormatURLPath("/v1/payouts/%s", id)
	payout := &stripe.Payout{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, payout)
	return payout, err
}

// Updates the specified payout by setting the values of the parameters you pass. We don't change parameters that you don't provide. This request only accepts the metadata as arguments.
func Update(id string, params *stripe.PayoutParams) (*stripe.Payout, error) {
	return getC().Update(id, params)
}

// Updates the specified payout by setting the values of the parameters you pass. We don't change parameters that you don't provide. This request only accepts the metadata as arguments.
func (c Client) Update(id string, params *stripe.PayoutParams) (*stripe.Payout, error) {
	path := stripe.FormatURLPath("/v1/payouts/%s", id)
	payout := &stripe.Payout{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, payout)
	return payout, err
}

// You can cancel a previously created payout if its status is pending. Stripe refunds the funds to your available balance. You can't cancel automatic Stripe payouts.
func Cancel(id string, params *stripe.PayoutParams) (*stripe.Payout, error) {
	return getC().Cancel(id, params)
}

// You can cancel a previously created payout if its status is pending. Stripe refunds the funds to your available balance. You can't cancel automatic Stripe payouts.
func (c Client) Cancel(id string, params *stripe.PayoutParams) (*stripe.Payout, error) {
	path := stripe.FormatURLPath("/v1/payouts/%s/cancel", id)
	payout := &stripe.Payout{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, payout)
	return payout, err
}

// Reverses a payout by debiting the destination bank account. At this time, you can only reverse payouts for connected accounts to US bank accounts. If the payout is manual and in the pending status, use /v1/payouts/:id/cancel instead.
//
// By requesting a reversal through /v1/payouts/:id/reverse, you confirm that the authorized signatory of the selected bank account authorizes the debit on the bank account and that no other authorization is required.
func Reverse(id string, params *stripe.PayoutReverseParams) (*stripe.Payout, error) {
	return getC().Reverse(id, params)
}

// Reverses a payout by debiting the destination bank account. At this time, you can only reverse payouts for connected accounts to US bank accounts. If the payout is manual and in the pending status, use /v1/payouts/:id/cancel instead.
//
// By requesting a reversal through /v1/payouts/:id/reverse, you confirm that the authorized signatory of the selected bank account authorizes the debit on the bank account and that no other authorization is required.
func (c Client) Reverse(id string, params *stripe.PayoutReverseParams) (*stripe.Payout, error) {
	path := stripe.FormatURLPath("/v1/payouts/%s/reverse", id)
	payout := &stripe.Payout{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, payout)
	return payout, err
}

// Returns a list of existing payouts sent to third-party bank accounts or payouts that Stripe sent to you. The payouts return in sorted order, with the most recently created payouts appearing first.
func List(params *stripe.PayoutListParams) *Iter {
	return getC().List(params)
}

// Returns a list of existing payouts sent to third-party bank accounts or payouts that Stripe sent to you. The payouts return in sorted order, with the most recently created payouts appearing first.
func (c Client) List(listParams *stripe.PayoutListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.PayoutList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/payouts", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for payouts.
type Iter struct {
	*stripe.Iter
}

// Payout returns the payout which the iterator is currently pointing to.
func (i *Iter) Payout() *stripe.Payout {
	return i.Current().(*stripe.Payout)
}

// PayoutList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) PayoutList() *stripe.PayoutList {
	return i.List().(*stripe.PayoutList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
