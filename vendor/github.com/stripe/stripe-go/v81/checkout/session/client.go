//
//
// File generated from our OpenAPI spec
//
//

// Package session provides the /checkout/sessions APIs
package session

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /checkout/sessions APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Creates a Session object.
func New(params *stripe.CheckoutSessionParams) (*stripe.CheckoutSession, error) {
	return getC().New(params)
}

// Creates a Session object.
func (c Client) New(params *stripe.CheckoutSessionParams) (*stripe.CheckoutSession, error) {
	session := &stripe.CheckoutSession{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/checkout/sessions",
		c.Key,
		params,
		session,
	)
	return session, err
}

// Retrieves a Session object.
func Get(id string, params *stripe.CheckoutSessionParams) (*stripe.CheckoutSession, error) {
	return getC().Get(id, params)
}

// Retrieves a Session object.
func (c Client) Get(id string, params *stripe.CheckoutSessionParams) (*stripe.CheckoutSession, error) {
	path := stripe.FormatURLPath("/v1/checkout/sessions/%s", id)
	session := &stripe.CheckoutSession{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, session)
	return session, err
}

// Updates a Session object.
func Update(id string, params *stripe.CheckoutSessionParams) (*stripe.CheckoutSession, error) {
	return getC().Update(id, params)
}

// Updates a Session object.
func (c Client) Update(id string, params *stripe.CheckoutSessionParams) (*stripe.CheckoutSession, error) {
	path := stripe.FormatURLPath("/v1/checkout/sessions/%s", id)
	session := &stripe.CheckoutSession{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, session)
	return session, err
}

// A Session can be expired when it is in one of these statuses: open
//
// After it expires, a customer can't complete a Session and customers loading the Session see a message saying the Session is expired.
func Expire(id string, params *stripe.CheckoutSessionExpireParams) (*stripe.CheckoutSession, error) {
	return getC().Expire(id, params)
}

// A Session can be expired when it is in one of these statuses: open
//
// After it expires, a customer can't complete a Session and customers loading the Session see a message saying the Session is expired.
func (c Client) Expire(id string, params *stripe.CheckoutSessionExpireParams) (*stripe.CheckoutSession, error) {
	path := stripe.FormatURLPath("/v1/checkout/sessions/%s/expire", id)
	session := &stripe.CheckoutSession{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, session)
	return session, err
}

// Returns a list of Checkout Sessions.
func List(params *stripe.CheckoutSessionListParams) *Iter {
	return getC().List(params)
}

// Returns a list of Checkout Sessions.
func (c Client) List(listParams *stripe.CheckoutSessionListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.CheckoutSessionList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/checkout/sessions", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for checkout sessions.
type Iter struct {
	*stripe.Iter
}

// CheckoutSession returns the checkout session which the iterator is currently pointing to.
func (i *Iter) CheckoutSession() *stripe.CheckoutSession {
	return i.Current().(*stripe.CheckoutSession)
}

// CheckoutSessionList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) CheckoutSessionList() *stripe.CheckoutSessionList {
	return i.List().(*stripe.CheckoutSessionList)
}

// When retrieving a Checkout Session, there is an includable line_items property containing the first handful of those items. There is also a URL where you can retrieve the full (paginated) list of line items.
func ListLineItems(params *stripe.CheckoutSessionListLineItemsParams) *LineItemIter {
	return getC().ListLineItems(params)
}

// When retrieving a Checkout Session, there is an includable line_items property containing the first handful of those items. There is also a URL where you can retrieve the full (paginated) list of line items.
func (c Client) ListLineItems(listParams *stripe.CheckoutSessionListLineItemsParams) *LineItemIter {
	path := stripe.FormatURLPath(
		"/v1/checkout/sessions/%s/line_items",
		stripe.StringValue(listParams.Session),
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
