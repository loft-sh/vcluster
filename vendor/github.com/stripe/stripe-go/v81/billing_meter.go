//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// The method for mapping a meter event to a customer.
type BillingMeterCustomerMappingType string

// List of values that BillingMeterCustomerMappingType can take
const (
	BillingMeterCustomerMappingTypeByID BillingMeterCustomerMappingType = "by_id"
)

// Specifies how events are aggregated.
type BillingMeterDefaultAggregationFormula string

// List of values that BillingMeterDefaultAggregationFormula can take
const (
	BillingMeterDefaultAggregationFormulaCount BillingMeterDefaultAggregationFormula = "count"
	BillingMeterDefaultAggregationFormulaSum   BillingMeterDefaultAggregationFormula = "sum"
)

// The time window to pre-aggregate meter events for, if any.
type BillingMeterEventTimeWindow string

// List of values that BillingMeterEventTimeWindow can take
const (
	BillingMeterEventTimeWindowDay  BillingMeterEventTimeWindow = "day"
	BillingMeterEventTimeWindowHour BillingMeterEventTimeWindow = "hour"
)

// The meter's status.
type BillingMeterStatus string

// List of values that BillingMeterStatus can take
const (
	BillingMeterStatusActive   BillingMeterStatus = "active"
	BillingMeterStatusInactive BillingMeterStatus = "inactive"
)

// Retrieve a list of billing meters.
type BillingMeterListParams struct {
	ListParams `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Filter results to only include meters with the given status.
	Status *string `form:"status"`
}

// AddExpand appends a new field to expand.
func (p *BillingMeterListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Fields that specify how to map a meter event to a customer.
type BillingMeterCustomerMappingParams struct {
	// The key in the meter event payload to use for mapping the event to a customer.
	EventPayloadKey *string `form:"event_payload_key"`
	// The method for mapping a meter event to a customer. Must be `by_id`.
	Type *string `form:"type"`
}

// The default settings to aggregate a meter's events with.
type BillingMeterDefaultAggregationParams struct {
	// Specifies how events are aggregated. Allowed values are `count` to count the number of events and `sum` to sum each event's value.
	Formula *string `form:"formula"`
}

// Fields that specify how to calculate a meter event's value.
type BillingMeterValueSettingsParams struct {
	// The key in the usage event payload to use as the value for this meter. For example, if the event payload contains usage on a `bytes_used` field, then set the event_payload_key to "bytes_used".
	EventPayloadKey *string `form:"event_payload_key"`
}

// Creates a billing meter.
type BillingMeterParams struct {
	Params `form:"*"`
	// Fields that specify how to map a meter event to a customer.
	CustomerMapping *BillingMeterCustomerMappingParams `form:"customer_mapping"`
	// The default settings to aggregate a meter's events with.
	DefaultAggregation *BillingMeterDefaultAggregationParams `form:"default_aggregation"`
	// The meter's name. Not visible to the customer.
	DisplayName *string `form:"display_name"`
	// The name of the meter event to record usage for. Corresponds with the `event_name` field on meter events.
	EventName *string `form:"event_name"`
	// The time window to pre-aggregate meter events for, if any.
	EventTimeWindow *string `form:"event_time_window"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Fields that specify how to calculate a meter event's value.
	ValueSettings *BillingMeterValueSettingsParams `form:"value_settings"`
}

// AddExpand appends a new field to expand.
func (p *BillingMeterParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// When a meter is deactivated, no more meter events will be accepted for this meter. You can't attach a deactivated meter to a price.
type BillingMeterDeactivateParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *BillingMeterDeactivateParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// When a meter is reactivated, events for this meter can be accepted and you can attach the meter to a price.
type BillingMeterReactivateParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *BillingMeterReactivateParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

type BillingMeterCustomerMapping struct {
	// The key in the meter event payload to use for mapping the event to a customer.
	EventPayloadKey string `json:"event_payload_key"`
	// The method for mapping a meter event to a customer.
	Type BillingMeterCustomerMappingType `json:"type"`
}
type BillingMeterDefaultAggregation struct {
	// Specifies how events are aggregated.
	Formula BillingMeterDefaultAggregationFormula `json:"formula"`
}
type BillingMeterStatusTransitions struct {
	// The time the meter was deactivated, if any. Measured in seconds since Unix epoch.
	DeactivatedAt int64 `json:"deactivated_at"`
}
type BillingMeterValueSettings struct {
	// The key in the meter event payload to use as the value for this meter.
	EventPayloadKey string `json:"event_payload_key"`
}

// Meters specify how to aggregate meter events over a billing period. Meter events represent the actions that customers take in your system. Meters attach to prices and form the basis of the bill.
//
// Related guide: [Usage based billing](https://docs.stripe.com/billing/subscriptions/usage-based)
type BillingMeter struct {
	APIResource
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created            int64                           `json:"created"`
	CustomerMapping    *BillingMeterCustomerMapping    `json:"customer_mapping"`
	DefaultAggregation *BillingMeterDefaultAggregation `json:"default_aggregation"`
	// The meter's name.
	DisplayName string `json:"display_name"`
	// The name of the meter event to record usage for. Corresponds with the `event_name` field on meter events.
	EventName string `json:"event_name"`
	// The time window to pre-aggregate meter events for, if any.
	EventTimeWindow BillingMeterEventTimeWindow `json:"event_time_window"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The meter's status.
	Status            BillingMeterStatus             `json:"status"`
	StatusTransitions *BillingMeterStatusTransitions `json:"status_transitions"`
	// Time at which the object was last updated. Measured in seconds since the Unix epoch.
	Updated       int64                      `json:"updated"`
	ValueSettings *BillingMeterValueSettings `json:"value_settings"`
}

// BillingMeterList is a list of Meters as retrieved from a list endpoint.
type BillingMeterList struct {
	APIResource
	ListMeta
	Data []*BillingMeter `json:"data"`
}

// UnmarshalJSON handles deserialization of a BillingMeter.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (b *BillingMeter) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		b.ID = id
		return nil
	}

	type billingMeter BillingMeter
	var v billingMeter
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*b = BillingMeter(v)
	return nil
}
