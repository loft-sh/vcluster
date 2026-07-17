// Package specpriority assigns ginkgo SpecPriority decorators from historical
// per-spec duration data so the suite dispatches the heaviest execution groups
// first across the parallel workers (a Longest-Processing-Time schedule), keeping
// wall-clock time short and stable.
//
// The data is a daily per-spec duration export, downloaded to the path named by
// E2E_SPEC_STATS_FILE. When that variable is unset or the file is missing the
// package is a clean no-op and the suite runs in its normal order.
//
// The unit tests here are plain testing.T tests, not Ginkgo specs: they call no
// RunSpecs and use no Ginkgo DSL. That is deliberate. Inside a consumer's e2e tree
// a `ginkgo -r` scan treats any directory with test files as a suite, so a Ginkgo-
// importing package with test files but no RunSpecs would hang the parallel run
// waiting for worker procs that never report back. Here the package ships as an
// imported dependency under pkg/ and is never part of a consumer's `ginkgo -r`
// scan, so plain testing.T tests are safe and run under the normal `go test`
// entrypoint. Keep any tests added here as plain testing.T tests, not Ginkgo specs.
package specpriority

// SpecStat is one leaf (It) aggregate from the duration export. The JSON field
// names match the export emitted by the loft-prod e2e-insights export function.
type SpecStat struct {
	ContainerHierarchy []string `json:"container_hierarchy"`
	Leaf               string   `json:"leaf"`
	P95Seconds         float64  `json:"p95_seconds"`
}

// StatsFile is the top-level export document.
type StatsFile struct {
	Version int        `json:"_version"`
	Specs   []SpecStat `json:"specs"`
}
