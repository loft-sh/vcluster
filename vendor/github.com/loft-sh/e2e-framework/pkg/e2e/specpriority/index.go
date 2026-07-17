package specpriority

import (
	"hash/fnv"
	"math"
	"sort"
	"strings"
)

// keySep joins hierarchy parts into a map key. NUL never appears in spec text.
const keySep = "\x00"

// keyFor builds a stable lookup key from a hierarchy, dropping empty entries so the
// producer's container_hierarchy (no root, no empties) and ginkgo's construction-time
// ContainerHierarchyTexts (which carries a leading "" for the synthetic root) both
// normalize to the same key. This is the single shared key function, used when
// building the index and when looking up at construction time.
func keyFor(parts []string) string {
	filtered := parts[:0:0]
	for _, p := range parts {
		if p != "" {
			filtered = append(filtered, p)
		}
	}
	return strings.Join(filtered, keySep)
}

// Index holds the precomputed lookups derived from a StatsFile.
type Index struct {
	// price maps a group key to its P95 duration in ms. Every prefix of every
	// leaf's full path (hierarchy + leaf) is an entry: a lone It's full-path key
	// holds its own duration, and an Ordered container's key holds the summed
	// duration of its whole subtree, so a single lookup prices either kind of
	// execution group.
	price map[string]int
	// thresholdMillis is the 80th-percentile leaf duration in ms. Execution groups
	// with a raw priority below this value (the fastest ~80%) receive a spread
	// priority drawn from [1, thresholdMillis) instead of their raw duration,
	// spreading them throughout the run rather than concentrating them in a
	// low-priority tail that raises platform load just as the run ends. Capping the
	// spread strictly below thresholdMillis guarantees a spread draw can never
	// outrank a retained raw LPT priority (every retained priority is >=
	// thresholdMillis) — sizing the spread off maxKnownMillis instead let spread
	// draws for small, unmeasured specs bury the heaviest Ordered execution groups
	// behind noise. Zero means spread is disabled (no leaf durations).
	thresholdMillis int
	// spreadSeed salts the per-group spread hash. It MUST be identical across all
	// parallel ginkgo processes: each process builds the spec tree and sorts
	// execution groups by SpecPriority independently, and ginkgo dispatches work
	// by a shared counter index into each process's sorted list, so divergent
	// priorities would make some groups run on several workers and others never
	// run at all. A per-run (but cross-process-constant) seed keeps the ordering
	// consistent within a run while still varying the spread between runs so
	// flakes surface in different spec combinations.
	spreadSeed string
}

// BuildIndex precomputes the lookups from sf using the P95 metric. spreadSeed salts
// the spread hash and must be the same value in every parallel ginkgo process (see
// Index.spreadSeed).
func BuildIndex(sf *StatsFile, spreadSeed string) *Index {
	ix := &Index{
		price:      map[string]int{},
		spreadSeed: spreadSeed,
	}
	if sf == nil {
		return ix
	}

	// leafMillis collects only leaf durations so the 80th-percentile threshold is
	// computed over leaves, not container sums.
	leafMillis := make([]int, 0, len(sf.Specs))
	priceSum := map[string]float64{}
	for i := range sf.Specs {
		s := sf.Specs[i]
		v := s.P95Seconds
		if v < 0 {
			v = 0
		}

		full := make([]string, 0, len(s.ContainerHierarchy)+1)
		full = append(full, s.ContainerHierarchy...)
		full = append(full, s.Leaf)

		leafMillis = append(leafMillis, floorMillis(v))

		// Add this leaf's duration to every prefix key so an Ordered container at
		// any depth is priced by the sum of its subtree while the full-path key
		// prices the lone leaf itself.
		filtered := make([]string, 0, len(full))
		for _, p := range full {
			if p != "" {
				filtered = append(filtered, p)
			}
		}
		for j := 1; j <= len(filtered); j++ {
			priceSum[keyFor(filtered[:j])] += v
		}
	}

	for k, v := range priceSum {
		ix.price[k] = floorMillis(v)
	}

	sort.Ints(leafMillis)
	if len(leafMillis) > 0 {
		ix.thresholdMillis = leafMillis[len(leafMillis)*8/10]
	}

	return ix
}

// priceGroup prices an execution group identified by its full path parts (parent
// hierarchy plus the node's own text). ms is the priority in milliseconds to attach,
// and matched reports whether a historical row was found (used only for the CI
// match-rate signal). Groups below the spread threshold (the fastest ~80%) and
// unmatched groups receive a spread priority so they interleave throughout the run;
// the slowest ~20% keep their raw LPT priority.
func (ix *Index) priceGroup(parts []string) (ms int, matched bool) {
	key := keyFor(parts)
	if v, ok := ix.price[key]; ok {
		if ix.thresholdMillis > 0 && v < ix.thresholdMillis {
			return ix.spreadPriority(key), true
		}
		return v, true
	}
	return ix.spreadPriority(key), false
}

// spreadPriority returns a priority in [1, thresholdMillis) derived deterministically
// from the group key and the seed, used to scatter a group throughout the run
// instead of clustering it at one end. It is a pure function of (spreadSeed, key),
// so every parallel process computes the same value for the same group (required
// for consistent dispatch ordering) while a different seed reshuffles the spread
// between runs. The range is capped below thresholdMillis, not maxKnownMillis, so a
// spread draw can never outrank a retained raw LPT priority. When there is no known
// range to sample from (no leaf durations) it returns 1, leaving such groups at a
// uniform low priority.
func (ix *Index) spreadPriority(key string) int {
	spreadCap := ix.thresholdMillis - 1
	if spreadCap < 1 {
		return 1
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(ix.spreadSeed))
	_, _ = h.Write([]byte(keySep))
	_, _ = h.Write([]byte(key))
	return int(h.Sum64()%uint64(spreadCap)) + 1
}

// floorMillis converts seconds to whole milliseconds with a floor of 1, so a
// matched-but-known-fast group stays below the unmatched band yet never collides
// with the default-0 priority of non-target nodes.
func floorMillis(seconds float64) int {
	if seconds <= 0 {
		return 1
	}
	if ms := int(math.Round(seconds * 1000)); ms > 1 {
		return ms
	}
	return 1
}
