//
//
// File generated from our OpenAPI spec
//
//

package stripe

// Deletes a ValueListItem object, removing it from its parent value list.
type RadarValueListItemParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// The value of the item (whose type must match the type of the parent value list).
	Value *string `form:"value"`
	// The identifier of the value list which the created item will be added to.
	ValueList *string `form:"value_list"`
}

// AddExpand appends a new field to expand.
func (p *RadarValueListItemParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Returns a list of ValueListItem objects. The objects are sorted in descending order by creation date, with the most recently created object appearing first.
type RadarValueListItemListParams struct {
	ListParams `form:"*"`
	// Only return items that were created during the given date interval.
	Created *int64 `form:"created"`
	// Only return items that were created during the given date interval.
	CreatedRange *RangeQueryParams `form:"created"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Return items belonging to the parent list whose value matches the specified value (using an "is like" match).
	Value *string `form:"value"`
	// Identifier for the parent value list this item belongs to.
	ValueList *string `form:"value_list"`
}

// AddExpand appends a new field to expand.
func (p *RadarValueListItemListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Value list items allow you to add specific values to a given Radar value list, which can then be used in rules.
//
// Related guide: [Managing list items](https://stripe.com/docs/radar/lists#managing-list-items)
type RadarValueListItem struct {
	APIResource
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// The name or email address of the user who added this item to the value list.
	CreatedBy string `json:"created_by"`
	Deleted   bool   `json:"deleted"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The value of the item.
	Value string `json:"value"`
	// The identifier of the value list this item belongs to.
	ValueList string `json:"value_list"`
}

// RadarValueListItemList is a list of ValueListItems as retrieved from a list endpoint.
type RadarValueListItemList struct {
	APIResource
	ListMeta
	Data []*RadarValueListItem `json:"data"`
}
