//
//
// File generated from our OpenAPI spec
//
//

package stripe

// Creates a billing meter event.
type BillingMeterEventParams struct {
	Params `form:"*"`
	// The name of the meter event. Corresponds with the `event_name` field on a meter.
	EventName *string `form:"event_name"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// A unique identifier for the event. If not provided, one is generated. We recommend using UUID-like identifiers. We will enforce uniqueness within a rolling period of at least 24 hours. The enforcement of uniqueness primarily addresses issues arising from accidental retries or other problems occurring within extremely brief time intervals. This approach helps prevent duplicate entries and ensures data integrity in high-frequency operations.
	Identifier *string `form:"identifier"`
	// The payload of the event. This must contain the fields corresponding to a meter's `customer_mapping.event_payload_key` (default is `stripe_customer_id`) and `value_settings.event_payload_key` (default is `value`). Read more about the [payload](https://docs.stripe.com/billing/subscriptions/usage-based/recording-usage#payload-key-overrides).
	Payload map[string]string `form:"payload"`
	// The time of the event. Measured in seconds since the Unix epoch. Must be within the past 35 calendar days or up to 5 minutes in the future. Defaults to current timestamp if not specified.
	Timestamp *int64 `form:"timestamp"`
}

// AddExpand appends a new field to expand.
func (p *BillingMeterEventParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Meter events represent actions that customers take in your system. You can use meter events to bill a customer based on their usage. Meter events are associated with billing meters, which define both the contents of the event's payload and how to aggregate those events.
type BillingMeterEvent struct {
	APIResource
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// The name of the meter event. Corresponds with the `event_name` field on a meter.
	EventName string `json:"event_name"`
	// A unique identifier for the event.
	Identifier string `json:"identifier"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The payload of the event. This contains the fields corresponding to a meter's `customer_mapping.event_payload_key` (default is `stripe_customer_id`) and `value_settings.event_payload_key` (default is `value`). Read more about the [payload](https://stripe.com/docs/billing/subscriptions/usage-based/recording-usage#payload-key-overrides).
	Payload map[string]string `json:"payload"`
	// The timestamp passed in when creating the event. Measured in seconds since the Unix epoch.
	Timestamp int64 `json:"timestamp"`
}
