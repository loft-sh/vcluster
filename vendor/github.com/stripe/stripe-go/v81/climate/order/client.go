//
//
// File generated from our OpenAPI spec
//
//

// Package order provides the /climate/orders APIs
package order

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /climate/orders APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Creates a Climate order object for a given Climate product. The order will be processed immediately
// after creation and payment will be deducted your Stripe balance.
func New(params *stripe.ClimateOrderParams) (*stripe.ClimateOrder, error) {
	return getC().New(params)
}

// Creates a Climate order object for a given Climate product. The order will be processed immediately
// after creation and payment will be deducted your Stripe balance.
func (c Client) New(params *stripe.ClimateOrderParams) (*stripe.ClimateOrder, error) {
	order := &stripe.ClimateOrder{}
	err := c.B.Call(http.MethodPost, "/v1/climate/orders", c.Key, params, order)
	return order, err
}

// Retrieves the details of a Climate order object with the given ID.
func Get(id string, params *stripe.ClimateOrderParams) (*stripe.ClimateOrder, error) {
	return getC().Get(id, params)
}

// Retrieves the details of a Climate order object with the given ID.
func (c Client) Get(id string, params *stripe.ClimateOrderParams) (*stripe.ClimateOrder, error) {
	path := stripe.FormatURLPath("/v1/climate/orders/%s", id)
	order := &stripe.ClimateOrder{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, order)
	return order, err
}

// Updates the specified order by setting the values of the parameters passed.
func Update(id string, params *stripe.ClimateOrderParams) (*stripe.ClimateOrder, error) {
	return getC().Update(id, params)
}

// Updates the specified order by setting the values of the parameters passed.
func (c Client) Update(id string, params *stripe.ClimateOrderParams) (*stripe.ClimateOrder, error) {
	path := stripe.FormatURLPath("/v1/climate/orders/%s", id)
	order := &stripe.ClimateOrder{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, order)
	return order, err
}

// Cancels a Climate order. You can cancel an order within 24 hours of creation. Stripe refunds the
// reservation amount_subtotal, but not the amount_fees for user-triggered cancellations. Frontier
// might cancel reservations if suppliers fail to deliver. If Frontier cancels the reservation, Stripe
// provides 90 days advance notice and refunds the amount_total.
func Cancel(id string, params *stripe.ClimateOrderCancelParams) (*stripe.ClimateOrder, error) {
	return getC().Cancel(id, params)
}

// Cancels a Climate order. You can cancel an order within 24 hours of creation. Stripe refunds the
// reservation amount_subtotal, but not the amount_fees for user-triggered cancellations. Frontier
// might cancel reservations if suppliers fail to deliver. If Frontier cancels the reservation, Stripe
// provides 90 days advance notice and refunds the amount_total.
func (c Client) Cancel(id string, params *stripe.ClimateOrderCancelParams) (*stripe.ClimateOrder, error) {
	path := stripe.FormatURLPath("/v1/climate/orders/%s/cancel", id)
	order := &stripe.ClimateOrder{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, order)
	return order, err
}

// Lists all Climate order objects. The orders are returned sorted by creation date, with the
// most recently created orders appearing first.
func List(params *stripe.ClimateOrderListParams) *Iter {
	return getC().List(params)
}

// Lists all Climate order objects. The orders are returned sorted by creation date, with the
// most recently created orders appearing first.
func (c Client) List(listParams *stripe.ClimateOrderListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.ClimateOrderList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/climate/orders", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for climate orders.
type Iter struct {
	*stripe.Iter
}

// ClimateOrder returns the climate order which the iterator is currently pointing to.
func (i *Iter) ClimateOrder() *stripe.ClimateOrder {
	return i.Current().(*stripe.ClimateOrder)
}

// ClimateOrderList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) ClimateOrderList() *stripe.ClimateOrderList {
	return i.List().(*stripe.ClimateOrderList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
