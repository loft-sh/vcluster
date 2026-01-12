//
//
// File generated from our OpenAPI spec
//
//

// Package session provides the /billing_portal/sessions APIs
package session

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
)

// Client is used to invoke /billing_portal/sessions APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Creates a session of the customer portal.
func New(params *stripe.BillingPortalSessionParams) (*stripe.BillingPortalSession, error) {
	return getC().New(params)
}

// Creates a session of the customer portal.
func (c Client) New(params *stripe.BillingPortalSessionParams) (*stripe.BillingPortalSession, error) {
	session := &stripe.BillingPortalSession{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/billing_portal/sessions",
		c.Key,
		params,
		session,
	)
	return session, err
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
