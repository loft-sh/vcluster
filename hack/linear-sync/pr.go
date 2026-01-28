package main

import (
	"regexp"
	"strings"

	pullrequests "github.com/loft-sh/changelog/pull-requests"
)

var issuesInBodyREs = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(?P<issue>[A-Z]{2,10}-\d{1,5})`),
}

const PageSize = 100

// ValidTeamKeys holds a set of known Linear team keys (lowercase) for filtering
type ValidTeamKeys map[string]struct{}

type LinearPullRequest struct {
	pullrequests.PullRequest
	validTeamKeys ValidTeamKeys
}

func NewLinearPullRequests(prs []pullrequests.PullRequest, validTeamKeys ValidTeamKeys) []LinearPullRequest {
	linearPRs := make([]LinearPullRequest, 0, len(prs))

	for _, pr := range prs {
		linearPRs = append(linearPRs, LinearPullRequest{
			PullRequest:   pr,
			validTeamKeys: validTeamKeys,
		})
	}

	return linearPRs
}

// IssueIDs extracts the Linear issue IDs from either the pull requests body
// or it's branch name.
//
// Returns only issue IDs that match known Linear team keys (e.g., ENG-1234, DOC-567).
// Filters out false positives like pr-3354, snap-1 that match the regex pattern
// but aren't actual Linear issues.
//
// Will return an empty slice if no valid issues are found.
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

					// Filter by valid team keys if provided
					if issueID != "" && p.validTeamKeys != nil {
						teamKey := extractTeamKey(issueID)
						if _, valid := p.validTeamKeys[teamKey]; !valid {
							issueID = ""
						}
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

// extractTeamKey extracts the team key from an issue ID (e.g., "eng" from "eng-1234")
func extractTeamKey(issueID string) string {
	parts := strings.SplitN(issueID, "-", 2)
	if len(parts) < 1 {
		return ""
	}
	return strings.ToLower(parts[0])
}
