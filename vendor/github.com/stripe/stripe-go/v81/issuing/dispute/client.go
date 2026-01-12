//
//
// File generated from our OpenAPI spec
//
//

// Package dispute provides the /issuing/disputes APIs
package dispute

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /issuing/disputes APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Creates an Issuing Dispute object. Individual pieces of evidence within the evidence object are optional at this point. Stripe only validates that required evidence is present during submission. Refer to [Dispute reasons and evidence](https://stripe.com/docs/issuing/purchases/disputes#dispute-reasons-and-evidence) for more details about evidence requirements.
func New(params *stripe.IssuingDisputeParams) (*stripe.IssuingDispute, error) {
	return getC().New(params)
}

// Creates an Issuing Dispute object. Individual pieces of evidence within the evidence object are optional at this point. Stripe only validates that required evidence is present during submission. Refer to [Dispute reasons and evidence](https://stripe.com/docs/issuing/purchases/disputes#dispute-reasons-and-evidence) for more details about evidence requirements.
func (c Client) New(params *stripe.IssuingDisputeParams) (*stripe.IssuingDispute, error) {
	dispute := &stripe.IssuingDispute{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/issuing/disputes",
		c.Key,
		params,
		dispute,
	)
	return dispute, err
}

// Retrieves an Issuing Dispute object.
func Get(id string, params *stripe.IssuingDisputeParams) (*stripe.IssuingDispute, error) {
	return getC().Get(id, params)
}

// Retrieves an Issuing Dispute object.
func (c Client) Get(id string, params *stripe.IssuingDisputeParams) (*stripe.IssuingDispute, error) {
	path := stripe.FormatURLPath("/v1/issuing/disputes/%s", id)
	dispute := &stripe.IssuingDispute{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, dispute)
	return dispute, err
}

// Updates the specified Issuing Dispute object by setting the values of the parameters passed. Any parameters not provided will be left unchanged. Properties on the evidence object can be unset by passing in an empty string.
func Update(id string, params *stripe.IssuingDisputeParams) (*stripe.IssuingDispute, error) {
	return getC().Update(id, params)
}

// Updates the specified Issuing Dispute object by setting the values of the parameters passed. Any parameters not provided will be left unchanged. Properties on the evidence object can be unset by passing in an empty string.
func (c Client) Update(id string, params *stripe.IssuingDisputeParams) (*stripe.IssuingDispute, error) {
	path := stripe.FormatURLPath("/v1/issuing/disputes/%s", id)
	dispute := &stripe.IssuingDispute{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, dispute)
	return dispute, err
}

// Submits an Issuing Dispute to the card network. Stripe validates that all evidence fields required for the dispute's reason are present. For more details, see [Dispute reasons and evidence](https://stripe.com/docs/issuing/purchases/disputes#dispute-reasons-and-evidence).
func Submit(id string, params *stripe.IssuingDisputeSubmitParams) (*stripe.IssuingDispute, error) {
	return getC().Submit(id, params)
}

// Submits an Issuing Dispute to the card network. Stripe validates that all evidence fields required for the dispute's reason are present. For more details, see [Dispute reasons and evidence](https://stripe.com/docs/issuing/purchases/disputes#dispute-reasons-and-evidence).
func (c Client) Submit(id string, params *stripe.IssuingDisputeSubmitParams) (*stripe.IssuingDispute, error) {
	path := stripe.FormatURLPath("/v1/issuing/disputes/%s/submit", id)
	dispute := &stripe.IssuingDispute{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, dispute)
	return dispute, err
}

// Returns a list of Issuing Dispute objects. The objects are sorted in descending order by creation date, with the most recently created object appearing first.
func List(params *stripe.IssuingDisputeListParams) *Iter {
	return getC().List(params)
}

// Returns a list of Issuing Dispute objects. The objects are sorted in descending order by creation date, with the most recently created object appearing first.
func (c Client) List(listParams *stripe.IssuingDisputeListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.IssuingDisputeList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/issuing/disputes", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for issuing disputes.
type Iter struct {
	*stripe.Iter
}

// IssuingDispute returns the issuing dispute which the iterator is currently pointing to.
func (i *Iter) IssuingDispute() *stripe.IssuingDispute {
	return i.Current().(*stripe.IssuingDispute)
}

// IssuingDisputeList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) IssuingDisputeList() *stripe.IssuingDisputeList {
	return i.List().(*stripe.IssuingDisputeList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
