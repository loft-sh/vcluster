package regex

import (
	"regexp"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/types"
)

const (
	RegexNameGroup      = "NAME"
	RegexNamespaceGroup = "NAMESPACE"
)

func PrepareRegex(regex string) (*regexp.Regexp, error) {
	re := strings.TrimSpace(regex)
	// replacement order is critical, replace $NAMESPACE first
	re = strings.ReplaceAll(re, "$NAMESPACE", "(?P<NAMESPACE>[a-z0-9](?:[-a-z0-9]*[a-z0-9])?)")
	re = strings.ReplaceAll(re, "$NAME", "(?P<NAME>[a-z](?:[-a-z0-9]*[a-z0-9])?)")
	return regexp.Compile(re)
}

type TranslateFunc func(name, namespace string) types.NamespacedName

func ProcessRegex(regex *regexp.Regexp, input string, translateFunc TranslateFunc) string {
	// Get group number of the named NAME and NAMESPACE regex groups
	namePos := -1
	namespacePos := -1
	groupNames := regex.SubexpNames()
	for pos, gn := range groupNames {
		if gn == RegexNameGroup {
			namePos = pos
		} else if gn == RegexNamespaceGroup {
			namespacePos = pos
		}
	}

	// Find indexes of all matches and create a list of replacements from it
	replacements := []IndexBasedReplaceItem{}
	allIndexes := regex.FindAllStringSubmatchIndex(input, -1)
	for _, indexes := range allIndexes {
		if namePos != -1 && indexes[2*namePos] != -1 && indexes[2*namePos+1] != -1 {
			name := input[indexes[2*namePos]:indexes[2*namePos+1]]
			namespace := ""
			if namespacePos != -1 && indexes[2*namespacePos] != -1 && indexes[2*namespacePos+1] != -1 {
				// get the NAMESPACE value for use in the name translation
				namespace = input[indexes[2*namespacePos]:indexes[2*namespacePos+1]]
			}

			translatedName := translateFunc(name, namespace)

			replacements = append(replacements, IndexBasedReplaceItem{
				StartIndex:  indexes[2*namePos],
				EndIndex:    indexes[2*namePos+1],
				Replacement: translatedName.Name,
			})

			if namespacePos != -1 && indexes[2*namespacePos] != -1 && indexes[2*namespacePos+1] != -1 {
				replacements = append(replacements, IndexBasedReplaceItem{
					StartIndex:  indexes[2*namespacePos],
					EndIndex:    indexes[2*namespacePos+1],
					Replacement: translatedName.Namespace,
				})
			}
		}
	}

	return IndexBasedReplace(input, replacements)
}

type IndexBasedReplaceItem struct {
	StartIndex  int
	EndIndex    int
	Replacement string
}

// IndexBasedReplace replaces multiple substrings in the input string
// with the replacement values based on the indexes in the original input string.
// input - string that will have parts of it replaced
// items - slice of IndexBasedRelaceItem(s). Only nonoverlaping index pairs are supported.
func IndexBasedReplace(input string, items []IndexBasedReplaceItem) string {
	// sort thereplace items because otherwise indexOffset could cause
	// panic due to out of bound access
	sort.Slice(items, func(i, j int) bool {
		return items[i].StartIndex < items[j].StartIndex
	})

	s := input
	indexOffset := 0
	for _, i := range items {
		s = s[:i.StartIndex+indexOffset] + i.Replacement + s[i.EndIndex+indexOffset:]
		// track how replacements affects indexes based on the difference between
		// the length of the original substring and its replacement
		indexOffset = indexOffset + len(i.Replacement) - (i.EndIndex - i.StartIndex)
	}
	return s
}
