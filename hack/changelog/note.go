package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	pullrequests "github.com/loft-sh/changelog/pull-requests"
)

var notesInBodyREs = []*regexp.Regexp{
	regexp.MustCompile("(?ms)^```release-note[s]?:(?P<type>[^\r\n]*)\r?\n?(?P<note>.*?)\r?\n?```"),
	regexp.MustCompile("(?ms)^```release-note[s]?\r?\ntype:\\s?(?P<type>[^\r\n]*)\r?\nnote:\\s?(?P<note>.*?)\r?\n?```"),
}

type Note struct {
	Type   string
	Author string
	Body   string
	PR     int
}

func (n Note) String() string {
	return fmt.Sprintf("- %s: %s (by @%v in #%d)\n", n.Type, n.Body, n.Author, n.PR)
}

func NewNotesFromPullRequest(p pullrequests.PullRequest) []Note {
	return NewNotes(p.Body, p.Author.Login, p.Number)
}

func SortNotes(res []Note) func(i, j int) bool {
	return func(i, j int) bool {
		if res[i].Type < res[j].Type {
			return true
		} else if res[j].Type < res[i].Type {
			return false
		} else if res[i].Body < res[j].Body {
			return true
		} else if res[j].Body < res[i].Body {
			return false
		} else if res[i].PR < res[j].PR {
			return true
		} else if res[j].PR < res[i].PR {
			return false
		}
		return false
	}
}

func NewNotes(body, author string, number int) []Note {
	var res []Note
	for _, re := range notesInBodyREs {
		matches := re.FindAllStringSubmatch(body, -1)
		if len(matches) == 0 {
			continue
		}

		for _, match := range matches {
			note := ""
			typ := ""

			for i, name := range re.SubexpNames() {
				switch name {
				case "note":
					note = match[i]
				case "type":
					typ = strings.ToLower(match[i])
				}
				if note != "" && typ != "" {
					break
				}
			}

			note = strings.TrimSpace(note)
			typ = strings.TrimSpace(typ)

			if typ == "type" || typ == "none" || note == "none" {
				note = ""
				typ = ""
			}
			if note == "" && typ == "" {
				continue
			}

			res = append(res, Note{
				Type:   typ,
				Body:   note,
				PR:     number,
				Author: author,
			})
		}
	}
	sort.Slice(res, SortNotes(res))

	return res
}
