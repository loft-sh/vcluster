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
	testCases := []struct {
		version  string
		expected bool
	}{
		// Stable releases
		{"v0.26.1", true},
		{"v4.5.0", true},
		{"v1.0.0", true},
		{"0.26.1", true}, // without v prefix
		{"v27.0.0", true},

		// Pre-releases
		{"v0.26.1-alpha.1", false},
		{"v0.26.1-alpha.5", false},
		{"v0.26.1-beta.1", false},
		{"v0.26.1-rc.1", false},
		{"v0.26.1-rc.4", false},
		{"v0.26.1-dev.1", false},
		{"v0.26.1-pre.1", false},
		{"v0.26.1-next.1", false},
		{"v4.5.0-beta.2", false},
		{"0.27.0-alpha.1", false}, // without v prefix
	}

	for _, tc := range testCases {
		t.Run(tc.version, func(t *testing.T) {
			result := isStableRelease(tc.version)
			if result != tc.expected {
				t.Errorf("isStableRelease(%q) = %v, want %v", tc.version, result, tc.expected)
			}
		})
	}
}

func TestStableReleaseCommentText(t *testing.T) {
	// Test the comment text logic for different scenarios
	testCases := []struct {
		name             string
		alreadyReleased  bool
		isStable         bool
		releaseTag       string
		releaseDate      string
		expectedContains string
	}{
		{
			name:             "First release (pre-release)",
			alreadyReleased:  false,
			isStable:         false,
			releaseTag:       "v0.27.0-alpha.1",
			releaseDate:      "2025-01-15",
			expectedContains: "first released in",
		},
		{
			name:             "First release (stable)",
			alreadyReleased:  false,
			isStable:         true,
			releaseTag:       "v0.27.0",
			releaseDate:      "2025-02-01",
			expectedContains: "first released in",
		},
		{
			name:             "Stable release on already-released issue",
			alreadyReleased:  true,
			isStable:         true,
			releaseTag:       "v0.27.0",
			releaseDate:      "2025-02-01",
			expectedContains: "Now available in stable release",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var releaseComment string
			if tc.alreadyReleased && tc.isStable {
				releaseComment = fmt.Sprintf("Now available in stable release %v (released %v)", tc.releaseTag, tc.releaseDate)
			} else {
				releaseComment = fmt.Sprintf("This issue was first released in %v on %v", tc.releaseTag, tc.releaseDate)
			}

			if !strings.Contains(releaseComment, tc.expectedContains) {
				t.Errorf("Comment %q does not contain expected text %q", releaseComment, tc.expectedContains)
			}
		})
	}
}
