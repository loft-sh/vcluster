package specpriority

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
)

const (
	// EnvStatsFile is the path to the duration stats file. When unset or unreadable
	// the transformer is a no-op.
	EnvStatsFile = "E2E_SPEC_STATS_FILE"
	// EnvSpreadSeed salts the per-group spread hash. CI must set it to a value that
	// is the same for all parallel processes of one run but differs between runs
	// (e.g. the GitHub run id), so spec ordering is consistent within a run yet
	// flakes surface in different combinations across runs. When unset the spread is
	// deterministic, which is fine for local single-process runs.
	EnvSpreadSeed = "E2E_SPEC_SPREAD_SEED"
)

var (
	loadOnce    sync.Once
	index       *Index // nil => no data => no-op
	loadedSpecs int

	matched atomic.Int64
	total   atomic.Int64
)

// Transformer is a ginkgo NodeArgsTransformer registered during tree construction.
// It appends a SpecPriority decorator to the execution-group representative nodes
// (outermost Ordered containers and lone Its) based on historical durations, so
// ginkgo dispatches the heaviest groups first across the parallel workers.
//
// It is a no-op when no stats file is configured, and it never overrides a
// manually-set SpecPriority on a node.
func Transformer(nodeType types.NodeType, _ ginkgo.Offset, text string, args []any) (string, []any, []error) {
	load()
	if index == nil {
		return text, args, nil
	}
	if hasExplicitPriority(args) {
		// Respect a deliberate manual priority and let it win.
		return text, args, nil
	}

	ancestors, inOrdered := ancestorsAndOrdered()

	// Classify the node: an outermost Ordered container (the whole subtree is one
	// execution group) or a lone It not inside any Ordered container (its own
	// execution group). Everything else carries no priority.
	switch {
	case nodeType.Is(types.NodeTypeContainer) && !inOrdered && containsOrdered(args):
	case nodeType.Is(types.NodeTypeIt) && !inOrdered:
	default:
		return text, args, nil
	}

	ms, ok := index.priceGroup(append(ancestors[:len(ancestors):len(ancestors)], text))
	total.Add(1)
	if ok {
		matched.Add(1)
	}
	return text, append(args, ginkgo.SpecPriority(ms)), nil
}

// load reads and indexes the stats file exactly once. It is safe (and cheap) to
// call on every node; the env var is read at first node construction, which works
// for both top-level nodes (package init) and nested nodes (BuildTree).
func load() {
	loadOnce.Do(func() {
		path := os.Getenv(EnvStatsFile)
		if path == "" {
			return
		}

		f, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "specpriority: cannot open %s: %v\n", path, err)
			return
		}
		defer f.Close()

		var sf StatsFile
		if err := json.NewDecoder(f).Decode(&sf); err != nil {
			fmt.Fprintf(os.Stderr, "specpriority: cannot parse %s: %v\n", path, err)
			return
		}

		index = BuildIndex(&sf, os.Getenv(EnvSpreadSeed))
		loadedSpecs = len(sf.Specs)
	})
}

// ancestorsAndOrdered returns the parent container hierarchy and whether the node
// is inside an Ordered container. CurrentTreeConstructionNodeReport panics for
// top-level nodes, which are created at package-init time before tree construction
// begins; a top-level node has no ancestors, so recovering to (nil, false) is the
// correct answer, not merely a fallback.
func ancestorsAndOrdered() (ancestors []string, inOrdered bool) {
	defer func() { _ = recover() }()

	r := ginkgo.CurrentTreeConstructionNodeReport()
	return r.ContainerHierarchyTexts, r.IsInOrderedContainer
}

// containsOrdered reports whether args carry the Ordered decorator. By the time a
// transformer runs, ginkgo has already unrolled any decorator slices, so a direct
// scan is sufficient.
func containsOrdered(args []any) bool {
	for _, a := range args {
		if a == ginkgo.Ordered {
			return true
		}
	}
	return false
}

func hasExplicitPriority(args []any) bool {
	for _, a := range args {
		if _, ok := a.(ginkgo.SpecPriority); ok {
			return true
		}
	}
	return false
}

// Report a one-line summary on the primary process when the feature is active, so
// CI logs show how many execution groups matched the historical data. A low match
// rate signals spec renames have drifted from the stats — the only drift detector.
// Multiple ReportAfterSuite nodes are allowed; this is a no-op when no stats file
// was loaded.
var _ = ginkgo.ReportAfterSuite("specpriority", func(_ ginkgo.Report) {
	if index == nil {
		return
	}
	fmt.Fprintf(os.Stderr,
		"specpriority: active — %d specs loaded, matched %d/%d execution-group nodes\n",
		loadedSpecs, matched.Load(), total.Load())
})
