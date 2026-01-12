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
)

// Client is used to invoke /issuing/authorizations APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Create a test-mode authorization.
func New(params *stripe.TestHelpersIssuingAuthorizationParams) (*stripe.IssuingAuthorization, error) {
	return getC().New(params)
}

// Create a test-mode authorization.
func (c Client) New(params *stripe.TestHelpersIssuingAuthorizationParams) (*stripe.IssuingAuthorization, error) {
	authorization := &stripe.IssuingAuthorization{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/test_helpers/issuing/authorizations",
		c.Key,
		params,
		authorization,
	)
	return authorization, err
}

// Capture a test-mode authorization.
func Capture(id string, params *stripe.TestHelpersIssuingAuthorizationCaptureParams) (*stripe.IssuingAuthorization, error) {
	return getC().Capture(id, params)
}

// Capture a test-mode authorization.
func (c Client) Capture(id string, params *stripe.TestHelpersIssuingAuthorizationCaptureParams) (*stripe.IssuingAuthorization, error) {
	path := stripe.FormatURLPath(
		"/v1/test_helpers/issuing/authorizations/%s/capture",
		id,
	)
	authorization := &stripe.IssuingAuthorization{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, authorization)
	return authorization, err
}

// Expire a test-mode Authorization.
func Expire(id string, params *stripe.TestHelpersIssuingAuthorizationExpireParams) (*stripe.IssuingAuthorization, error) {
	return getC().Expire(id, params)
}

// Expire a test-mode Authorization.
func (c Client) Expire(id string, params *stripe.TestHelpersIssuingAuthorizationExpireParams) (*stripe.IssuingAuthorization, error) {
	path := stripe.FormatURLPath(
		"/v1/test_helpers/issuing/authorizations/%s/expire",
		id,
	)
	authorization := &stripe.IssuingAuthorization{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, authorization)
	return authorization, err
}

// Finalize the amount on an Authorization prior to capture, when the initial authorization was for an estimated amount.
func FinalizeAmount(id string, params *stripe.TestHelpersIssuingAuthorizationFinalizeAmountParams) (*stripe.IssuingAuthorization, error) {
	return getC().FinalizeAmount(id, params)
}

// Finalize the amount on an Authorization prior to capture, when the initial authorization was for an estimated amount.
func (c Client) FinalizeAmount(id string, params *stripe.TestHelpersIssuingAuthorizationFinalizeAmountParams) (*stripe.IssuingAuthorization, error) {
	path := stripe.FormatURLPath(
		"/v1/test_helpers/issuing/authorizations/%s/finalize_amount",
		id,
	)
	authorization := &stripe.IssuingAuthorization{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, authorization)
	return authorization, err
}

// Increment a test-mode Authorization.
func Increment(id string, params *stripe.TestHelpersIssuingAuthorizationIncrementParams) (*stripe.IssuingAuthorization, error) {
	return getC().Increment(id, params)
}

// Increment a test-mode Authorization.
func (c Client) Increment(id string, params *stripe.TestHelpersIssuingAuthorizationIncrementParams) (*stripe.IssuingAuthorization, error) {
	path := stripe.FormatURLPath(
		"/v1/test_helpers/issuing/authorizations/%s/increment",
		id,
	)
	authorization := &stripe.IssuingAuthorization{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, authorization)
	return authorization, err
}

// Respond to a fraud challenge on a testmode Issuing authorization, simulating either a confirmation of fraud or a correction of legitimacy.
func Respond(id string, params *stripe.TestHelpersIssuingAuthorizationRespondParams) (*stripe.IssuingAuthorization, error) {
	return getC().Respond(id, params)
}

// Respond to a fraud challenge on a testmode Issuing authorization, simulating either a confirmation of fraud or a correction of legitimacy.
func (c Client) Respond(id string, params *stripe.TestHelpersIssuingAuthorizationRespondParams) (*stripe.IssuingAuthorization, error) {
	path := stripe.FormatURLPath(
		"/v1/test_helpers/issuing/authorizations/%s/fraud_challenges/respond",
		id,
	)
	authorization := &stripe.IssuingAuthorization{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, authorization)
	return authorization, err
}

// Reverse a test-mode Authorization.
func Reverse(id string, params *stripe.TestHelpersIssuingAuthorizationReverseParams) (*stripe.IssuingAuthorization, error) {
	return getC().Reverse(id, params)
}

// Reverse a test-mode Authorization.
func (c Client) Reverse(id string, params *stripe.TestHelpersIssuingAuthorizationReverseParams) (*stripe.IssuingAuthorization, error) {
	path := stripe.FormatURLPath(
		"/v1/test_helpers/issuing/authorizations/%s/reverse",
		id,
	)
	authorization := &stripe.IssuingAuthorization{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, authorization)
	return authorization, err
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
