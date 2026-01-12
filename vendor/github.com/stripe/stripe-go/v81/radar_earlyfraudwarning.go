//
//
// File generated from our OpenAPI spec
//
//

package stripe

// The type of fraud labelled by the issuer. One of `card_never_received`, `fraudulent_card_application`, `made_with_counterfeit_card`, `made_with_lost_card`, `made_with_stolen_card`, `misc`, `unauthorized_use_of_card`.
type RadarEarlyFraudWarningFraudType string

// List of values that RadarEarlyFraudWarningFraudType can take
const (
	RadarEarlyFraudWarningFraudTypeCardNeverReceived         RadarEarlyFraudWarningFraudType = "card_never_received"
	RadarEarlyFraudWarningFraudTypeFraudulentCardApplication RadarEarlyFraudWarningFraudType = "fraudulent_card_application"
	RadarEarlyFraudWarningFraudTypeMadeWithCounterfeitCard   RadarEarlyFraudWarningFraudType = "made_with_counterfeit_card"
	RadarEarlyFraudWarningFraudTypeMadeWithLostCard          RadarEarlyFraudWarningFraudType = "made_with_lost_card"
	RadarEarlyFraudWarningFraudTypeMadeWithStolenCard        RadarEarlyFraudWarningFraudType = "made_with_stolen_card"
	RadarEarlyFraudWarningFraudTypeMisc                      RadarEarlyFraudWarningFraudType = "misc"
	RadarEarlyFraudWarningFraudTypeUnauthorizedUseOfCard     RadarEarlyFraudWarningFraudType = "unauthorized_use_of_card"
)

// Returns a list of early fraud warnings.
type RadarEarlyFraudWarningListParams struct {
	ListParams `form:"*"`
	// Only return early fraud warnings for the charge specified by this charge ID.
	Charge *string `form:"charge"`
	// Only return early fraud warnings that were created during the given date interval.
	Created *int64 `form:"created"`
	// Only return early fraud warnings that were created during the given date interval.
	CreatedRange *RangeQueryParams `form:"created"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Only return early fraud warnings for charges that were created by the PaymentIntent specified by this PaymentIntent ID.
	PaymentIntent *string `form:"payment_intent"`
}

// AddExpand appends a new field to expand.
func (p *RadarEarlyFraudWarningListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Retrieves the details of an early fraud warning that has previously been created.
//
// Please refer to the [early fraud warning](https://stripe.com/docs/api#early_fraud_warning_object) object reference for more details.
type RadarEarlyFraudWarningParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *RadarEarlyFraudWarningParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// An early fraud warning indicates that the card issuer has notified us that a
// charge may be fraudulent.
//
// Related guide: [Early fraud warnings](https://stripe.com/docs/disputes/measuring#early-fraud-warnings)
type RadarEarlyFraudWarning struct {
	APIResource
	// An EFW is actionable if it has not received a dispute and has not been fully refunded. You may wish to proactively refund a charge that receives an EFW, in order to avoid receiving a dispute later.
	Actionable bool `json:"actionable"`
	// ID of the charge this early fraud warning is for, optionally expanded.
	Charge *Charge `json:"charge"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// The type of fraud labelled by the issuer. One of `card_never_received`, `fraudulent_card_application`, `made_with_counterfeit_card`, `made_with_lost_card`, `made_with_stolen_card`, `misc`, `unauthorized_use_of_card`.
	FraudType RadarEarlyFraudWarningFraudType `json:"fraud_type"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// ID of the Payment Intent this early fraud warning is for, optionally expanded.
	PaymentIntent *PaymentIntent `json:"payment_intent"`
}

// RadarEarlyFraudWarningList is a list of EarlyFraudWarnings as retrieved from a list endpoint.
type RadarEarlyFraudWarningList struct {
	APIResource
	ListMeta
	Data []*RadarEarlyFraudWarning `json:"data"`
}
