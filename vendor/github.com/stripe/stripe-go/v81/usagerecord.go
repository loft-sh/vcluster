//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "github.com/stripe/stripe-go/v81/form"

// Possible values for the action parameter on usage record creation.
const (
	UsageRecordActionIncrement string = "increment"
	UsageRecordActionSet       string = "set"
)

// Creates a usage record for a specified subscription item and date, and fills it with a quantity.
//
// Usage records provide quantity information that Stripe uses to track how much a customer is using your service. With usage information and the pricing model set up by the [metered billing](https://stripe.com/docs/billing/subscriptions/metered-billing) plan, Stripe helps you send accurate invoices to your customers.
//
// The default calculation for usage is to add up all the quantity values of the usage records within a billing period. You can change this default behavior with the billing plan's aggregate_usage [parameter](https://stripe.com/docs/api/plans/create#create_plan-aggregate_usage). When there is more than one usage record with the same timestamp, Stripe adds the quantity values together. In most cases, this is the desired resolution, however, you can change this behavior with the action parameter.
//
// The default pricing model for metered billing is [per-unit pricing. For finer granularity, you can configure metered billing to have a <a href="https://stripe.com/docs/billing/subscriptions/tiers">tiered pricing](https://stripe.com/docs/api/plans/object#plan_object-billing_scheme) model.
type UsageRecordParams struct {
	Params           `form:"*"`
	SubscriptionItem *string `form:"-"` // Included in URL
	// Valid values are `increment` (default) or `set`. When using `increment` the specified `quantity` will be added to the usage at the specified timestamp. The `set` action will overwrite the usage quantity at that timestamp. If the subscription has [billing thresholds](https://stripe.com/docs/api/subscriptions/object#subscription_object-billing_thresholds), `increment` is the only allowed value.
	Action *string `form:"action"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// The usage quantity for the specified timestamp.
	Quantity *int64 `form:"quantity"`
	// The timestamp for the usage event. This timestamp must be within the current billing period of the subscription of the provided `subscription_item`, and must not be in the future. When passing `"now"`, Stripe records usage for the current time. Default is `"now"` if a value is not provided.
	Timestamp    *int64 `form:"timestamp"`
	TimestampNow *bool  `form:"-"` // See custom AppendTo
}

// AddExpand appends a new field to expand.
func (p *UsageRecordParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AppendTo implements custom encoding logic for UsageRecordParams.
func (p *UsageRecordParams) AppendTo(body *form.Values, keyParts []string) {
	if BoolValue(p.TimestampNow) {
		body.Add(form.FormatKey(append(keyParts, "timestamp")), "now")
	}
}

// Usage records allow you to report customer usage and metrics to Stripe for
// metered billing of subscription prices.
//
// Related guide: [Metered billing](https://stripe.com/docs/billing/subscriptions/metered-billing)
//
// This is our legacy usage-based billing API. See the [updated usage-based billing docs](https://docs.stripe.com/billing/subscriptions/usage-based).
type UsageRecord struct {
	APIResource
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The usage quantity for the specified date.
	Quantity int64 `json:"quantity"`
	// The ID of the subscription item this usage record contains data for.
	SubscriptionItem string `json:"subscription_item"`
	// The timestamp when this usage occurred.
	Timestamp int64 `json:"timestamp"`
}
