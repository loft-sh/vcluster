package main

import (
	"testing"
	"time"

	pullrequests "github.com/loft-sh/changelog/pull-requests"
	"github.com/shurcooL/githubv4"
)

func TestLinearPullRequestIssueExtraction(t *testing.T) {
	// Test data with different issue ID patterns
	testCases := []struct {
		name           string
		prBody         string
		prBranch       string
		expectedIssues []string
		description    string
	}{
		{
			name:           "Single issue in body",
			prBody:         "This PR fixes ENG-1234",
			prBranch:       "feature/some-feature",
			expectedIssues: []string{"eng-1234"},
			description:    "Should extract single issue ID from PR body",
		},
		{
			name:           "Multiple issues in body",
			prBody:         "This PR addresses ENG-1234 and also fixes ENG-5678",
			prBranch:       "feature/multi-fix",
			expectedIssues: []string{"eng-1234", "eng-5678"},
			description:    "Should extract multiple issue IDs from PR body",
		},
		{
			name:           "Issue in branch name",
			prBody:         "Update documentation",
			prBranch:       "ENG-9012/update-docs",
			expectedIssues: []string{"eng-9012"},
			description:    "Should extract issue ID from branch name",
		},
		{
			name:           "Issue in both body and branch",
			prBody:         "Fix bug ENG-1111",
			prBranch:       "ENG-2222/fix-implementation",
			expectedIssues: []string{"eng-1111", "eng-2222"},
			description:    "Should extract issue IDs from both body and branch",
		},
		{
			name:           "No issues found",
			prBody:         "Simple documentation update",
			prBranch:       "feature/docs-update",
			expectedIssues: []string{},
			description:    "Should return empty list when no issues found",
		},
		{
			name:           "Different issue patterns",
			prBody:         "Fixes ABC-1234, DEF-5678, and GHI-9012",
			prBranch:       "feature/multiple-patterns",
			expectedIssues: []string{"abc-1234", "def-5678", "ghi-9012"},
			description:    "Should handle different three-letter prefixes",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pr := pullrequests.PullRequest{
				Number:      1,
				Body:        tc.prBody,
				HeadRefName: tc.prBranch,
				Merged:      true,
			}

			linearPR := LinearPullRequest{PullRequest: pr, validTeamKeys: nil}
			extractedIssues := linearPR.IssueIDs()

			if len(extractedIssues) != len(tc.expectedIssues) {
				t.Errorf("%s: expected %d issues, got %d issues", tc.description, len(tc.expectedIssues), len(extractedIssues))
				t.Errorf("Expected: %v, Got: %v", tc.expectedIssues, extractedIssues)
				return
			}

			// Check that all expected issues are present (order doesn't matter)
			expectedMap := make(map[string]bool)
			for _, issue := range tc.expectedIssues {
				expectedMap[issue] = true
			}

			for _, issue := range extractedIssues {
				if !expectedMap[issue] {
					t.Errorf("%s: unexpected issue ID found: %s", tc.description, issue)
				}
				delete(expectedMap, issue)
			}

			// Check for any missing issues
			for issue := range expectedMap {
				t.Errorf("%s: expected issue ID not found: %s", tc.description, issue)
			}
		})
	}
}

