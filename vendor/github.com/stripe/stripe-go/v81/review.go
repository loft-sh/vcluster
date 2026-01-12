//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// The reason the review was closed, or null if it has not yet been closed. One of `approved`, `refunded`, `refunded_as_fraud`, `disputed`, or `redacted`.
type ReviewClosedReason string

// List of values that ReviewClosedReason can take
const (
	ReviewClosedReasonApproved        ReviewClosedReason = "approved"
	ReviewClosedReasonDisputed        ReviewClosedReason = "disputed"
	ReviewClosedReasonRedacted        ReviewClosedReason = "redacted"
	ReviewClosedReasonRefunded        ReviewClosedReason = "refunded"
	ReviewClosedReasonRefundedAsFraud ReviewClosedReason = "refunded_as_fraud"
)

// The reason the review was opened. One of `rule` or `manual`.
type ReviewOpenedReason string

// List of values that ReviewOpenedReason can take
const (
	ReviewOpenedReasonManual ReviewOpenedReason = "manual"
	ReviewOpenedReasonRule   ReviewOpenedReason = "rule"
)

// The reason the review is currently open or closed. One of `rule`, `manual`, `approved`, `refunded`, `refunded_as_fraud`, `disputed`, or `redacted`.
type ReviewReason string

// List of values that ReviewReason can take
const (
	ReviewReasonApproved        ReviewReason = "approved"
	ReviewReasonDisputed        ReviewReason = "disputed"
	ReviewReasonManual          ReviewReason = "manual"
	ReviewReasonRefunded        ReviewReason = "refunded"
	ReviewReasonRefundedAsFraud ReviewReason = "refunded_as_fraud"
	ReviewReasonRedacted        ReviewReason = "redacted"
	ReviewReasonRule            ReviewReason = "rule"
)

// Returns a list of Review objects that have open set to true. The objects are sorted in descending order by creation date, with the most recently created object appearing first.
type ReviewListParams struct {
	ListParams `form:"*"`
	// Only return reviews that were created during the given date interval.
	Created *int64 `form:"created"`
	// Only return reviews that were created during the given date interval.
	CreatedRange *RangeQueryParams `form:"created"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *ReviewListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Retrieves a Review object.
type ReviewParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *ReviewParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Approves a Review object, closing it and removing it from the list of reviews.
type ReviewApproveParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *ReviewApproveParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Information related to the location of the payment. Note that this information is an approximation and attempts to locate the nearest population center - it should not be used to determine a specific address.
type ReviewIPAddressLocation struct {
	// The city where the payment originated.
	City string `json:"city"`
	// Two-letter ISO code representing the country where the payment originated.
	Country string `json:"country"`
	// The geographic latitude where the payment originated.
	Latitude float64 `json:"latitude"`
	// The geographic longitude where the payment originated.
	Longitude float64 `json:"longitude"`
	// The state/county/province/region where the payment originated.
	Region string `json:"region"`
}

// Information related to the browsing session of the user who initiated the payment.
type ReviewSession struct {
	// The browser used in this browser session (e.g., `Chrome`).
	Browser string `json:"browser"`
	// Information about the device used for the browser session (e.g., `Samsung SM-G930T`).
	Device string `json:"device"`
	// The platform for the browser session (e.g., `Macintosh`).
	Platform string `json:"platform"`
	// The version for the browser session (e.g., `61.0.3163.100`).
	Version string `json:"version"`
}

// Reviews can be used to supplement automated fraud detection with human expertise.
//
// Learn more about [Radar](https://stripe.com/radar) and reviewing payments
// [here](https://stripe.com/docs/radar/reviews).
type Review struct {
	APIResource
	// The ZIP or postal code of the card used, if applicable.
	BillingZip string `json:"billing_zip"`
	// The charge associated with this review.
	Charge *Charge `json:"charge"`
	// The reason the review was closed, or null if it has not yet been closed. One of `approved`, `refunded`, `refunded_as_fraud`, `disputed`, or `redacted`.
	ClosedReason ReviewClosedReason `json:"closed_reason"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// The IP address where the payment originated.
	IPAddress string `json:"ip_address"`
	// Information related to the location of the payment. Note that this information is an approximation and attempts to locate the nearest population center - it should not be used to determine a specific address.
	IPAddressLocation *ReviewIPAddressLocation `json:"ip_address_location"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// If `true`, the review needs action.
	Open bool `json:"open"`
	// The reason the review was opened. One of `rule` or `manual`.
	OpenedReason ReviewOpenedReason `json:"opened_reason"`
	// The PaymentIntent ID associated with this review, if one exists.
	PaymentIntent *PaymentIntent `json:"payment_intent"`
	// The reason the review is currently open or closed. One of `rule`, `manual`, `approved`, `refunded`, `refunded_as_fraud`, `disputed`, or `redacted`.
	Reason ReviewReason `json:"reason"`
	// Information related to the browsing session of the user who initiated the payment.
	Session *ReviewSession `json:"session"`
}

// ReviewList is a list of Reviews as retrieved from a list endpoint.
type ReviewList struct {
	APIResource
	ListMeta
	Data []*Review `json:"data"`
}

// UnmarshalJSON handles deserialization of a Review.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (r *Review) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		r.ID = id
		return nil
	}

	type review Review
	var v review
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*r = Review(v)
	return nil
}
