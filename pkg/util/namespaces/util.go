package namespaces

import "strings"

const (
	// Name placeholder will be replaced with this virtual cluster name
	NamePlaceholder string = "${name}"

	// WildcardChar is used in pattern mappings.
	WildcardChar string = "*"
)

// IsPattern checks if a string contains a wildcard character '*'.
func IsPattern(val string) bool {
	return strings.Contains(val, WildcardChar)
}

// MatchAndExtractWildcard checks if a given name matches a pattern that contains a single wildcard.
// It returns the string captured by the wildcard and a boolean indicating if the match was successful.
// If the provided pattern string does not contain a wildcard, it's not considered a pattern by this function,
// and it will return matched = false.
func MatchAndExtractWildcard(name, pattern string) (wildcardValue string, matched bool) {
	if !IsPattern(pattern) {
		return "", false
	}

	parts := strings.SplitN(pattern, WildcardChar, 2)
	prefix := parts[0]
	suffix := parts[1]

	if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, suffix) && len(name) >= (len(prefix)+len(suffix)) {
		wildcardValue = name[len(prefix) : len(name)-len(suffix)]
		return wildcardValue, true
	}

	return "", false
}

// ProcessNamespaceName returns namespace name after applying all pre-processing to it
func ProcessNamespaceName(namespaceName string, vclusterName string) string {
	return strings.ReplaceAll(namespaceName, NamePlaceholder, vclusterName)
}