func TestStrictFilteringIntegration(t *testing.T) {
	// Simulate the complete flow with mock data
	releaseTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	allPRs := []pullrequests.PullRequest{
		{
			Number:      1,
			Body:        "Fix critical bug ENG-1234",
			HeadRefName: "hotfix/critical-bug",
			Merged:      true,
			MergedAt:    &githubv4.DateTime{Time: releaseTime.Add(-2 * time.Hour)}, // Before release
		},
		{
			Number:      2,
			Body:        "Add new feature ENG-5678",
			HeadRefName: "feature/new-feature",
			Merged:      true,
			MergedAt:    &githubv4.DateTime{Time: releaseTime.Add(1 * time.Hour)}, // After release
		},
		{
			Number:      3,
			Body:        "Update config ENG-9012",
			HeadRefName: "ENG-9012/update-config",
			Merged:      true,
			MergedAt:    &githubv4.DateTime{Time: releaseTime.Add(-30 * time.Minute)}, // Before release
		},
		{
			Number:      4,
			Body:        "Documentation update",
			HeadRefName: "docs/update",
			Merged:      true,
			MergedAt:    &githubv4.DateTime{Time: releaseTime.Add(-1 * time.Hour)}, // Before release, no issue ID
		},
	}

	testCases := []struct {
		name            string
		strictFiltering bool
		expectedPRs     []int    // PR numbers that should be included
		expectedIssues  []string // Issue IDs that should be extracted
		description     string
	}{
		{
			name:            "Strict filtering enabled",
			strictFiltering: true,
			expectedPRs:     []int{1, 3, 4},                   // Only PRs merged before release
			expectedIssues:  []string{"eng-1234", "eng-9012"}, // PR 4 has no issue ID
			description:     "Should include only PRs merged before release",
		},
		{
			name:            "Strict filtering disabled",
			strictFiltering: false,
			expectedPRs:     []int{1, 2, 3, 4},                            // All PRs
			expectedIssues:  []string{"eng-1234", "eng-5678", "eng-9012"}, // All issue IDs
			description:     "Should include all PRs between tags",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var pullRequests []LinearPullRequest

			if tc.strictFiltering {
				// Apply time-based filtering
				var filteredPRs []pullrequests.PullRequest
				for _, pr := range allPRs {
					if pr.MergedAt != nil && pr.MergedAt.After(releaseTime) {
						continue
					}
					if pr.MergedAt != nil {
						filteredPRs = append(filteredPRs, pr)
					}
				}
				pullRequests = NewLinearPullRequests(filteredPRs, nil)
			} else {
				// Use all PRs
				pullRequests = NewLinearPullRequests(allPRs, nil)
			}

			// Verify correct PRs are included
			if len(pullRequests) != len(tc.expectedPRs) {
				t.Errorf("%s: expected %d PRs, got %d PRs", tc.description, len(tc.expectedPRs), len(pullRequests))
			}

			// Extract issue IDs
			issueSet := make(map[string]bool)
			for _, pr := range pullRequests {
				if issueIDs := pr.IssueIDs(); len(issueIDs) > 0 {
					for _, issueID := range issueIDs {
						issueSet[issueID] = true
					}
				}
			}

			// Convert set to slice for comparison
			var releasedIssues []string
			for issueID := range issueSet {
				releasedIssues = append(releasedIssues, issueID)
			}

			// Verify correct issues are extracted
			if len(releasedIssues) != len(tc.expectedIssues) {
				t.Errorf("%s: expected %d issues, got %d issues", tc.description, len(tc.expectedIssues), len(releasedIssues))
				t.Errorf("Expected: %v, Got: %v", tc.expectedIssues, releasedIssues)
				return
			}

			// Check that all expected issues are present
			expectedMap := make(map[string]bool)
			for _, issue := range tc.expectedIssues {
				expectedMap[issue] = true
			}

			for _, issue := range releasedIssues {
				if !expectedMap[issue] {
					t.Errorf("%s: unexpected issue ID found: %s", tc.description, issue)
				}
				delete(expectedMap, issue)
			}

			// Check for any missing issues
			for issue := range expectedMap {
				t.Errorf("%s: expected issue ID not found: %s", tc.description, issue)
			}
		})
	}
}

// Benchmark test to ensure performance is reasonable
func BenchmarkLinearPullRequestIssueExtraction(b *testing.B) {
	pr := pullrequests.PullRequest{
		Number:      1,
		Body:        "This PR fixes ENG-1234, ENG-5678, and addresses ENG-9012 along with ENG-3456",
		HeadRefName: "ENG-7890/complex-fix",
		Merged:      true,
	}

	linearPR := LinearPullRequest{PullRequest: pr, validTeamKeys: nil}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = linearPR.IssueIDs()
	}
}

func BenchmarkTimeBasedFiltering(b *testing.B) {
	releaseTime := time.Now()

	// Create test data
	prs := make([]pullrequests.PullRequest, 100)
	for i := 0; i < 100; i++ {
		prs[i] = pullrequests.PullRequest{
			Number:   i + 1,
			Body:     "Test PR body",
			Merged:   true,
			MergedAt: &githubv4.DateTime{Time: releaseTime.Add(time.Duration(i-50) * time.Hour)}, // Mix of before/after
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = filterPRsByTime(prs, releaseTime)
	}
}
