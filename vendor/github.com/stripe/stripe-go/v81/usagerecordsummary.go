//
//
// File generated from our OpenAPI spec
//
//

package stripe

// For the specified subscription item, returns a list of summary objects. Each object in the list provides usage information that's been summarized from multiple usage records and over a subscription billing period (e.g., 15 usage records in the month of September).
//
// The list is sorted in reverse-chronological order (newest first). The first list item represents the most current usage period that hasn't ended yet. Since new usage records can still be added, the returned summary information for the subscription item's ID should be seen as unstable until the subscription billing period ends.
type UsageRecordSummaryListParams struct {
	ListParams       `form:"*"`
	SubscriptionItem *string `form:"-"` // Included in URL
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *UsageRecordSummaryListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// A usage record summary represents an aggregated view of how much usage was accrued for a subscription item within a subscription billing period.
type UsageRecordSummary struct {
	// Unique identifier for the object.
	ID string `json:"id"`
	// The invoice in which this usage period has been billed for.
	Invoice string `json:"invoice"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string  `json:"object"`
	Period *Period `json:"period"`
	// The ID of the subscription item this summary is describing.
	SubscriptionItem string `json:"subscription_item"`
	// The total usage within this usage period.
	TotalUsage int64 `json:"total_usage"`
}

// UsageRecordSummaryList is a list of UsageRecordSummaries as retrieved from a list endpoint.
type UsageRecordSummaryList struct {
	APIResource
	ListMeta
	Data []*UsageRecordSummary `json:"data"`
}
