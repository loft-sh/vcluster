package loghelper

import (
	"regexp"
	"strings"

	"k8s.io/klog/v2"
)

var klogRegEx1 = regexp.MustCompile(`^[A-Z][0-9]{4} [0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}\s+[0-9]+\s([^]]+)] (.+)$`)

var structuredComponent = regexp.MustCompile(`^([a-zA-Z\-_]+)=`)

// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/logging.md
func PrintKlogLine(line string, args []interface{}) {
	if klogRegEx1.MatchString(line) {
		matches := klogRegEx1.FindStringSubmatch(line)
		args = append(args, "location", matches[1])
		line = matches[2]
	}

	// try to parse structured logging
	line, extraArgs := parseStructuredLogging(line)
	klog.InfoS(line, append(args, extraArgs...)...)
}

func parseStructuredLogging(line string) (string, []interface{}) {
	if len(line) == 0 {
		return line, nil
	}

	line = strings.TrimSpace(line)

	// parse message
	message, line := parseQuotedMessage(line, true)
	if line == "" && structuredComponent.MatchString(message) {
		line = message
		message = ""
	}

	// parse structured
	retArgs := []interface{}{}
	for line != "" {
		if !structuredComponent.MatchString(line) {
			break
		}

		matches := structuredComponent.FindStringSubmatch(line)
		name := matches[1]
		line = line[len(matches[1])+1:]
		if message == "" && name == "msg" {
			value, restOfLine := parseQuotedMessage(line, false)

			message = value
			line = restOfLine
		} else {
			retArgs = append(retArgs, name)
			value, restOfLine := parseQuotedMessage(line, false)
			retArgs = append(retArgs, value)

			line = restOfLine
		}
	}

	return message, retArgs
}

func parseQuotedMessage(line string, allowSpace bool) (string, string) {
	message := ""
	if strings.HasPrefix(line, `"`) {
		message = line[1:]
		if strings.HasPrefix(message, `"`) {
			message = ""
		} else {
			// find the next non \"
			baseIndex := 0
			for {
				nextIndex := strings.Index(message[baseIndex:], `"`)
				nextIndex += baseIndex

				// unclosed "
				if nextIndex == -1 {
					return line, ""
				} else if nextIndex > 0 && message[nextIndex-1] != '\\' {
					message = message[:nextIndex]
					break
				}

				baseIndex = nextIndex + 1
				if baseIndex >= len(message) {
					return line, ""
				}
			}
		}

		line = strings.TrimSpace(line[len(message)+2:])
	} else {
		if allowSpace {
			return strings.ReplaceAll(line, `\"`, `"`), ""
		}

		nextSpace := strings.Index(line, ` `)
		if nextSpace > 0 {
			return strings.ReplaceAll(line[:nextSpace], `\"`, `"`), line[nextSpace+1:]
		}
	}

	return strings.ReplaceAll(message, `\"`, `"`), line
}
