package patches

import (
	"regexp"
	"strings"
)

// OpPath is an yaml-jsonpath 'selector'. For more info see: https://github.com/vmware-labs/yaml-jsonpath
type OpPath string

var rootChildPath = regexp.MustCompile(`^(\$|\.)([^\.]+)$`)

func (p *OpPath) isRootChild() bool {
	path := string(*p)
	return rootChildPath.MatchString(path)
}

var endsWithBracketExp = regexp.MustCompile(`\[([^\]]+)\]$`)

func (p *OpPath) getChildName() string {
	path := string(*p)
	if path == "$" || path == "." {
		return ""
	}

	if endsWithBracketExp.MatchString(path) {
		matches := endsWithBracketExp.FindStringSubmatch(path)
		if len(matches) > 0 {
			match := matches[1]
			match = strings.ReplaceAll(match, `'`, "")
			match = strings.ReplaceAll(match, `"`, "")
			return match
		}
		return ""
	}

	tokens := regexp.MustCompile(`\$|\.`).Split(path, -1)
	if len(tokens) == 0 {
		return ""
	}

	return tokens[len(tokens)-1]
}

func (p *OpPath) getParentPath() string {
	path := string(*p)
	if endsWithBracketExp.MatchString(path) {
		return endsWithBracketExp.ReplaceAllString(path, "")
	}

	tokens := strings.Split(path, ".")
	return strings.Join(tokens[0:len(tokens)-1], ".")
}
