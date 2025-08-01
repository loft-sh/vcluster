package patch

import (
	"fmt"
	"strings"
)

func parsePath(path string) ([]string, error) {
	return parsePathWithIndexing(path, false)
}

// parsePathWithIndexing parses a given json path into different segments
// which can be used to navigate an object. If preserveIndexNotation is true,
// it will keep the index notation (e.g. [0]) in the segments, otherwise
// it will split the segments at the index notation and return only the key names.
func parsePathWithIndexing(path string, preserveIndexNotation bool) ([]string, error) {
	path = strings.TrimSpace(path)
	retSegments := []string{}

	curSegment := []byte{}
	bracketOpen := false
	quoteOpen := false
	for i, v := range path {
		if v == '"' {
			quoteOpen = !quoteOpen
		} else if !quoteOpen && !bracketOpen && v == '.' {
			if len(curSegment) == 0 {
				continue
			}

			retSegments = append(retSegments, string(curSegment))
			curSegment = []byte{}
		} else if !quoteOpen && v == '[' {
			if bracketOpen {
				return nil, fmt.Errorf("unexpected bracket in bracket in %s at %d", path, i)
			}

			bracketOpen = true
			if len(curSegment) > 0 {
				retSegments = append(retSegments, string(curSegment))
			}
			curSegment = []byte{}
		} else if !quoteOpen && v == ']' {
			if len(curSegment) == 0 {
				return nil, fmt.Errorf("unexpected empty segment in %s at %d", path, i)
			} else if !bracketOpen {
				return nil, fmt.Errorf("unexpected bracket close in bracket in %s at %d", path, i)
			}

			bracketOpen = false
			retSegment := string(curSegment)
			if preserveIndexNotation {
				retSegment = fmt.Sprintf("[%s]", retSegment)
			}
			retSegments = append(retSegments, retSegment)
			curSegment = []byte{}
		} else {
			curSegment = append(curSegment, byte(v))
		}
	}
	if len(curSegment) > 0 {
		retSegments = append(retSegments, string(curSegment))
	}
	if quoteOpen {
		return nil, fmt.Errorf("unclosed quote in path")
	}
	if bracketOpen {
		return nil, fmt.Errorf("unclosed bracket in path")
	}

	return retSegments, nil
}
