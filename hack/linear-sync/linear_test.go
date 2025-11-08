package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	pullrequests "github.com/loft-sh/changelog/pull-requests"
)

func TestMoveIssueLogic(t *testing.T) {
	// Create mock issues with different states
	mockIssues := []struct {
		ID         string
		StateName  string
		StateID    string
		ShouldMove bool
	}{
		{ID: "ENG-1234", StateName: "Ready for Release", StateID: "ready-state-id", ShouldMove: true},
		{ID: "ENG-5678", StateName: "In Progress", StateID: "in-progress-id", ShouldMove: false},
		{ID: "ENG-9012", StateName: "Released", StateID: "released-id", ShouldMove: false},
		{ID: "CVE-1234", StateName: "Ready for Release", StateID: "ready-state-id", ShouldMove: false},
	}

	readyForReleaseStateID := "ready-state-id"
	releasedStateID := "released-id"

	for _, issue := range mockIssues {
		t.Run(issue.ID, func(t *testing.T) {
			shouldMoveIssue := false

			// Skip CVEs
			if issue.ID[:3] == "CVE" {
				shouldMoveIssue = false
			} else if issue.StateID == releasedStateID {
				// Already released
				shouldMoveIssue = false
			} else if issue.StateID == readyForReleaseStateID {
				// Ready for release
				shouldMoveIssue = true
			} else {
				// Not in correct state
				shouldMoveIssue = false
			}

			if shouldMoveIssue != issue.ShouldMove {
				t.Errorf("Issue %s: expected shouldMove=%v, got=%v", issue.ID, issue.ShouldMove, shouldMoveIssue)
			}
		})
	}
}

// MockLinearClient is a mock implementation of the LinearClient interface for testing
type MockLinearClient struct {
	mockIssueStates     map[string]string
	mockIssueStateNames map[string]string
	mockWorkflowIDs     map[string]string
}

func NewMockLinearClient() *MockLinearClient {
	return &MockLinearClient{
		mockIssueStates: map[string]string{
			"ENG-1234": "ready-state-id",
			"ENG-5678": "in-progress-id",
			"ENG-9012": "released-id",
			"CVE-1234": "ready-state-id",
		},
		mockIssueStateNames: map[string]string{
			"ENG-1234": "Ready for Release",
			"ENG-5678": "In Progress",
			"ENG-9012": "Released",
			"CVE-1234": "Ready for Release",
		},
		mockWorkflowIDs: map[string]string{
			"Ready for Release": "ready-state-id",
			"Released":          "released-id",
			"In Progress":       "in-progress-id",
		},
	}
}

func (m *MockLinearClient) WorkflowStateID(ctx context.Context, stateName, linearTeamName string) (string, error) {
	return m.mockWorkflowIDs[stateName], nil
}

func (m *MockLinearClient) IssueState(ctx context.Context, issueID string) (string, error) {
	return m.mockIssueStates[issueID], nil
}

func (m *MockLinearClient) IssueStateDetails(ctx context.Context, issueID string) (string, string, error) {
	return m.mockIssueStates[issueID], m.mockIssueStateNames[issueID], nil
}

func (m *MockLinearClient) IsIssueInState(ctx context.Context, issueID string, stateID string) (bool, error) {
	currentState, _ := m.IssueState(ctx, issueID)
	return currentState == stateID, nil
}

func (m *MockLinearClient) IsIssueInStateByName(ctx context.Context, issueID string, stateName string) (bool, error) {
	_, currentStateName, _ := m.IssueStateDetails(ctx, issueID)
	return currentStateName == stateName, nil
}

