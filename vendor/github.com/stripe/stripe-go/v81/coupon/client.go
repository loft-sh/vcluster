//
//
// File generated from our OpenAPI spec
//
//

// Package coupon provides the /coupons APIs
package coupon

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /coupons APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// You can create coupons easily via the [coupon management](https://dashboard.stripe.com/coupons) page of the Stripe dashboard. Coupon creation is also accessible via the API if you need to create coupons on the fly.
//
// A coupon has either a percent_off or an amount_off and currency. If you set an amount_off, that amount will be subtracted from any invoice's subtotal. For example, an invoice with a subtotal of 100 will have a final total of 0 if a coupon with an amount_off of 200 is applied to it and an invoice with a subtotal of 300 will have a final total of 100 if a coupon with an amount_off of 200 is applied to it.
func New(params *stripe.CouponParams) (*stripe.Coupon, error) {
	return getC().New(params)
}

// You can create coupons easily via the [coupon management](https://dashboard.stripe.com/coupons) page of the Stripe dashboard. Coupon creation is also accessible via the API if you need to create coupons on the fly.
//
// A coupon has either a percent_off or an amount_off and currency. If you set an amount_off, that amount will be subtracted from any invoice's subtotal. For example, an invoice with a subtotal of 100 will have a final total of 0 if a coupon with an amount_off of 200 is applied to it and an invoice with a subtotal of 300 will have a final total of 100 if a coupon with an amount_off of 200 is applied to it.
func (c Client) New(params *stripe.CouponParams) (*stripe.Coupon, error) {
	coupon := &stripe.Coupon{}
	err := c.B.Call(http.MethodPost, "/v1/coupons", c.Key, params, coupon)
	return coupon, err
}

// Retrieves the coupon with the given ID.
func Get(id string, params *stripe.CouponParams) (*stripe.Coupon, error) {
	return getC().Get(id, params)
}

// Retrieves the coupon with the given ID.
func (c Client) Get(id string, params *stripe.CouponParams) (*stripe.Coupon, error) {
	path := stripe.FormatURLPath("/v1/coupons/%s", id)
	coupon := &stripe.Coupon{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, coupon)
	return coupon, err
}

// Updates the metadata of a coupon. Other coupon details (currency, duration, amount_off) are, by design, not editable.
func Update(id string, params *stripe.CouponParams) (*stripe.Coupon, error) {
	return getC().Update(id, params)
}

// Updates the metadata of a coupon. Other coupon details (currency, duration, amount_off) are, by design, not editable.
func (c Client) Update(id string, params *stripe.CouponParams) (*stripe.Coupon, error) {
	path := stripe.FormatURLPath("/v1/coupons/%s", id)
	coupon := &stripe.Coupon{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, coupon)
	return coupon, err
}

// You can delete coupons via the [coupon management](https://dashboard.stripe.com/coupons) page of the Stripe dashboard. However, deleting a coupon does not affect any customers who have already applied the coupon; it means that new customers can't redeem the coupon. You can also delete coupons via the API.
func Del(id string, params *stripe.CouponParams) (*stripe.Coupon, error) {
	return getC().Del(id, params)
}

// You can delete coupons via the [coupon management](https://dashboard.stripe.com/coupons) page of the Stripe dashboard. However, deleting a coupon does not affect any customers who have already applied the coupon; it means that new customers can't redeem the coupon. You can also delete coupons via the API.
func (c Client) Del(id string, params *stripe.CouponParams) (*stripe.Coupon, error) {
	path := stripe.FormatURLPath("/v1/coupons/%s", id)
	coupon := &stripe.Coupon{}
	err := c.B.Call(http.MethodDelete, path, c.Key, params, coupon)
	return coupon, err
}

// Returns a list of your coupons.
func List(params *stripe.CouponListParams) *Iter {
	return getC().List(params)
}

// Returns a list of your coupons.
func (c Client) List(listParams *stripe.CouponListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.CouponList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/coupons", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for coupons.
type Iter struct {
	*stripe.Iter
}

// Coupon returns the coupon which the iterator is currently pointing to.
func (i *Iter) Coupon() *stripe.Coupon {
	return i.Current().(*stripe.Coupon)
}

// CouponList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) CouponList() *stripe.CouponList {
	return i.List().(*stripe.CouponList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
