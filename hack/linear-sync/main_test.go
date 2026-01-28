package main

import (
	"bytes"
	"context"
	"flag"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	pullrequests "github.com/loft-sh/changelog/pull-requests"
	"github.com/loft-sh/changelog/releases"
	"github.com/shurcooL/githubv4"
)

func TestStrictFilteringFlag(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		expectedValue bool
		description   string
	}{
		{
			name:          "Default strict filtering (true)",
			args:          []string{"linear-sync", "--release-tag", "v1.0.0"},
			expectedValue: true,
			description:   "Default should be strict filtering enabled",
		},
		{
			name:          "Explicit strict filtering true",
			args:          []string{"linear-sync", "--release-tag", "v1.0.0", "--strict-filtering=true"},
			expectedValue: true,
			description:   "Explicitly setting strict filtering to true",
		},
		{
			name:          "Explicit strict filtering false",
			args:          []string{"linear-sync", "--release-tag", "v1.0.0", "--strict-filtering=false"},
			expectedValue: false,
			description:   "Explicitly setting strict filtering to false",
		},
		{
			name:          "Explicit strict filtering false with equals",
			args:          []string{"linear-sync", "--release-tag", "v1.0.0", "--strict-filtering=false"},
			expectedValue: false,
			description:   "Using equals form for boolean flag",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse flags to test the strict-filtering flag
			flagset := flag.NewFlagSet("test", flag.ContinueOnError)
			flagset.SetOutput(io.Discard) // Suppress flag parsing output

			var (
				releaseTag      = flagset.String("release-tag", "", "The tag of the new release")
				strictFiltering = flagset.Bool("strict-filtering", true, "Only include PRs that were actually merged before the release was published")
			)

			err := flagset.Parse(tc.args[1:])
			if err != nil {
				t.Fatalf("Failed to parse flags: %v", err)
			}

			if *strictFiltering != tc.expectedValue {
				t.Errorf("%s: expected strict-filtering=%v, got=%v", tc.description, tc.expectedValue, *strictFiltering)
			}

			// Verify release-tag is parsed correctly
			if *releaseTag != "v1.0.0" {
				t.Errorf("Expected release-tag to be v1.0.0, got %s", *releaseTag)
			}
		})
	}
}

func TestLinearSyncLogic_StrictFiltering(t *testing.T) {
	// This test simulates the core logic flow with strict filtering
	releaseTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	// Mock data
	allPRs := []pullrequests.PullRequest{
		{
			Number:   1,
			Body:     "Fix bug ENG-1234",
			Merged:   true,
			MergedAt: &githubv4.DateTime{Time: releaseTime.Add(-2 * time.Hour)}, // Before release
		},
		{
			Number:   2,
			Body:     "Add feature ENG-5678",
			Merged:   true,
			MergedAt: &githubv4.DateTime{Time: releaseTime.Add(1 * time.Hour)}, // After release
		},
		{
			Number:   3,
			Body:     "Update docs ENG-9012",
			Merged:   true,
			MergedAt: &githubv4.DateTime{Time: releaseTime.Add(-30 * time.Minute)}, // Before release
		},
	}

	currentRelease := releases.Release{
		PublishedAt: githubv4.DateTime{Time: releaseTime},
		TagName:     "v1.2.0",
	}

	testCases := []struct {
		name               string
		strictFiltering    bool
		expectedPRCount    int
		expectedIssueCount int
		description        string
	}{
		{
			name:               "With strict filtering",
			strictFiltering:    true,
			expectedPRCount:    2, // Only PRs 1 and 3 (merged before release)
			expectedIssueCount: 2, // ENG-1234 and ENG-9012
			description:        "Should filter out PRs merged after release",
		},
		{
			name:               "Without strict filtering",
			strictFiltering:    false,
			expectedPRCount:    3, // All PRs
			expectedIssueCount: 3, // All issues
			description:        "Should include all PRs between tags",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var pullRequests []LinearPullRequest

			if tc.strictFiltering {
				// Simulate filtered PRs (would come from FetchPRsForRelease)
				filteredPRs := filterPRsByTime(allPRs, currentRelease.PublishedAt.Time)
				pullRequests = NewLinearPullRequests(filteredPRs, nil)
			} else {
				// Use all PRs (original behavior)
				pullRequests = NewLinearPullRequests(allPRs, nil)
			}

			if len(pullRequests) != tc.expectedPRCount {
				t.Errorf("%s: expected %d PRs, got %d PRs", tc.description, tc.expectedPRCount, len(pullRequests))
			}

			// Extract issue IDs
			var releasedIssues []string
			for _, pr := range pullRequests {
				if issueIDs := pr.IssueIDs(); len(issueIDs) > 0 {
					releasedIssues = append(releasedIssues, issueIDs...)
				}
			}

			if len(releasedIssues) != tc.expectedIssueCount {
				t.Errorf("%s: expected %d issues, got %d issues", tc.description, tc.expectedIssueCount, len(releasedIssues))
			}
		})
	}
}

