package orderedmap

import (
	"sort"
)

type pair struct {
	Key   string
	Value float32
}

// OrderedMap is a slice of pairs type with string keys and float values.
// It implement sorting methods by values.
type OrderedMap []pair

// Len return length of a given OrderedMap
func (p OrderedMap) Len() int { return len(p) }

// Less return if a element of an OrderedMap is smaller than another
func (p OrderedMap) Less(i, j int) bool { return p[i].Value < p[j].Value }

// Swap two members of an OrderedMap
func (p OrderedMap) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

// ToArray export keys of an OrderedMap into a slice
func (p OrderedMap) ToArray() []string {
	mapSize := p.Len()
	arr := make([]string, mapSize)
	for i, elem := range p {
		arr[i] = elem.Key
	}

	return arr
}

// SortByValues sort by values an OrderedMap in decreasing order
func (p OrderedMap) SortByValues() {
	sort.Sort(sort.Reverse(p))
}
