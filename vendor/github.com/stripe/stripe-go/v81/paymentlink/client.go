//
//
// File generated from our OpenAPI spec
//
//

// Package paymentlink provides the /payment_links APIs
package paymentlink

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /payment_links APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Creates a payment link.
func New(params *stripe.PaymentLinkParams) (*stripe.PaymentLink, error) {
	return getC().New(params)
}

// Creates a payment link.
func (c Client) New(params *stripe.PaymentLinkParams) (*stripe.PaymentLink, error) {
	paymentlink := &stripe.PaymentLink{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/payment_links",
		c.Key,
		params,
		paymentlink,
	)
	return paymentlink, err
}

// Retrieve a payment link.
func Get(id string, params *stripe.PaymentLinkParams) (*stripe.PaymentLink, error) {
	return getC().Get(id, params)
}

// Retrieve a payment link.
func (c Client) Get(id string, params *stripe.PaymentLinkParams) (*stripe.PaymentLink, error) {
	path := stripe.FormatURLPath("/v1/payment_links/%s", id)
	paymentlink := &stripe.PaymentLink{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, paymentlink)
	return paymentlink, err
}

// Updates a payment link.
func Update(id string, params *stripe.PaymentLinkParams) (*stripe.PaymentLink, error) {
	return getC().Update(id, params)
}

// Updates a payment link.
func (c Client) Update(id string, params *stripe.PaymentLinkParams) (*stripe.PaymentLink, error) {
	path := stripe.FormatURLPath("/v1/payment_links/%s", id)
	paymentlink := &stripe.PaymentLink{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, paymentlink)
	return paymentlink, err
}

// Returns a list of your payment links.
func List(params *stripe.PaymentLinkListParams) *Iter {
	return getC().List(params)
}

// Returns a list of your payment links.
func (c Client) List(listParams *stripe.PaymentLinkListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.PaymentLinkList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/payment_links", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for payment links.
type Iter struct {
	*stripe.Iter
}

// PaymentLink returns the payment link which the iterator is currently pointing to.
func (i *Iter) PaymentLink() *stripe.PaymentLink {
	return i.Current().(*stripe.PaymentLink)
}

// PaymentLinkList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) PaymentLinkList() *stripe.PaymentLinkList {
	return i.List().(*stripe.PaymentLinkList)
}

// When retrieving a payment link, there is an includable line_items property containing the first handful of those items. There is also a URL where you can retrieve the full (paginated) list of line items.
func ListLineItems(params *stripe.PaymentLinkListLineItemsParams) *LineItemIter {
	return getC().ListLineItems(params)
}

// When retrieving a payment link, there is an includable line_items property containing the first handful of those items. There is also a URL where you can retrieve the full (paginated) list of line items.
func (c Client) ListLineItems(listParams *stripe.PaymentLinkListLineItemsParams) *LineItemIter {
	path := stripe.FormatURLPath(
		"/v1/payment_links/%s/line_items",
		stripe.StringValue(listParams.PaymentLink),
	)
	return &LineItemIter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.LineItemList{}
			err := c.B.CallRaw(http.MethodGet, path, c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// LineItemIter is an iterator for line items.
type LineItemIter struct {
	*stripe.Iter
}

// LineItem returns the line item which the iterator is currently pointing to.
func (i *LineItemIter) LineItem() *stripe.LineItem {
	return i.Current().(*stripe.LineItem)
}

// LineItemList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *LineItemIter) LineItemList() *stripe.LineItemList {
	return i.List().(*stripe.LineItemList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
