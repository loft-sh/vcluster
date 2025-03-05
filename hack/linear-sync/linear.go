package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/shurcooL/graphql"
)

var ErrNoWorkflowFound = errors.New("no workflow state found")

type LinearClient struct {
	client *graphql.Client
}

var _ http.RoundTripper = (*transport)(nil)

type transport struct {
	token string
}

// RoundTrip implements http.RoundTripper.
func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", t.token)
	return http.DefaultTransport.RoundTrip(req)
}

// NewLinearClient creates a new LinearClient.
func NewLinearClient(ctx context.Context, token string) LinearClient {
	httpClient := &http.Client{
		Transport: &transport{token: token},
	}
	client := graphql.NewClient("https://api.linear.app/graphql", httpClient)

	return LinearClient{client: client}
}

// WorkflowStateID returns the ID of the a workflow state for the given team.
func (l *LinearClient) WorkflowStateID(ctx context.Context, stateName, linearTeamName string) (string, error) {
	var query struct {
		WorkflowStates struct {
			Nodes []struct {
				Id string
			}
		} `graphql:"workflowStates(filter: { name: { eq: $name }, team: { name: { eq: $team } } })"`
	}

	variables := map[string]any{
		"name": graphql.String(stateName),
		"team": graphql.String(linearTeamName),
	}

	if err := l.client.Query(ctx, &query, variables); err != nil {
		return "", fmt.Errorf("query failed: %w", err)
	}

	if len(query.WorkflowStates.Nodes) == 0 {
		return "", ErrNoWorkflowFound
	}

	return query.WorkflowStates.Nodes[0].Id, nil
}

// IssueState returns the current state ID of the issue.
func (l *LinearClient) IssueState(ctx context.Context, issueID string) (string, error) {
	stateID, _, err := l.IssueStateDetails(ctx, issueID)
	return stateID, err
}

// IssueStateDetails returns the current state ID and name of the issue.
func (l *LinearClient) IssueStateDetails(ctx context.Context, issueID string) (string, string, error) {
	var query struct {
		Issue struct {
			State struct {
				Id   string
				Name string
			}
		} `graphql:"issue(id: $id)"`
	}

	variables := map[string]any{
		"id": graphql.String(issueID),
	}

	if err := l.client.Query(ctx, &query, variables); err != nil {
		return "", "", fmt.Errorf("query failed (issue ID: %v): %w", issueID, err)
	}

	return query.Issue.State.Id, query.Issue.State.Name, nil
}

// IsIssueInState checks if an issue is in a specific state.
func (l *LinearClient) IsIssueInState(ctx context.Context, issueID string, stateID string) (bool, error) {
	currentState, err := l.IssueState(ctx, issueID)
	if err != nil {
		return false, fmt.Errorf("get issue state: %w", err)
	}

	return currentState == stateID, nil
}

// IsIssueInStateByName checks if an issue is in a state with the specified name.
func (l *LinearClient) IsIssueInStateByName(ctx context.Context, issueID string, stateName string) (bool, error) {
	_, currentStateName, err := l.IssueStateDetails(ctx, issueID)
	if err != nil {
		return false, fmt.Errorf("get issue state details: %w", err)
	}

	return currentStateName == stateName, nil
}

// MoveIssueToState moves the issue to the given state if it's not already there.
// It also adds a comment to the issue about when it was first released and on which tag.
func (l *LinearClient) MoveIssueToState(ctx context.Context, dryRun bool, issueID, releasedStateID, readyForReleaseStateName, releaseTagName, releaseDate string) error {
	// (ThomasK33): Skip CVEs
	if strings.HasPrefix(strings.ToLower(issueID), "cve") {
		return nil
	}

	logger := ctx.Value(LoggerKey).(*slog.Logger)

	currentIssueStateID, currentIssueStateName, err := l.IssueStateDetails(ctx, issueID)
	if err != nil {
		return fmt.Errorf("get issue state details: %w", err)
	}

	if currentIssueStateID == releasedStateID {
		logger.Debug("Issue already has desired state", "issueID", issueID, "stateID", releasedStateID)
		return nil
	}

	// Skip issues not in ready for release state
	if currentIssueStateName != readyForReleaseStateName {
		logger.Debug("Skipping issue not in ready for release state", "issueID", issueID, "currentState", currentIssueStateName, "requiredState", readyForReleaseStateName)
		return nil
	}

	if !dryRun {
		if err := l.updateIssueState(ctx, issueID, releasedStateID); err != nil {
			return fmt.Errorf("update issue state: %w", err)
		}
	} else {
		logger.Info("Would update issue state", "issueID", issueID, "releasedStateID", releasedStateID)
	}

	releaseComment := fmt.Sprintf("This issue was first released in %v on %v", releaseTagName, releaseDate)

	if !dryRun {
		if err := l.createComment(ctx, issueID, releaseComment); err != nil {
			return fmt.Errorf("create comment: %w", err)
		}
	} else {
		logger.Info("Would create comment on issue", "issueID", issueID, "comment", releaseComment)
	}

	logger.Info("Moved issue to desired state", "issueID", issueID, "stateID", releasedStateID)

	return nil
}

// updateIssueState updates the state of the given issue.
func (l *LinearClient) updateIssueState(ctx context.Context, issueID, releasedStateID string) error {
	var mutation struct {
		IssueUpdate struct {
			Success bool
		} `graphql:"issueUpdate(input: { stateId: $stateID }, id: $issueID)"`
	}

	variables := map[string]any{
		"issueID": graphql.String(issueID),
		"stateID": graphql.String(releasedStateID),
	}

	if err := l.client.Mutate(ctx, &mutation, variables); err != nil || !mutation.IssueUpdate.Success {
		return fmt.Errorf("mutation failed: %w", err)
	}

	return nil
}

// createComment creates a comment on the given issue.
func (l *LinearClient) createComment(ctx context.Context, issueID, releaseComment string) error {
	var mutation struct {
		CommentCreate struct {
			Success bool
		} `graphql:"commentCreate(input: { issueId: $issueID, body: $body, doNotSubscribeToIssue: true })"`
	}

	variables := map[string]any{
		"issueID": graphql.String(issueID),
		"body":    graphql.String(releaseComment),
	}

	if err := l.client.Mutate(ctx, &mutation, variables); err != nil || !mutation.CommentCreate.Success {
		return fmt.Errorf("mutation failed: %w", err)
	}

	return nil
}

