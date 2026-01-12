//
//
// File generated from our OpenAPI spec
//
//

package stripe

// Status of this report run. This will be `pending` when the run is initially created.
//
//	When the run finishes, this will be set to `succeeded` and the `result` field will be populated.
//	Rarely, we may encounter an error, at which point this will be set to `failed` and the `error` field will be populated.
type ReportingReportRunStatus string

// List of values that ReportingReportRunStatus can take
const (
	ReportingReportRunStatusFailed    ReportingReportRunStatus = "failed"
	ReportingReportRunStatusPending   ReportingReportRunStatus = "pending"
	ReportingReportRunStatusSucceeded ReportingReportRunStatus = "succeeded"
)

// Returns a list of Report Runs, with the most recent appearing first.
type ReportingReportRunListParams struct {
	ListParams `form:"*"`
	// Only return Report Runs that were created during the given date interval.
	Created *int64 `form:"created"`
	// Only return Report Runs that were created during the given date interval.
	CreatedRange *RangeQueryParams `form:"created"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *ReportingReportRunListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Parameters specifying how the report should be run. Different Report Types have different required and optional parameters, listed in the [API Access to Reports](https://stripe.com/docs/reporting/statements/api) documentation.
type ReportingReportRunParametersParams struct {
	// The set of report columns to include in the report output. If omitted, the Report Type is run with its default column set.
	Columns []*string `form:"columns"`
	// Connected account ID to filter for in the report run.
	ConnectedAccount *string `form:"connected_account"`
	// Currency of objects to be included in the report run.
	Currency *string `form:"currency"`
	// Ending timestamp of data to be included in the report run (exclusive).
	IntervalEnd *int64 `form:"interval_end"`
	// Starting timestamp of data to be included in the report run.
	IntervalStart *int64 `form:"interval_start"`
	// Payout ID by which to filter the report run.
	Payout *string `form:"payout"`
	// Category of balance transactions to be included in the report run.
	ReportingCategory *string `form:"reporting_category"`
	// Defaults to `Etc/UTC`. The output timezone for all timestamps in the report. A list of possible time zone values is maintained at the [IANA Time Zone Database](http://www.iana.org/time-zones). Has no effect on `interval_start` or `interval_end`.
	Timezone *string `form:"timezone"`
}

// Creates a new object and begin running the report. (Certain report types require a [live-mode API key](https://stripe.com/docs/keys#test-live-modes).)
type ReportingReportRunParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Parameters specifying how the report should be run. Different Report Types have different required and optional parameters, listed in the [API Access to Reports](https://stripe.com/docs/reporting/statements/api) documentation.
	Parameters *ReportingReportRunParametersParams `form:"parameters"`
	// The ID of the [report type](https://stripe.com/docs/reporting/statements/api#report-types) to run, such as `"balance.summary.1"`.
	ReportType *string `form:"report_type"`
}

// AddExpand appends a new field to expand.
func (p *ReportingReportRunParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

type ReportingReportRunParameters struct {
	// The set of output columns requested for inclusion in the report run.
	Columns []string `json:"columns"`
	// Connected account ID by which to filter the report run.
	ConnectedAccount string `json:"connected_account"`
	// Currency of objects to be included in the report run.
	Currency Currency `json:"currency"`
	// Ending timestamp of data to be included in the report run. Can be any UTC timestamp between 1 second after the user specified `interval_start` and 1 second before this report's last `data_available_end` value.
	IntervalEnd int64 `json:"interval_end"`
	// Starting timestamp of data to be included in the report run. Can be any UTC timestamp between 1 second after this report's `data_available_start` and 1 second before the user specified `interval_end` value.
	IntervalStart int64 `json:"interval_start"`
	// Payout ID by which to filter the report run.
	Payout string `json:"payout"`
	// Category of balance transactions to be included in the report run.
	ReportingCategory string `json:"reporting_category"`
	// Defaults to `Etc/UTC`. The output timezone for all timestamps in the report. A list of possible time zone values is maintained at the [IANA Time Zone Database](http://www.iana.org/time-zones). Has no effect on `interval_start` or `interval_end`.
	Timezone string `json:"timezone"`
}

// The Report Run object represents an instance of a report type generated with
// specific run parameters. Once the object is created, Stripe begins processing the report.
// When the report has finished running, it will give you a reference to a file
// where you can retrieve your results. For an overview, see
// [API Access to Reports](https://stripe.com/docs/reporting/statements/api).
//
// Note that certain report types can only be run based on your live-mode data (not test-mode
// data), and will error when queried without a [live-mode API key](https://stripe.com/docs/keys#test-live-modes).
type ReportingReportRun struct {
	APIResource
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// If something should go wrong during the run, a message about the failure (populated when
	//  `status=failed`).
	Error string `json:"error"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// `true` if the report is run on live mode data and `false` if it is run on test mode data.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object     string                        `json:"object"`
	Parameters *ReportingReportRunParameters `json:"parameters"`
	// The ID of the [report type](https://stripe.com/docs/reports/report-types) to run, such as `"balance.summary.1"`.
	ReportType string `json:"report_type"`
	// The file object representing the result of the report run (populated when
	//  `status=succeeded`).
	Result *File `json:"result"`
	// Status of this report run. This will be `pending` when the run is initially created.
	//  When the run finishes, this will be set to `succeeded` and the `result` field will be populated.
	//  Rarely, we may encounter an error, at which point this will be set to `failed` and the `error` field will be populated.
	Status ReportingReportRunStatus `json:"status"`
	// Timestamp at which this run successfully finished (populated when
	//  `status=succeeded`). Measured in seconds since the Unix epoch.
	SucceededAt int64 `json:"succeeded_at"`
}

// ReportingReportRunList is a list of ReportRuns as retrieved from a list endpoint.
type ReportingReportRunList struct {
	APIResource
	ListMeta
	Data []*ReportingReportRun `json:"data"`
}