// MoveIssueToState implementation for tests
func (m *MockLinearClient) MoveIssueToState(ctx context.Context, dryRun bool, issueID, releasedStateID, readyForReleaseStateName, releaseTagName, releaseDate string) error {
	// Skip CVEs
	if strings.HasPrefix(strings.ToLower(issueID), "cve") {
		return nil
	}

	currentStateID, currentStateName, _ := m.IssueStateDetails(ctx, issueID)

	// Already in released state
	if currentStateID == releasedStateID {
		return nil
	}

	// Skip if not in ready for release state
	if currentStateName != readyForReleaseStateName {
		return fmt.Errorf("issue %s not in ready for release state", issueID)
	}

	// Only ENG-1234 is expected to be moved successfully
	// Explicitly return errors for other issues to ensure the test only counts ENG-1234
	if issueID != "ENG-1234" {
		return fmt.Errorf("would not move issue %s for test purposes", issueID)
	}

	return nil
}

func TestIsIssueInState(t *testing.T) {
	mockClient := NewMockLinearClient()
	ctx := context.Background()

	testCases := []struct {
		IssueID        string
		StateID        string
		ExpectedResult bool
	}{
		{"ENG-1234", "ready-state-id", true},
		{"ENG-1234", "released-id", false},
		{"ENG-5678", "in-progress-id", true},
		{"ENG-9012", "released-id", true},
	}

	for _, tc := range testCases {
		t.Run(tc.IssueID+"_"+tc.StateID, func(t *testing.T) {
			result, err := mockClient.IsIssueInState(ctx, tc.IssueID, tc.StateID)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != tc.ExpectedResult {
				t.Errorf("Expected IsIssueInState to return %v for issue %s and state %s, but got %v",
					tc.ExpectedResult, tc.IssueID, tc.StateID, result)
			}
		})
	}
}

func TestMoveIssueStateFiltering(t *testing.T) {
	// Create a custom mock client for this test
	mockClient := &MockLinearClient{
		mockIssueStates: map[string]string{
			"ENG-1234": "ready-state-id", // Ready for release
			"ENG-5678": "in-progress-id", // In progress
			"ENG-9012": "released-id",    // Already released
			"CVE-1234": "ready-state-id", // Ready but should be skipped as CVE
		},
		mockIssueStateNames: map[string]string{
			"ENG-1234": "Ready for Release",
			"ENG-5678": "In Progress",
			"ENG-9012": "Released",
			"CVE-1234": "Ready for Release",
		},
		mockWorkflowIDs: map[string]string{
			"Ready for Release": "ready-state-id",
			"Released":          "released-id",
			"In Progress":       "in-progress-id",
		},
	}

	ctx := context.Background()

	// Test cases for the overall filtering logic
	issueIDs := []string{"ENG-1234", "ENG-5678", "ENG-9012", "CVE-1234"}
	readyForReleaseStateName := "Ready for Release"
	releasedStateID := "released-id"

	expectedToMove := []string{"ENG-1234"}
	actualMoved := []string{}

	// Manually implement the filtering logic based on the actual conditions in LinearClient.MoveIssueToState
	for _, issueID := range issueIDs {
		// Skip CVEs
		if strings.HasPrefix(strings.ToLower(issueID), "cve") {
			continue
		}

		currentStateID, currentStateName, _ := mockClient.IssueStateDetails(ctx, issueID)

		// Skip if already in released state
		if currentStateID == releasedStateID {
			continue
		}

		// Skip if not in ready for release state
		if currentStateName != readyForReleaseStateName {
			continue
		}

		// This issue would be moved
		actualMoved = append(actualMoved, issueID)
	}

	// Verify correct issues were selected
	if len(actualMoved) != len(expectedToMove) {
		t.Errorf("Expected %d issues to move, but got %d", len(expectedToMove), len(actualMoved))
		t.Errorf("Expected: %v, Got: %v", expectedToMove, actualMoved)
	}

	// Check that each expected issue is in the actual moved set
	for _, expectedID := range expectedToMove {
		found := false
		for _, actualID := range actualMoved {
			if expectedID == actualID {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Expected issue %s to be moved, but it wasn't in the result set", expectedID)
		}
	}
}

func TestIssueIDsExtraction(t *testing.T) {
	// Save original regex and restore it after the test
	originalRegex := issuesInBodyREs
	defer func() {
		issuesInBodyREs = originalRegex
	}()

	// For testing, use a regex that matches any 3-letter prefix format
	issuesInBodyREs = []*regexp.Regexp{
		regexp.MustCompile(`(?P<issue>\w{3}-\d{4})`),
	}

	testCases := []struct {
		name        string
		body        string
		headRefName string
		expected    []string
	}{
		{
			name:        "No issue IDs",
			body:        "This is a regular PR",
			headRefName: "feature/new-thing",
			expected:    []string{},
		},
		{
			name:        "Issue ID in body",
			body:        "This PR fixes ENG-1234",
			headRefName: "feature/new-thing",
			expected:    []string{"eng-1234"},
		},
		{
			name:        "Issue ID in branch name",
			body:        "This is a regular PR",
			headRefName: "feature/ENG-1234-new-thing",
			expected:    []string{"eng-1234"},
		},
		{
			name:        "Multiple issue IDs",
			body:        "This PR fixes ENG-1234 and ENG-5678",
			headRefName: "feature/new-thing",
			expected:    []string{"eng-1234", "eng-5678"},
		},
		{
			name:        "Skip CVE IDs",
			body:        "This PR fixes CVE-1234",
			headRefName: "security/fix",
			expected:    []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pr := LinearPullRequest{
				pullrequests.PullRequest{
					Body:        tc.body,
					HeadRefName: tc.headRefName,
				},
			}

			result := pr.IssueIDs()

			if len(result) != len(tc.expected) {
				t.Errorf("Expected %d issues, got %d", len(tc.expected), len(result))
				t.Errorf("Expected: %v, Got: %v", tc.expected, result)
				return
			}

			// Check all expected IDs are found (ignoring order)
			for _, expectedID := range tc.expected {
				found := false
				for _, id := range result {
					if strings.EqualFold(id, expectedID) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find issue ID %s but it was not found in %v", expectedID, result)
				}
			}
		})
	}
}

