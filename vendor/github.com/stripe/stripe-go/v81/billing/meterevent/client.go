//
//
// File generated from our OpenAPI spec
//
//

// Package meterevent provides the /billing/meter_events APIs
package meterevent

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
)

// Client is used to invoke /billing/meter_events APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Creates a billing meter event.
func New(params *stripe.BillingMeterEventParams) (*stripe.BillingMeterEvent, error) {
	return getC().New(params)
}

// Creates a billing meter event.
func (c Client) New(params *stripe.BillingMeterEventParams) (*stripe.BillingMeterEvent, error) {
	meterevent := &stripe.BillingMeterEvent{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/billing/meter_events",
		c.Key,
		params,
		meterevent,
	)
	return meterevent, err
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
