package util

import (
	"regexp"
	"strings"
)

var k8sCommentMetadata = regexp.MustCompile(`\n\+([^=].*)(=([^=].*))?`)

func PostProcessVendoredComments(commentMap map[string]string) {
	for name, _ := range commentMap {
		if !strings.Contains(name, "/vendor/") {
			continue
		}

		parts := strings.Split(name, "/vendor/")
		if len(parts) > 1 {
			commentMap[parts[1]] = commentMap[name]
			delete(commentMap, name)
		}
	}
}

func PostProcessK8sComments(commentMap map[string]string) {
	for name, comment := range commentMap {
		commentMap[name] = k8sCommentMetadata.ReplaceAllString(comment, "")
	}
}
