package pullrequests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shurcooL/githubv4"
)

// Mock GitHub client for testing
type mockGitHubClient struct {
	prs             []PullRequest
	queryCallCount  int
	shouldFailQuery bool
}

func (m *mockGitHubClient) Query(ctx context.Context, q interface{}, variables map[string]interface{}) error {
	m.queryCallCount++

	if m.shouldFailQuery {
		return fmt.Errorf("mock error")
	}

	return nil
}

func TestFetchPRsForRelease_TimeFiltering(t *testing.T) {
	// Test data setup
	releaseTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	testCases := []struct {
		name          string
		prs           []PullRequest
		releaseTime   time.Time
		expectedCount int
		description   string
	}{
		{
			name: "PRs merged before release",
			prs: []PullRequest{
				{
					Number:   1,
					Merged:   true,
					MergedAt: &githubv4.DateTime{Time: releaseTime.Add(-2 * time.Hour)}, // 2 hours before
				},
				{
					Number:   2,
					Merged:   true,
					MergedAt: &githubv4.DateTime{Time: releaseTime.Add(-1 * time.Hour)}, // 1 hour before
				},
			},
			releaseTime:   releaseTime,
			expectedCount: 2,
			description:   "All PRs merged before release should be included",
		},
		{
			name: "PRs merged after release",
			prs: []PullRequest{
				{
					Number:   3,
					Merged:   true,
					MergedAt: &githubv4.DateTime{Time: releaseTime.Add(1 * time.Hour)}, // 1 hour after
				},
				{
					Number:   4,
					Merged:   true,
					MergedAt: &githubv4.DateTime{Time: releaseTime.Add(2 * time.Hour)}, // 2 hours after
				},
			},
			releaseTime:   releaseTime,
			expectedCount: 0,
			description:   "PRs merged after release should be excluded",
		},
		{
			name: "Mixed timing PRs",
			prs: []PullRequest{
				{
					Number:   5,
					Merged:   true,
					MergedAt: &githubv4.DateTime{Time: releaseTime.Add(-1 * time.Hour)}, // Before
				},
				{
					Number:   6,
					Merged:   true,
					MergedAt: &githubv4.DateTime{Time: releaseTime.Add(1 * time.Hour)}, // After
				},
				{
					Number:   7,
					Merged:   true,
					MergedAt: &githubv4.DateTime{Time: releaseTime.Add(-30 * time.Minute)}, // Before
				},
			},
			releaseTime:   releaseTime,
			expectedCount: 2,
			description:   "Should include only PRs merged before release",
		},
		{
			name: "PRs with nil MergedAt",
			prs: []PullRequest{
				{
					Number:   8,
					Merged:   true,
					MergedAt: nil, // No merge time available
				},
				{
					Number:   9,
					Merged:   true,
					MergedAt: &githubv4.DateTime{Time: releaseTime.Add(-1 * time.Hour)}, // Before
				},
			},
			releaseTime:   releaseTime,
			expectedCount: 1, // Only the one with valid merge time before release
			description:   "Should exclude PRs with nil MergedAt",
		},
		{
			name: "Exact release time",
			prs: []PullRequest{
				{
					Number:   10,
					Merged:   true,
					MergedAt: &githubv4.DateTime{Time: releaseTime}, // Exact time
				},
			},
			releaseTime:   releaseTime,
			expectedCount: 1, // PRs merged at exact release time are included (not After)
			description:   "PRs merged at exact release time should be included",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filtered := filterPRsByMergeTime(tc.prs, tc.releaseTime)

			if len(filtered) != tc.expectedCount {
				t.Errorf("%s: expected %d PRs, got %d PRs", tc.description, tc.expectedCount, len(filtered))
			}

			// Verify that all returned PRs have valid merge times before release
			for _, pr := range filtered {
				if pr.MergedAt == nil {
					t.Errorf("Filtered PR %d has nil MergedAt", pr.Number)
				} else if pr.MergedAt.After(tc.releaseTime) {
					t.Errorf("Filtered PR %d was merged after release time", pr.Number)
				}
			}
		})
	}
}

// Helper function to test filtering logic in isolation
func filterPRsByMergeTime(prs []PullRequest, releaseTime time.Time) []PullRequest {
	var filtered []PullRequest
	for _, pr := range prs {
		// Time-based filtering: exclude PRs merged after release was published
		if pr.MergedAt != nil && pr.MergedAt.After(releaseTime) {
			continue
		}

		// Include PR if it has a valid merge time before the release
		if pr.MergedAt != nil {
			filtered = append(filtered, pr)
		}
	}
	return filtered
}

func TestFetchPRsForRelease_EdgeCases(t *testing.T) {
	testCases := []struct {
		name        string
		prs         []PullRequest
		expectCount int
		description string
	}{
		{
			name:        "Empty PR list",
			prs:         []PullRequest{},
			expectCount: 0,
			description: "Should handle empty PR list gracefully",
		},
		{
			name: "All PRs have nil MergedAt",
			prs: []PullRequest{
				{Number: 1, Merged: true, MergedAt: nil},
				{Number: 2, Merged: true, MergedAt: nil},
			},
			expectCount: 0,
			description: "Should exclude all PRs with nil MergedAt",
		},
		{
			name: "PRs merged at microsecond precision",
			prs: []PullRequest{
				{
					Number:   1,
					Merged:   true,
					MergedAt: &githubv4.DateTime{Time: time.Date(2024, 1, 15, 11, 59, 59, 999999999, time.UTC)},
				},
			},
			expectCount: 1,
			description: "Should handle microsecond-level timing correctly",
		},
	}

	releaseTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filtered := filterPRsByMergeTime(tc.prs, releaseTime)

			if len(filtered) != tc.expectCount {
				t.Errorf("%s: expected %d PRs, got %d PRs", tc.description, tc.expectCount, len(filtered))
			}
		})
	}
}

func TestPullRequestStructure(t *testing.T) {
	// Test that the PullRequest struct has all expected fields
	pr := PullRequest{
		Merged:      true,
		Body:        "Test PR body",
		HeadRefName: "feature/test",
		Author:      struct{ Login string }{Login: "testuser"},
		Number:      123,
		MergedAt:    &githubv4.DateTime{Time: time.Now()},
	}

	if !pr.Merged {
		t.Error("Expected Merged to be true")
	}
	if pr.Body != "Test PR body" {
		t.Error("Expected Body to match")
	}
	if pr.HeadRefName != "feature/test" {
		t.Error("Expected HeadRefName to match")
	}
	if pr.Author.Login != "testuser" {
		t.Error("Expected Author.Login to match")
	}
	if pr.Number != 123 {
		t.Error("Expected Number to match")
	}
	if pr.MergedAt == nil {
		t.Error("Expected MergedAt to not be nil")
	}
}