// Helper function to simulate the filtering logic
func filterPRsByTime(prs []pullrequests.PullRequest, releaseTime time.Time) []pullrequests.PullRequest {
	var filtered []pullrequests.PullRequest
	for _, pr := range prs {
		if pr.MergedAt != nil && pr.MergedAt.After(releaseTime) {
			continue
		}
		if pr.MergedAt != nil {
			filtered = append(filtered, pr)
		}
	}
	return filtered
}

func TestRunFunction_FlagValidation(t *testing.T) {
	testCases := []struct {
		name          string
		envVars       map[string]string
		args          []string
		expectError   bool
		expectedError string
		description   string
	}{
		{
			name: "Missing GitHub token",
			envVars: map[string]string{
				"LINEAR_TOKEN": "test-linear-token",
			},
			args:          []string{"linear-sync", "--release-tag", "v1.0.0"},
			expectError:   true,
			expectedError: "github token must be set",
			description:   "Should fail when GitHub token is missing",
		},
		{
			name: "Missing Linear token",
			envVars: map[string]string{
				"GITHUB_TOKEN": "test-github-token",
			},
			args:          []string{"linear-sync", "--release-tag", "v1.0.0"},
			expectError:   true,
			expectedError: "linear token must be set",
			description:   "Should fail when Linear token is missing",
		},
		{
			name: "Missing release tag",
			envVars: map[string]string{
				"GITHUB_TOKEN": "test-github-token",
				"LINEAR_TOKEN": "test-linear-token",
			},
			args:          []string{"linear-sync"},
			expectError:   true,
			expectedError: "release tag must be set",
			description:   "Should fail when release tag is missing",
		},
		{
			name: "All required parameters provided",
			envVars: map[string]string{
				"GITHUB_TOKEN": "test-github-token",
				"LINEAR_TOKEN": "test-linear-token",
			},
			args:        []string{"linear-sync", "--release-tag", "v1.0.0"},
			expectError: false,
			description: "Should succeed when all required parameters are provided",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tc.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			// Clear any existing env vars not in test case
			if _, exists := tc.envVars["GITHUB_TOKEN"]; !exists {
				os.Unsetenv("GITHUB_TOKEN")
			}
			if _, exists := tc.envVars["LINEAR_TOKEN"]; !exists {
				os.Unsetenv("LINEAR_TOKEN")
			}

			var stderr bytes.Buffer
			err := run(context.Background(), &stderr, tc.args)

			if tc.expectError {
				if err == nil {
					t.Errorf("%s: expected error but got none", tc.description)
				} else if !strings.Contains(err.Error(), tc.expectedError) {
					t.Errorf("%s: expected error containing '%s', got '%s'", tc.description, tc.expectedError, err.Error())
				}
			} else {
				if err != nil {
					// For successful cases, we expect to fail later in the process (API calls)
					// but not during initial validation
					if strings.Contains(err.Error(), "github token must be set") ||
						strings.Contains(err.Error(), "linear token must be set") ||
						strings.Contains(err.Error(), "release tag must be set") {
						t.Errorf("%s: unexpected validation error: %s", tc.description, err.Error())
					}
					// Other errors (like API failures) are expected in this test environment
				}
			}
		})
	}
}

func TestDeduplicateIssueIDs(t *testing.T) {
	testCases := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "no duplicates",
			input:    []string{"eng-1234", "eng-5678", "eng-9012"},
			expected: []string{"eng-1234", "eng-5678", "eng-9012"},
		},
		{
			name:     "with duplicates within single PR (body + branch)",
			input:    []string{"eng-8061", "eng-8061"},
			expected: []string{"eng-8061"},
		},
		{
			name:     "with duplicates across multiple PRs",
			input:    []string{"eng-1234", "eng-5678", "eng-1234", "eng-9012", "eng-5678"},
			expected: []string{"eng-1234", "eng-5678", "eng-9012"},
		},
		{
			name:     "empty list",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "all duplicates",
			input:    []string{"eng-1234", "eng-1234", "eng-1234"},
			expected: []string{"eng-1234"},
		},
		{
			name:     "preserves order",
			input:    []string{"eng-3333", "eng-1111", "eng-2222", "eng-1111"},
			expected: []string{"eng-3333", "eng-1111", "eng-2222"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := deduplicateIssueIDs(tc.input)

			if len(result) != len(tc.expected) {
				t.Errorf("expected %d items, got %d", len(tc.expected), len(result))
				return
			}

			for i, v := range result {
				if v != tc.expected[i] {
					t.Errorf("at index %d: expected %q, got %q", i, tc.expected[i], v)
				}
			}
		})
	}
}

