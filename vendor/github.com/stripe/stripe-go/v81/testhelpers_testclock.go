//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// The status of the Test Clock.
type TestHelpersTestClockStatus string

// List of values that TestHelpersTestClockStatus can take
const (
	TestHelpersTestClockStatusAdvancing       TestHelpersTestClockStatus = "advancing"
	TestHelpersTestClockStatusInternalFailure TestHelpersTestClockStatus = "internal_failure"
	TestHelpersTestClockStatusReady           TestHelpersTestClockStatus = "ready"
)

// Deletes a test clock.
type TestHelpersTestClockParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// The initial frozen time for this test clock.
	FrozenTime *int64 `form:"frozen_time"`
	// The name for this test clock.
	Name *string `form:"name"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersTestClockParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Returns a list of your test clocks.
type TestHelpersTestClockListParams struct {
	ListParams `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersTestClockListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Starts advancing a test clock to a specified time in the future. Advancement is done when status changes to Ready.
type TestHelpersTestClockAdvanceParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// The time to advance the test clock. Must be after the test clock's current frozen time. Cannot be more than two intervals in the future from the shortest subscription in this test clock. If there are no subscriptions in this test clock, it cannot be more than two years in the future.
	FrozenTime *int64 `form:"frozen_time"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersTestClockAdvanceParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

type TestHelpersTestClockStatusDetailsAdvancing struct {
	// The `frozen_time` that the Test Clock is advancing towards.
	TargetFrozenTime int64 `json:"target_frozen_time"`
}
type TestHelpersTestClockStatusDetails struct {
	Advancing *TestHelpersTestClockStatusDetailsAdvancing `json:"advancing"`
}

// A test clock enables deterministic control over objects in testmode. With a test clock, you can create
// objects at a frozen time in the past or future, and advance to a specific future time to observe webhooks and state changes. After the clock advances,
// you can either validate the current state of your scenario (and test your assumptions), change the current state of your scenario (and test more complex scenarios), or keep advancing forward in time.
type TestHelpersTestClock struct {
	APIResource
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	Deleted bool  `json:"deleted"`
	// Time at which this clock is scheduled to auto delete.
	DeletesAfter int64 `json:"deletes_after"`
	// Time at which all objects belonging to this clock are frozen.
	FrozenTime int64 `json:"frozen_time"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// The custom name supplied at creation.
	Name string `json:"name"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The status of the Test Clock.
	Status        TestHelpersTestClockStatus         `json:"status"`
	StatusDetails *TestHelpersTestClockStatusDetails `json:"status_details"`
}

// TestHelpersTestClockList is a list of TestClocks as retrieved from a list endpoint.
type TestHelpersTestClockList struct {
	APIResource
	ListMeta
	Data []*TestHelpersTestClock `json:"data"`
}

// UnmarshalJSON handles deserialization of a TestHelpersTestClock.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (t *TestHelpersTestClock) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		t.ID = id
		return nil
	}

	type testHelpersTestClock TestHelpersTestClock
	var v testHelpersTestClock
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*t = TestHelpersTestClock(v)
	return nil
}
