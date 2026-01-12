//
//
// File generated from our OpenAPI spec
//
//

package stripe

// The meter event adjustment's status.
type BillingMeterEventAdjustmentStatus string

// List of values that BillingMeterEventAdjustmentStatus can take
const (
	BillingMeterEventAdjustmentStatusComplete BillingMeterEventAdjustmentStatus = "complete"
	BillingMeterEventAdjustmentStatusPending  BillingMeterEventAdjustmentStatus = "pending"
)

// Specifies whether to cancel a single event or a range of events for a time period. Time period cancellation is not supported yet.
type BillingMeterEventAdjustmentType string

// List of values that BillingMeterEventAdjustmentType can take
const (
	BillingMeterEventAdjustmentTypeCancel BillingMeterEventAdjustmentType = "cancel"
)

// Specifies which event to cancel.
type BillingMeterEventAdjustmentCancelParams struct {
	// Unique identifier for the event. You can only cancel events within 24 hours of Stripe receiving them.
	Identifier *string `form:"identifier"`
}

// Creates a billing meter event adjustment.
type BillingMeterEventAdjustmentParams struct {
	Params `form:"*"`
	// Specifies which event to cancel.
	Cancel *BillingMeterEventAdjustmentCancelParams `form:"cancel"`
	// The name of the meter event. Corresponds with the `event_name` field on a meter.
	EventName *string `form:"event_name"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Specifies whether to cancel a single event or a range of events for a time period. Time period cancellation is not supported yet.
	Type *string `form:"type"`
}

// AddExpand appends a new field to expand.
func (p *BillingMeterEventAdjustmentParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Specifies which event to cancel.
type BillingMeterEventAdjustmentCancel struct {
	// Unique identifier for the event.
	Identifier string `json:"identifier"`
}

// A billing meter event adjustment is a resource that allows you to cancel a meter event. For example, you might create a billing meter event adjustment to cancel a meter event that was created in error or attached to the wrong customer.
type BillingMeterEventAdjustment struct {
	APIResource
	// Specifies which event to cancel.
	Cancel *BillingMeterEventAdjustmentCancel `json:"cancel"`
	// The name of the meter event. Corresponds with the `event_name` field on a meter.
	EventName string `json:"event_name"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The meter event adjustment's status.
	Status BillingMeterEventAdjustmentStatus `json:"status"`
	// Specifies whether to cancel a single event or a range of events for a time period. Time period cancellation is not supported yet.
	Type BillingMeterEventAdjustmentType `json:"type"`
}
