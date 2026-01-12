//
//
// File generated from our OpenAPI spec
//
//

package stripe

// To connect to a reader the Stripe Terminal SDK needs to retrieve a short-lived connection token from Stripe, proxied through your server. On your backend, add an endpoint that creates and returns a connection token.
type TerminalConnectionTokenParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// The id of the location that this connection token is scoped to. If specified the connection token will only be usable with readers assigned to that location, otherwise the connection token will be usable with all readers. Note that location scoping only applies to internet-connected readers. For more details, see [the docs on scoping connection tokens](https://docs.stripe.com/terminal/fleet/locations-and-zones?dashboard-or-api=api#connection-tokens).
	Location *string `form:"location"`
}

// AddExpand appends a new field to expand.
func (p *TerminalConnectionTokenParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// A Connection Token is used by the Stripe Terminal SDK to connect to a reader.
//
// Related guide: [Fleet management](https://stripe.com/docs/terminal/fleet/locations)
type TerminalConnectionToken struct {
	APIResource
	// The id of the location that this connection token is scoped to. Note that location scoping only applies to internet-connected readers. For more details, see [the docs on scoping connection tokens](https://docs.stripe.com/terminal/fleet/locations-and-zones?dashboard-or-api=api#connection-tokens).
	Location string `json:"location"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// Your application should pass this token to the Stripe Terminal SDK.
	Secret string `json:"secret"`
}