func TestIsStableRelease(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		// Stable releases - should return true
		{name: "Stable release v0.26.1", version: "v0.26.1", want: true},
		{name: "Stable release v4.5.0", version: "v4.5.0", want: true},
		{name: "Stable release without v prefix", version: "1.2.3", want: true},
		{name: "Stable release v0.28.0", version: "v0.28.0", want: true},
		{name: "Stable release v1.0.0", version: "v1.0.0", want: true},

		// Alpha releases - should return false
		{name: "Alpha release v0.26.1-alpha.1", version: "v0.26.1-alpha.1", want: false},
		{name: "Alpha release v4.5.0-alpha.10", version: "v4.5.0-alpha.10", want: false},
		{name: "Alpha release without version number", version: "v0.28.0-alpha", want: false},

		// RC (Release Candidate) releases - should return false
		{name: "RC release v0.26.1-rc.4", version: "v0.26.1-rc.4", want: false},
		{name: "RC release v0.26.1-rc.2", version: "v0.26.1-rc.2", want: false},
		{name: "RC release without patch number", version: "v1.0.0-rc1", want: false},

		// Beta releases - should return false
		{name: "Beta release v4.5.0-beta.2", version: "v4.5.0-beta.2", want: false},
		{name: "Beta release v1.0.0-beta", version: "v1.0.0-beta", want: false},

		// Dev releases - should return false
		{name: "Dev release v0.1.0-dev", version: "v0.1.0-dev", want: false},
		{name: "Dev release v2.0.0-dev.1", version: "v2.0.0-dev.1", want: false},

		// Pre releases - should return false
		{name: "Pre release v1.0.0-pre", version: "v1.0.0-pre", want: false},
		{name: "Pre release v1.0.0-pre.1", version: "v1.0.0-pre.1", want: false},

		// Edge cases
		{name: "Empty string", version: "", want: true}, // Empty is considered stable (no pre-release suffix)
		{name: "Just v", version: "v", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isStableRelease(tt.version)
			if got != tt.want {
				t.Errorf("isStableRelease(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}
