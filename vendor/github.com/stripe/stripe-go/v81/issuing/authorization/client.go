//
//
// File generated from our OpenAPI spec
//
//

// Package authorization provides the /issuing/authorizations APIs
package authorization

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /issuing/authorizations APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Retrieves an Issuing Authorization object.
func Get(id string, params *stripe.IssuingAuthorizationParams) (*stripe.IssuingAuthorization, error) {
	return getC().Get(id, params)
}

// Retrieves an Issuing Authorization object.
func (c Client) Get(id string, params *stripe.IssuingAuthorizationParams) (*stripe.IssuingAuthorization, error) {
	path := stripe.FormatURLPath("/v1/issuing/authorizations/%s", id)
	authorization := &stripe.IssuingAuthorization{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, authorization)
	return authorization, err
}

// Updates the specified Issuing Authorization object by setting the values of the parameters passed. Any parameters not provided will be left unchanged.
func Update(id string, params *stripe.IssuingAuthorizationParams) (*stripe.IssuingAuthorization, error) {
	return getC().Update(id, params)
}

// Updates the specified Issuing Authorization object by setting the values of the parameters passed. Any parameters not provided will be left unchanged.
func (c Client) Update(id string, params *stripe.IssuingAuthorizationParams) (*stripe.IssuingAuthorization, error) {
	path := stripe.FormatURLPath("/v1/issuing/authorizations/%s", id)
	authorization := &stripe.IssuingAuthorization{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, authorization)
	return authorization, err
}

// [Deprecated] Approves a pending Issuing Authorization object. This request should be made within the timeout window of the [real-time authorization](https://stripe.com/docs/issuing/controls/real-time-authorizations) flow.
// This method is deprecated. Instead, [respond directly to the webhook request to approve an authorization](https://stripe.com/docs/issuing/controls/real-time-authorizations#authorization-handling).
// Deprecated:
func Approve(id string, params *stripe.IssuingAuthorizationApproveParams) (*stripe.IssuingAuthorization, error) {
	return getC().Approve(id, params)
}

// [Deprecated] Approves a pending Issuing Authorization object. This request should be made within the timeout window of the [real-time authorization](https://stripe.com/docs/issuing/controls/real-time-authorizations) flow.
// This method is deprecated. Instead, [respond directly to the webhook request to approve an authorization](https://stripe.com/docs/issuing/controls/real-time-authorizations#authorization-handling).
// Deprecated:
func (c Client) Approve(id string, params *stripe.IssuingAuthorizationApproveParams) (*stripe.IssuingAuthorization, error) {
	path := stripe.FormatURLPath("/v1/issuing/authorizations/%s/approve", id)
	authorization := &stripe.IssuingAuthorization{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, authorization)
	return authorization, err
}

// [Deprecated] Declines a pending Issuing Authorization object. This request should be made within the timeout window of the [real time authorization](https://stripe.com/docs/issuing/controls/real-time-authorizations) flow.
// This method is deprecated. Instead, [respond directly to the webhook request to decline an authorization](https://stripe.com/docs/issuing/controls/real-time-authorizations#authorization-handling).
// Deprecated:
func Decline(id string, params *stripe.IssuingAuthorizationDeclineParams) (*stripe.IssuingAuthorization, error) {
	return getC().Decline(id, params)
}

// [Deprecated] Declines a pending Issuing Authorization object. This request should be made within the timeout window of the [real time authorization](https://stripe.com/docs/issuing/controls/real-time-authorizations) flow.
// This method is deprecated. Instead, [respond directly to the webhook request to decline an authorization](https://stripe.com/docs/issuing/controls/real-time-authorizations#authorization-handling).
// Deprecated:
func (c Client) Decline(id string, params *stripe.IssuingAuthorizationDeclineParams) (*stripe.IssuingAuthorization, error) {
	path := stripe.FormatURLPath("/v1/issuing/authorizations/%s/decline", id)
	authorization := &stripe.IssuingAuthorization{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, authorization)
	return authorization, err
}

// Returns a list of Issuing Authorization objects. The objects are sorted in descending order by creation date, with the most recently created object appearing first.
func List(params *stripe.IssuingAuthorizationListParams) *Iter {
	return getC().List(params)
}

// Returns a list of Issuing Authorization objects. The objects are sorted in descending order by creation date, with the most recently created object appearing first.
func (c Client) List(listParams *stripe.IssuingAuthorizationListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.IssuingAuthorizationList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/issuing/authorizations", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for issuing authorizations.
type Iter struct {
	*stripe.Iter
}

// IssuingAuthorization returns the issuing authorization which the iterator is currently pointing to.
func (i *Iter) IssuingAuthorization() *stripe.IssuingAuthorization {
	return i.Current().(*stripe.IssuingAuthorization)
}

// IssuingAuthorizationList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) IssuingAuthorizationList() *stripe.IssuingAuthorizationList {
	return i.List().(*stripe.IssuingAuthorizationList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
