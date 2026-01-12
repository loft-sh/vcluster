//
//
// File generated from our OpenAPI spec
//
//

// Package promotioncode provides the /promotion_codes APIs
package promotioncode

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /promotion_codes APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// A promotion code points to a coupon. You can optionally restrict the code to a specific customer, redemption limit, and expiration date.
func New(params *stripe.PromotionCodeParams) (*stripe.PromotionCode, error) {
	return getC().New(params)
}

// A promotion code points to a coupon. You can optionally restrict the code to a specific customer, redemption limit, and expiration date.
func (c Client) New(params *stripe.PromotionCodeParams) (*stripe.PromotionCode, error) {
	promotioncode := &stripe.PromotionCode{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/promotion_codes",
		c.Key,
		params,
		promotioncode,
	)
	return promotioncode, err
}

// Retrieves the promotion code with the given ID. In order to retrieve a promotion code by the customer-facing code use [list](https://stripe.com/docs/api/promotion_codes/list) with the desired code.
func Get(id string, params *stripe.PromotionCodeParams) (*stripe.PromotionCode, error) {
	return getC().Get(id, params)
}

// Retrieves the promotion code with the given ID. In order to retrieve a promotion code by the customer-facing code use [list](https://stripe.com/docs/api/promotion_codes/list) with the desired code.
func (c Client) Get(id string, params *stripe.PromotionCodeParams) (*stripe.PromotionCode, error) {
	path := stripe.FormatURLPath("/v1/promotion_codes/%s", id)
	promotioncode := &stripe.PromotionCode{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, promotioncode)
	return promotioncode, err
}

// Updates the specified promotion code by setting the values of the parameters passed. Most fields are, by design, not editable.
func Update(id string, params *stripe.PromotionCodeParams) (*stripe.PromotionCode, error) {
	return getC().Update(id, params)
}

// Updates the specified promotion code by setting the values of the parameters passed. Most fields are, by design, not editable.
func (c Client) Update(id string, params *stripe.PromotionCodeParams) (*stripe.PromotionCode, error) {
	path := stripe.FormatURLPath("/v1/promotion_codes/%s", id)
	promotioncode := &stripe.PromotionCode{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, promotioncode)
	return promotioncode, err
}

// Returns a list of your promotion codes.
func List(params *stripe.PromotionCodeListParams) *Iter {
	return getC().List(params)
}

// Returns a list of your promotion codes.
func (c Client) List(listParams *stripe.PromotionCodeListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.PromotionCodeList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/promotion_codes", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for promotion codes.
type Iter struct {
	*stripe.Iter
}

// PromotionCode returns the promotion code which the iterator is currently pointing to.
func (i *Iter) PromotionCode() *stripe.PromotionCode {
	return i.Current().(*stripe.PromotionCode)
}

// PromotionCodeList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) PromotionCodeList() *stripe.PromotionCodeList {
	return i.List().(*stripe.PromotionCodeList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
