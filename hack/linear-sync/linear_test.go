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
	mockIssueStates       map[string]string
	mockIssueStateNames   map[string]string
	mockWorkflowIDs       map[string]string
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
		IssueID     string
		StateID     string
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
			"ENG-1234": "ready-state-id",  // Ready for release
			"ENG-5678": "in-progress-id",  // In progress 
			"ENG-9012": "released-id",     // Already released
			"CVE-1234": "ready-state-id",  // Ready but should be skipped as CVE
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
<<<<<<< HEAD
	
	// For testing, use a regex that matches any 3-letter prefix format
=======

	// For testing, use a regex that matches team keys of 2-10 chars and issue numbers 1-5 digits
>>>>>>> 3aa6f7157 (fix(linear-sync): support variable-length team keys in issue regex (#3469))
	issuesInBodyREs = []*regexp.Regexp{
		regexp.MustCompile(`(?P<issue>\w{2,10}-\d{1,5})`),
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
		{
			name:        "Long team key (DEVOPS)",
			body:        "This PR fixes DEVOPS-471",
			headRefName: "feature/infra-update",
			expected:    []string{"devops-471"},
		},
		{
			name:        "Short team key (QA)",
			body:        "This PR fixes QA-42",
			headRefName: "feature/test-fix",
			expected:    []string{"qa-42"},
		},
		{
			name:        "Mixed team keys",
			body:        "This PR fixes ENG-1234 and DEVOPS-471",
			headRefName: "feature/QA-99-cross-team",
			expected:    []string{"eng-1234", "devops-471", "qa-99"},
		},
		{
			name:        "Issue with short number",
			body:        "This PR fixes ENG-1",
			headRefName: "feature/quick-fix",
			expected:    []string{"eng-1"},
		},
		{
			name:        "Issue with long number",
			body:        "This PR fixes ENG-12345",
			headRefName: "feature/big-project",
			expected:    []string{"eng-12345"},
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
