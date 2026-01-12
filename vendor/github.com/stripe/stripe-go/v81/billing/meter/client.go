//
//
// File generated from our OpenAPI spec
//
//

// Package meter provides the /billing/meters APIs
package meter

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /billing/meters APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Creates a billing meter.
func New(params *stripe.BillingMeterParams) (*stripe.BillingMeter, error) {
	return getC().New(params)
}

// Creates a billing meter.
func (c Client) New(params *stripe.BillingMeterParams) (*stripe.BillingMeter, error) {
	meter := &stripe.BillingMeter{}
	err := c.B.Call(http.MethodPost, "/v1/billing/meters", c.Key, params, meter)
	return meter, err
}

// Retrieves a billing meter given an ID.
func Get(id string, params *stripe.BillingMeterParams) (*stripe.BillingMeter, error) {
	return getC().Get(id, params)
}

// Retrieves a billing meter given an ID.
func (c Client) Get(id string, params *stripe.BillingMeterParams) (*stripe.BillingMeter, error) {
	path := stripe.FormatURLPath("/v1/billing/meters/%s", id)
	meter := &stripe.BillingMeter{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, meter)
	return meter, err
}

// Updates a billing meter.
func Update(id string, params *stripe.BillingMeterParams) (*stripe.BillingMeter, error) {
	return getC().Update(id, params)
}

// Updates a billing meter.
func (c Client) Update(id string, params *stripe.BillingMeterParams) (*stripe.BillingMeter, error) {
	path := stripe.FormatURLPath("/v1/billing/meters/%s", id)
	meter := &stripe.BillingMeter{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, meter)
	return meter, err
}

// When a meter is deactivated, no more meter events will be accepted for this meter. You can't attach a deactivated meter to a price.
func Deactivate(id string, params *stripe.BillingMeterDeactivateParams) (*stripe.BillingMeter, error) {
	return getC().Deactivate(id, params)
}

// When a meter is deactivated, no more meter events will be accepted for this meter. You can't attach a deactivated meter to a price.
func (c Client) Deactivate(id string, params *stripe.BillingMeterDeactivateParams) (*stripe.BillingMeter, error) {
	path := stripe.FormatURLPath("/v1/billing/meters/%s/deactivate", id)
	meter := &stripe.BillingMeter{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, meter)
	return meter, err
}

// When a meter is reactivated, events for this meter can be accepted and you can attach the meter to a price.
func Reactivate(id string, params *stripe.BillingMeterReactivateParams) (*stripe.BillingMeter, error) {
	return getC().Reactivate(id, params)
}

// When a meter is reactivated, events for this meter can be accepted and you can attach the meter to a price.
func (c Client) Reactivate(id string, params *stripe.BillingMeterReactivateParams) (*stripe.BillingMeter, error) {
	path := stripe.FormatURLPath("/v1/billing/meters/%s/reactivate", id)
	meter := &stripe.BillingMeter{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, meter)
	return meter, err
}

// Retrieve a list of billing meters.
func List(params *stripe.BillingMeterListParams) *Iter {
	return getC().List(params)
}

// Retrieve a list of billing meters.
func (c Client) List(listParams *stripe.BillingMeterListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.BillingMeterList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/billing/meters", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for billing meters.
type Iter struct {
	*stripe.Iter
}

// BillingMeter returns the billing meter which the iterator is currently pointing to.
func (i *Iter) BillingMeter() *stripe.BillingMeter {
	return i.Current().(*stripe.BillingMeter)
}

// BillingMeterList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) BillingMeterList() *stripe.BillingMeterList {
	return i.List().(*stripe.BillingMeterList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
