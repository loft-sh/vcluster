//
//
// File generated from our OpenAPI spec
//
//

// Package charge provides the /charges APIs
package charge

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /charges APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// This method is no longer recommended—use the [Payment Intents API](https://stripe.com/docs/api/payment_intents)
// to initiate a new payment instead. Confirmation of the PaymentIntent creates the Charge
// object used to request payment.
func New(params *stripe.ChargeParams) (*stripe.Charge, error) {
	return getC().New(params)
}

// This method is no longer recommended—use the [Payment Intents API](https://stripe.com/docs/api/payment_intents)
// to initiate a new payment instead. Confirmation of the PaymentIntent creates the Charge
// object used to request payment.
func (c Client) New(params *stripe.ChargeParams) (*stripe.Charge, error) {
	charge := &stripe.Charge{}
	err := c.B.Call(http.MethodPost, "/v1/charges", c.Key, params, charge)
	return charge, err
}

// Retrieves the details of a charge that has previously been created. Supply the unique charge ID that was returned from your previous request, and Stripe will return the corresponding charge information. The same information is returned when creating or refunding the charge.
func Get(id string, params *stripe.ChargeParams) (*stripe.Charge, error) {
	return getC().Get(id, params)
}

// Retrieves the details of a charge that has previously been created. Supply the unique charge ID that was returned from your previous request, and Stripe will return the corresponding charge information. The same information is returned when creating or refunding the charge.
func (c Client) Get(id string, params *stripe.ChargeParams) (*stripe.Charge, error) {
	path := stripe.FormatURLPath("/v1/charges/%s", id)
	charge := &stripe.Charge{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, charge)
	return charge, err
}

// Updates the specified charge by setting the values of the parameters passed. Any parameters not provided will be left unchanged.
func Update(id string, params *stripe.ChargeParams) (*stripe.Charge, error) {
	return getC().Update(id, params)
}

// Updates the specified charge by setting the values of the parameters passed. Any parameters not provided will be left unchanged.
func (c Client) Update(id string, params *stripe.ChargeParams) (*stripe.Charge, error) {
	path := stripe.FormatURLPath("/v1/charges/%s", id)
	charge := &stripe.Charge{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, charge)
	return charge, err
}

// Capture the payment of an existing, uncaptured charge that was created with the capture option set to false.
//
// Uncaptured payments expire a set number of days after they are created ([7 by default](https://stripe.com/docs/charges/placing-a-hold)), after which they are marked as refunded and capture attempts will fail.
//
// Don't use this method to capture a PaymentIntent-initiated charge. Use [Capture a PaymentIntent](https://stripe.com/docs/api/payment_intents/capture).
func Capture(id string, params *stripe.ChargeCaptureParams) (*stripe.Charge, error) {
	return getC().Capture(id, params)
}

// Capture the payment of an existing, uncaptured charge that was created with the capture option set to false.
//
// Uncaptured payments expire a set number of days after they are created ([7 by default](https://stripe.com/docs/charges/placing-a-hold)), after which they are marked as refunded and capture attempts will fail.
//
// Don't use this method to capture a PaymentIntent-initiated charge. Use [Capture a PaymentIntent](https://stripe.com/docs/api/payment_intents/capture).
func (c Client) Capture(id string, params *stripe.ChargeCaptureParams) (*stripe.Charge, error) {
	path := stripe.FormatURLPath("/v1/charges/%s/capture", id)
	charge := &stripe.Charge{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, charge)
	return charge, err
}

// Returns a list of charges you've previously created. The charges are returned in sorted order, with the most recent charges appearing first.
func List(params *stripe.ChargeListParams) *Iter {
	return getC().List(params)
}

// Returns a list of charges you've previously created. The charges are returned in sorted order, with the most recent charges appearing first.
func (c Client) List(listParams *stripe.ChargeListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.ChargeList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/charges", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for charges.
type Iter struct {
	*stripe.Iter
}

// Charge returns the charge which the iterator is currently pointing to.
func (i *Iter) Charge() *stripe.Charge {
	return i.Current().(*stripe.Charge)
}

// ChargeList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) ChargeList() *stripe.ChargeList {
	return i.List().(*stripe.ChargeList)
}

// Search for charges you've previously created using Stripe's [Search Query Language](https://stripe.com/docs/search#search-query-language).
// Don't use search in read-after-write flows where strict consistency is necessary. Under normal operating
// conditions, data is searchable in less than a minute. Occasionally, propagation of new or updated data can be up
// to an hour behind during outages. Search functionality is not available to merchants in India.
func Search(params *stripe.ChargeSearchParams) *SearchIter {
	return getC().Search(params)
}

// Search for charges you've previously created using Stripe's [Search Query Language](https://stripe.com/docs/search#search-query-language).
// Don't use search in read-after-write flows where strict consistency is necessary. Under normal operating
// conditions, data is searchable in less than a minute. Occasionally, propagation of new or updated data can be up
// to an hour behind during outages. Search functionality is not available to merchants in India.
func (c Client) Search(params *stripe.ChargeSearchParams) *SearchIter {
	return &SearchIter{
		SearchIter: stripe.GetSearchIter(params, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.SearchContainer, error) {
			list := &stripe.ChargeSearchResult{}
			err := c.B.CallRaw(http.MethodGet, "/v1/charges/search", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// SearchIter is an iterator for charges.
type SearchIter struct {
	*stripe.SearchIter
}

// Charge returns the charge which the iterator is currently pointing to.
func (i *SearchIter) Charge() *stripe.Charge {
	return i.Current().(*stripe.Charge)
}

// ChargeSearchResult returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *SearchIter) ChargeSearchResult() *stripe.ChargeSearchResult {
	return i.SearchResult().(*stripe.ChargeSearchResult)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
