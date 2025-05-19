package main

import (
	"regexp"
	"strings"

	pullrequests "github.com/loft-sh/changelog/pull-requests"
)

var issuesInBodyREs = []*regexp.Regexp{
	regexp.MustCompile(`(?P<issue>\w{3}-\d{4})`),
}

const PageSize = 100

type LinearPullRequest struct {
	pullrequests.PullRequest
}

func NewLinearPullRequests(prs []pullrequests.PullRequest) []LinearPullRequest {
	linearPRs := make([]LinearPullRequest, 0, len(prs))

	for _, pr := range prs {
		linearPRs = append(linearPRs, LinearPullRequest{pr})
	}

	return linearPRs
}

// IssueIDs extracts the Linear issue IDs from either the pull requests body
// or it's branch name.
//
// Will return an empty string if it did not manage to find an issue.
func (p LinearPullRequest) IssueIDs() []string {
	issueIDs := []string{}

	for _, re := range issuesInBodyREs {
		for _, body := range []string{p.Body, p.HeadRefName} {
			matches := re.FindAllStringSubmatch(body, -1)
			if len(matches) == 0 {
				continue
			}

			for _, match := range matches {
				for i, name := range re.SubexpNames() {
					issueID := ""

					switch name {
					case "issue":
						issueID = strings.ToLower(match[i])
						issueID = strings.TrimSpace(issueID)
					}

					if strings.HasPrefix(strings.ToLower(issueID), "cve") {
						issueID = ""
					}

					if issueID != "" {
						issueIDs = append(issueIDs, issueID)
					}
				}
			}
		}
	}

	return issueIDs
}