package stripe

import (
	"context"
)

//
// Public types
//

// SearchContainer is a general interface for which all search result object structs
// should comply. They achieve this by embedding a SearchMeta struct and
// inheriting its implementation of this interface.
type SearchContainer interface {
	GetSearchMeta() *SearchMeta
}

// SearchMeta is the structure that contains the common properties of the search iterators
type SearchMeta struct {
	HasMore  bool    `json:"has_more"`
	NextPage *string `json:"next_page"`
	URL      string  `json:"url"`
	// TotalCount is the total number of objects in the search result (beyond just
	// on the current page).
	// The value is returned only when `total_count` is specified in `expand` parameter.
	TotalCount *uint32 `json:"total_count"`
}

// GetSearchMeta returns a SearchMeta struct (itself). It exists because any
// structs that embed SearchMeta will inherit it, and thus implement the
// SearchContainer interface.
func (l *SearchMeta) GetSearchMeta() *SearchMeta {
	return l
}

// SearchParams is the structure that contains the common properties
// of any *SearchParams structure.
type SearchParams struct {
	// Context used for request. It may carry deadlines, cancelation signals,
	// and other request-scoped values across API boundaries and between
	// processes.
	//
	// Note that a cancelled or timed out context does not provide any
	// guarantee whether the operation was or was not completed on Stripe's API
	// servers. For certainty, you must either retry with the same idempotency
	// key or query the state of the API.
	Context context.Context `form:"-"`

	Query string  `form:"query"`
	Limit *int64  `form:"limit"`
	Page  *string `form:"page"`
	// Deprecated: Please use Expand in the surrounding struct instead.
	Expand []*string `form:"expand"`

	// Single specifies whether this is a single page iterator. By default,
	// listing through an iterator will automatically grab additional pages as
	// the query progresses. To change this behavior and just load a single
	// page, set this to true.
	Single bool `form:"-"` // Not an API parameter

	// StripeAccount may contain the ID of a connected account. By including
	// this field, the request is made as if it originated from the connected
	// account instead of under the account of the owner of the configured
	// Stripe key.
	StripeAccount *string `form:"-"` // Passed as header
}

// AddExpand on the embedded SearchParams struct is deprecated
// Deprecated: please use .AddExpand of the surrounding struct instead.
func (p *SearchParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// GetSearchParams returns a SearchParams struct (itself). It exists because any
// structs that embed SearchParams will inherit it, and thus implement the
// SearchParamsContainer interface.
func (p *SearchParams) GetSearchParams() *SearchParams {
	return p
}

// GetParams returns SearchParams as a Params struct. It exists because any
// structs that embed Params will inherit it, and thus implement the
// ParamsContainer interface.
func (p *SearchParams) GetParams() *Params {
	return p.ToParams()
}

// SetStripeAccount sets a value for the Stripe-Account header.
func (p *SearchParams) SetStripeAccount(val string) {
	p.StripeAccount = &val
}

// ToParams converts a SearchParams to a Params by moving over any fields that
// have valid targets in the new type. This is useful because fields in
// Params can be injected directly into an http.Request while generally
// SearchParams is only used to build a set of parameters.
func (p *SearchParams) ToParams() *Params {
	return &Params{
		Context:       p.Context,
		StripeAccount: p.StripeAccount,
	}
}

// SearchParamsContainer is a general interface for which all search parameter
// structs should comply. They achieve this by embedding a SearchParams struct
// and inheriting its implementation of this interface.
type SearchParamsContainer interface {
	GetSearchParams() *SearchParams
}
