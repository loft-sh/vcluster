//
//
// File generated from our OpenAPI spec
//
//

// Package verificationsession provides the /identity/verification_sessions APIs
package verificationsession

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /identity/verification_sessions APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Creates a VerificationSession object.
//
// After the VerificationSession is created, display a verification modal using the session client_secret or send your users to the session's url.
//
// If your API key is in test mode, verification checks won't actually process, though everything else will occur as if in live mode.
//
// Related guide: [Verify your users' identity documents](https://stripe.com/docs/identity/verify-identity-documents)
func New(params *stripe.IdentityVerificationSessionParams) (*stripe.IdentityVerificationSession, error) {
	return getC().New(params)
}

// Creates a VerificationSession object.
//
// After the VerificationSession is created, display a verification modal using the session client_secret or send your users to the session's url.
//
// If your API key is in test mode, verification checks won't actually process, though everything else will occur as if in live mode.
//
// Related guide: [Verify your users' identity documents](https://stripe.com/docs/identity/verify-identity-documents)
func (c Client) New(params *stripe.IdentityVerificationSessionParams) (*stripe.IdentityVerificationSession, error) {
	verificationsession := &stripe.IdentityVerificationSession{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/identity/verification_sessions",
		c.Key,
		params,
		verificationsession,
	)
	return verificationsession, err
}

// Retrieves the details of a VerificationSession that was previously created.
//
// When the session status is requires_input, you can use this method to retrieve a valid
// client_secret or url to allow re-submission.
func Get(id string, params *stripe.IdentityVerificationSessionParams) (*stripe.IdentityVerificationSession, error) {
	return getC().Get(id, params)
}

// Retrieves the details of a VerificationSession that was previously created.
//
// When the session status is requires_input, you can use this method to retrieve a valid
// client_secret or url to allow re-submission.
func (c Client) Get(id string, params *stripe.IdentityVerificationSessionParams) (*stripe.IdentityVerificationSession, error) {
	path := stripe.FormatURLPath("/v1/identity/verification_sessions/%s", id)
	verificationsession := &stripe.IdentityVerificationSession{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, verificationsession)
	return verificationsession, err
}

// Updates a VerificationSession object.
//
// When the session status is requires_input, you can use this method to update the
// verification check and options.
func Update(id string, params *stripe.IdentityVerificationSessionParams) (*stripe.IdentityVerificationSession, error) {
	return getC().Update(id, params)
}

// Updates a VerificationSession object.
//
// When the session status is requires_input, you can use this method to update the
// verification check and options.
func (c Client) Update(id string, params *stripe.IdentityVerificationSessionParams) (*stripe.IdentityVerificationSession, error) {
	path := stripe.FormatURLPath("/v1/identity/verification_sessions/%s", id)
	verificationsession := &stripe.IdentityVerificationSession{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, verificationsession)
	return verificationsession, err
}

// A VerificationSession object can be canceled when it is in requires_input [status](https://stripe.com/docs/identity/how-sessions-work).
//
// Once canceled, future submission attempts are disabled. This cannot be undone. [Learn more](https://stripe.com/docs/identity/verification-sessions#cancel).
func Cancel(id string, params *stripe.IdentityVerificationSessionCancelParams) (*stripe.IdentityVerificationSession, error) {
	return getC().Cancel(id, params)
}

// A VerificationSession object can be canceled when it is in requires_input [status](https://stripe.com/docs/identity/how-sessions-work).
//
// Once canceled, future submission attempts are disabled. This cannot be undone. [Learn more](https://stripe.com/docs/identity/verification-sessions#cancel).
func (c Client) Cancel(id string, params *stripe.IdentityVerificationSessionCancelParams) (*stripe.IdentityVerificationSession, error) {
	path := stripe.FormatURLPath(
		"/v1/identity/verification_sessions/%s/cancel",
		id,
	)
	verificationsession := &stripe.IdentityVerificationSession{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, verificationsession)
	return verificationsession, err
}

// Redact a VerificationSession to remove all collected information from Stripe. This will redact
// the VerificationSession and all objects related to it, including VerificationReports, Events,
// request logs, etc.
//
// A VerificationSession object can be redacted when it is in requires_input or verified
// [status](https://stripe.com/docs/identity/how-sessions-work). Redacting a VerificationSession in requires_action
// state will automatically cancel it.
//
// The redaction process may take up to four days. When the redaction process is in progress, the
// VerificationSession's redaction.status field will be set to processing; when the process is
// finished, it will change to redacted and an identity.verification_session.redacted event
// will be emitted.
//
// Redaction is irreversible. Redacted objects are still accessible in the Stripe API, but all the
// fields that contain personal data will be replaced by the string [redacted] or a similar
// placeholder. The metadata field will also be erased. Redacted objects cannot be updated or
// used for any purpose.
//
// [Learn more](https://stripe.com/docs/identity/verification-sessions#redact).
func Redact(id string, params *stripe.IdentityVerificationSessionRedactParams) (*stripe.IdentityVerificationSession, error) {
	return getC().Redact(id, params)
}

// Redact a VerificationSession to remove all collected information from Stripe. This will redact
// the VerificationSession and all objects related to it, including VerificationReports, Events,
// request logs, etc.
//
// A VerificationSession object can be redacted when it is in requires_input or verified
// [status](https://stripe.com/docs/identity/how-sessions-work). Redacting a VerificationSession in requires_action
// state will automatically cancel it.
//
// The redaction process may take up to four days. When the redaction process is in progress, the
// VerificationSession's redaction.status field will be set to processing; when the process is
// finished, it will change to redacted and an identity.verification_session.redacted event
// will be emitted.
//
// Redaction is irreversible. Redacted objects are still accessible in the Stripe API, but all the
// fields that contain personal data will be replaced by the string [redacted] or a similar
// placeholder. The metadata field will also be erased. Redacted objects cannot be updated or
// used for any purpose.
//
// [Learn more](https://stripe.com/docs/identity/verification-sessions#redact).
func (c Client) Redact(id string, params *stripe.IdentityVerificationSessionRedactParams) (*stripe.IdentityVerificationSession, error) {
	path := stripe.FormatURLPath(
		"/v1/identity/verification_sessions/%s/redact",
		id,
	)
	verificationsession := &stripe.IdentityVerificationSession{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, verificationsession)
	return verificationsession, err
}

// Returns a list of VerificationSessions
func List(params *stripe.IdentityVerificationSessionListParams) *Iter {
	return getC().List(params)
}

// Returns a list of VerificationSessions
func (c Client) List(listParams *stripe.IdentityVerificationSessionListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.IdentityVerificationSessionList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/identity/verification_sessions", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for identity verification sessions.
type Iter struct {
	*stripe.Iter
}

// IdentityVerificationSession returns the identity verification session which the iterator is currently pointing to.
func (i *Iter) IdentityVerificationSession() *stripe.IdentityVerificationSession {
	return i.Current().(*stripe.IdentityVerificationSession)
}

// IdentityVerificationSessionList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) IdentityVerificationSessionList() *stripe.IdentityVerificationSessionList {
	return i.List().(*stripe.IdentityVerificationSessionList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
