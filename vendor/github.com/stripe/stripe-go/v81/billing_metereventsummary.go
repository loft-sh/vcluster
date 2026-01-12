//
//
// File generated from our OpenAPI spec
//
//

package stripe

// Retrieve a list of billing meter event summaries.
type BillingMeterEventSummaryListParams struct {
	ListParams `form:"*"`
	ID         *string `form:"-"` // Included in URL
	// The customer for which to fetch event summaries.
	Customer *string `form:"customer"`
	// The timestamp from when to stop aggregating meter events (exclusive). Must be aligned with minute boundaries.
	EndTime *int64 `form:"end_time"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// The timestamp from when to start aggregating meter events (inclusive). Must be aligned with minute boundaries.
	StartTime *int64 `form:"start_time"`
	// Specifies what granularity to use when generating event summaries. If not specified, a single event summary would be returned for the specified time range. For hourly granularity, start and end times must align with hour boundaries (e.g., 00:00, 01:00, ..., 23:00). For daily granularity, start and end times must align with UTC day boundaries (00:00 UTC).
	ValueGroupingWindow *string `form:"value_grouping_window"`
}

// AddExpand appends a new field to expand.
func (p *BillingMeterEventSummaryListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// A billing meter event summary represents an aggregated view of a customer's billing meter events within a specified timeframe. It indicates how much
// usage was accrued by a customer for that period.
type BillingMeterEventSummary struct {
	// Aggregated value of all the events within `start_time` (inclusive) and `end_time` (inclusive). The aggregation strategy is defined on meter via `default_aggregation`.
	AggregatedValue float64 `json:"aggregated_value"`
	// End timestamp for this event summary (exclusive). Must be aligned with minute boundaries.
	EndTime int64 `json:"end_time"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// The meter associated with this event summary.
	Meter string `json:"meter"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// Start timestamp for this event summary (inclusive). Must be aligned with minute boundaries.
	StartTime int64 `json:"start_time"`
}

// BillingMeterEventSummaryList is a list of MeterEventSummaries as retrieved from a list endpoint.
type BillingMeterEventSummaryList struct {
	APIResource
	ListMeta
	Data []*BillingMeterEventSummary `json:"data"`
}