func TestTeamKeyFiltering(t *testing.T) {
	// Test that issue IDs are filtered by valid team keys
	validKeys := ValidTeamKeys{
		"eng":    {},
		"doc":    {},
		"devops": {},
	}

	testCases := []struct {
		name           string
		prBody         string
		prBranch       string
		validTeamKeys  ValidTeamKeys
		expectedIssues []string
		description    string
	}{
		{
			name:           "Filter out invalid team keys",
			prBody:         "Fixes ENG-1234 and pr-3354",
			prBranch:       "feature/update",
			validTeamKeys:  validKeys,
			expectedIssues: []string{"eng-1234"},
			description:    "Should filter out pr-3354 as 'pr' is not a valid team key",
		},
		{
			name:           "Filter out multiple invalid patterns",
			prBody:         "Fixes snap-1, ENG-5678, and build-123",
			prBranch:       "feature/update",
			validTeamKeys:  validKeys,
			expectedIssues: []string{"eng-5678"},
			description:    "Should filter out snap-1 and build-123",
		},
		{
			name:           "Allow all valid team keys",
			prBody:         "Fixes ENG-1234, DOC-567, and DEVOPS-890",
			prBranch:       "feature/update",
			validTeamKeys:  validKeys,
			expectedIssues: []string{"eng-1234", "doc-567", "devops-890"},
			description:    "Should allow all issues with valid team keys",
		},
		{
			name:           "Case insensitive team keys",
			prBody:         "Fixes eng-1234 and ENG-5678",
			prBranch:       "DOC-999/update",
			validTeamKeys:  validKeys,
			expectedIssues: []string{"eng-1234", "eng-5678", "doc-999"},
			description:    "Should match team keys case-insensitively",
		},
		{
			name:           "No filtering when validTeamKeys is nil",
			prBody:         "Fixes pr-3354 and snap-1",
			prBranch:       "feature/update",
			validTeamKeys:  nil,
			expectedIssues: []string{"pr-3354", "snap-1"},
			description:    "Should not filter when validTeamKeys is nil",
		},
		{
			name:           "Empty validTeamKeys filters everything",
			prBody:         "Fixes ENG-1234",
			prBranch:       "feature/update",
			validTeamKeys:  ValidTeamKeys{},
			expectedIssues: []string{},
			description:    "Should filter all issues when validTeamKeys is empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pr := LinearPullRequest{
				PullRequest: pullrequests.PullRequest{
					Number:      1,
					Body:        tc.prBody,
					HeadRefName: tc.prBranch,
					Merged:      true,
				},
				validTeamKeys: tc.validTeamKeys,
			}

			extractedIssues := pr.IssueIDs()

			if len(extractedIssues) != len(tc.expectedIssues) {
				t.Errorf("%s: expected %d issues, got %d issues", tc.description, len(tc.expectedIssues), len(extractedIssues))
				t.Errorf("Expected: %v, Got: %v", tc.expectedIssues, extractedIssues)
				return
			}

			// Check that all expected issues are present
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

			for issue := range expectedMap {
				t.Errorf("%s: expected issue ID not found: %s", tc.description, issue)
			}
		})
	}
}

func TestExtractTeamKey(t *testing.T) {
	testCases := []struct {
		issueID     string
		expectedKey string
	}{
		{"eng-1234", "eng"},
		{"DOC-567", "doc"},
		{"DEVOPS-890", "devops"},
		{"pr-3354", "pr"},
		{"a-1", "a"},
		{"", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.issueID, func(t *testing.T) {
			result := extractTeamKey(tc.issueID)
			if result != tc.expectedKey {
				t.Errorf("extractTeamKey(%q) = %q, want %q", tc.issueID, result, tc.expectedKey)
			}
		})
	}
}

func TestFlagDescriptions(t *testing.T) {
	// Test that all flags have proper descriptions
	flagset := flag.NewFlagSet("test", flag.ContinueOnError)
	var buf bytes.Buffer
	flagset.SetOutput(&buf)

	// Define flags as in main function
	flagset.String("owner", "loft-sh", "The GitHub owner of the repository")
	flagset.String("repo", "vcluster", "The GitHub repository to generate the changelog for")
	flagset.String("token", "", "The GitHub token to use for authentication")
	flagset.String("previous-tag", "", "The previous tag to generate the changelog for (if not set, the last stable release will be used)")
	flagset.String("release-tag", "", "The tag of the new release")
	flagset.Bool("debug", false, "Enable debug logging")
	flagset.String("linear-token", "", "The Linear token to use for authentication")
	flagset.String("released-state-name", "Released", "The name of the state to use for the released state")
	flagset.String("ready-for-release-state-name", "Ready for Release", "The name of the state that indicates an issue is ready to be released")
	flagset.String("linear-team-name", "vCluster / Platform", "The name of the team to use for the linear team")
	flagset.Bool("dry-run", false, "Do not actually move issues to the released state")
	strictFiltering := flagset.Bool("strict-filtering", true, "Only include PRs that were actually merged before the release was published (recommended to avoid false positives)")

	// Test the new flag specifically
	if *strictFiltering != true {
		t.Error("strict-filtering flag should default to true")
	}

	// Generate help output
	flagset.Usage()
	helpOutput := buf.String()

	// Check that our new flag appears in help
	if !strings.Contains(helpOutput, "strict-filtering") {
		t.Error("Help output should contain strict-filtering flag")
	}

	if !strings.Contains(helpOutput, "recommended to avoid false positives") {
		t.Error("Help output should contain explanation about false positives")
	}
}
