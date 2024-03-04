package util

import (
	"regexp"
)

func GetExistingDescription(pageContent string) string {
	match := regexp.MustCompile(`(?is)^---.*?---\s*(import.*?\n)*\s*(.*?)\s*\n###? .*$`).ReplaceAllString(pageContent, "$2")

	if match != pageContent && match != "" {
		return "\n" + match + "\n"
	}
	return ""
}

func GetSection(headlineText, pageContent string) string {
	regex := `(?is)^.*\s*\n###?\s+` + headlineText + `\s*(.*?)(\n+((##)|$).*)?$`
	match := regexp.MustCompile(regex).ReplaceAllString(pageContent, "$1")

	if match != pageContent && match != "" {
		return match
	}
	return ""
}

/*
func GetPartOfAutogenSection(headlineText, pageContent string) string {
	regex := "(?is)^.*" + AutoGenTagBegin + `\s*\n###?\s+` + headlineText + `.*?` + AutoGenTagEnd + `\s*(.*?)\s*` + AutoGenTagBegin + ".*$"
	match := regexp.MustCompile(regex).ReplaceAllString(pageContent, "$1")

	if match != pageContent && match != "" {
		return match + "\n\n"
	}
	return ""
}*/
