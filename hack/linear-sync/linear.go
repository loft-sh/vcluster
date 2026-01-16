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

// AvailableWorkflowState lists all workflow states for a team (for debugging)
type AvailableWorkflowState struct {
	Name string
	Team string
}

// AvailableTeam lists a team with its key (for debugging)
type AvailableTeam struct {
	Name string
	Key  string
}

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

<<<<<<< HEAD
=======
// isStableRelease checks if a version is a stable release (no pre-release suffix).
// Returns true for stable releases like v0.26.1, v4.5.0
// Returns false for pre-releases like v0.26.1-alpha.1, v0.26.1-rc.4, v4.5.0-beta.2
func isStableRelease(version string) bool {
	// Remove 'v' prefix if present
	version = strings.TrimPrefix(version, "v")

	// Check for pre-release suffixes
	preReleaseSuffixes := []string{"-alpha", "-beta", "-rc", "-dev", "-pre", "-next"}
	for _, suffix := range preReleaseSuffixes {
		if strings.Contains(version, suffix) {
			return false
		}
	}

	return true
}

// ListTeams returns all available teams (for debugging workflow state lookup failures)
func (l *LinearClient) ListTeams(ctx context.Context) ([]AvailableTeam, error) {
	var query struct {
		Teams struct {
			Nodes []struct {
				Name string
				Key  string
			}
		} `graphql:"teams"`
	}

	if err := l.client.Query(ctx, &query, nil); err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	teams := make([]AvailableTeam, len(query.Teams.Nodes))
	for i, t := range query.Teams.Nodes {
		teams[i] = AvailableTeam{Name: t.Name, Key: t.Key}
	}
	return teams, nil
}

// ListWorkflowStates returns all workflow states for a team (for debugging workflow state lookup failures)
func (l *LinearClient) ListWorkflowStates(ctx context.Context, teamName string) ([]AvailableWorkflowState, error) {
	var query struct {
		WorkflowStates struct {
			Nodes []struct {
				Name string
				Team struct {
					Name string
				}
			}
		} `graphql:"workflowStates(filter: { team: { name: { eq: $team } } })"`
	}

	variables := map[string]any{
		"team": graphql.String(teamName),
	}

	if err := l.client.Query(ctx, &query, variables); err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	states := make([]AvailableWorkflowState, len(query.WorkflowStates.Nodes))
	for i, s := range query.WorkflowStates.Nodes {
		states[i] = AvailableWorkflowState{Name: s.Name, Team: s.Team.Name}
	}
	return states, nil
}

>>>>>>> be70d94c9 (fix(linear-sync): look up team per issue instead of using global default (#3495))
// WorkflowStateID returns the ID of the a workflow state for the given team.
// If no matching state is found, it provides debugging information about available teams and states.
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
		// Provide debugging information about available teams and states
		debugInfo := fmt.Sprintf("searched for state %q in team %q", stateName, linearTeamName)

		// Try to list available teams
		teams, err := l.ListTeams(ctx)
		if err == nil && len(teams) > 0 {
			teamNames := make([]string, len(teams))
			for i, t := range teams {
				teamNames[i] = fmt.Sprintf("%s (%s)", t.Name, t.Key)
			}
			debugInfo += fmt.Sprintf("; available teams: %s", strings.Join(teamNames, ", "))
		}

		// Try to list available workflow states for the team
		states, err := l.ListWorkflowStates(ctx, linearTeamName)
		if err == nil && len(states) > 0 {
			stateNames := make([]string, len(states))
			for i, s := range states {
				stateNames[i] = s.Name
			}
			debugInfo += fmt.Sprintf("; available states for team: %s", strings.Join(stateNames, ", "))
		} else if err == nil {
			debugInfo += "; no states found for team (team may not exist or may have been renamed)"
		}

		return "", fmt.Errorf("%w: %s", ErrNoWorkflowFound, debugInfo)
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
	details, err := l.GetIssueDetails(ctx, issueID)
	if err != nil {
		return "", "", err
	}
	return details.StateID, details.StateName, nil
}

// IssueDetails contains state and team information for an issue
type IssueDetails struct {
	StateID   string
	StateName string
	TeamName  string
}

// GetIssueDetails returns state and team information for an issue.
func (l *LinearClient) GetIssueDetails(ctx context.Context, issueID string) (*IssueDetails, error) {
	var query struct {
		Issue struct {
			State struct {
				Id   string
				Name string
			}
			Team struct {
				Name string
			}
		} `graphql:"issue(id: $id)"`
	}

	variables := map[string]any{
		"id": graphql.String(issueID),
	}

	if err := l.client.Query(ctx, &query, variables); err != nil {
		return nil, fmt.Errorf("query failed (issue ID: %v): %w", issueID, err)
	}

	return &IssueDetails{
		StateID:   query.Issue.State.Id,
		StateName: query.Issue.State.Name,
		TeamName:  query.Issue.Team.Name,
	}, nil
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

// MoveIssueToState moves the issue to the given state if it's not already there and if it's in the ready for release state.
// It also adds a comment to the issue about when it was first released and on which tag.
<<<<<<< HEAD
func (l *LinearClient) MoveIssueToState(ctx context.Context, dryRun bool, issueID, releasedStateID, readyForReleaseStateName, releaseTagName, releaseDate string) error {
=======
// For stable releases on already-released issues, it adds a "now available in stable" comment.
// issueDetails should be pre-fetched via GetIssueDetails to avoid redundant API calls.
func (l *LinearClient) MoveIssueToState(ctx context.Context, dryRun bool, issueID string, issueDetails *IssueDetails, releasedStateID, readyForReleaseStateName, releaseTagName, releaseDate string) error {
>>>>>>> be70d94c9 (fix(linear-sync): look up team per issue instead of using global default (#3495))
	// (ThomasK33): Skip CVEs
	if strings.HasPrefix(strings.ToLower(issueID), "cve") {
		return nil
	}

	logger := ctx.Value(LoggerKey).(*slog.Logger)

<<<<<<< HEAD
	currentIssueStateID, currentIssueStateName, err := l.IssueStateDetails(ctx, issueID)
	if err != nil {
		return fmt.Errorf("get issue state details: %w", err)
	}

	if currentIssueStateID == releasedStateID {
		logger.Debug("Issue already has desired state", "issueID", issueID, "stateID", releasedStateID)
		return nil
=======
	isStable := isStableRelease(releaseTagName)

	alreadyReleased := issueDetails.StateID == releasedStateID

	// If already in released state:
	// - Pre-releases: skip entirely (already released in a previous pre-release)
	// - Stable releases: skip state update but add "now available in stable" comment
	if alreadyReleased {
		if !isStable {
			logger.Debug("Issue already has desired state", "issueID", issueID, "stateID", releasedStateID)
			return nil
		}
		logger.Debug("Issue already released, adding stable release comment", "issueID", issueID)
	} else {
		// Skip issues not in ready for release state
		if issueDetails.StateName != readyForReleaseStateName {
			logger.Debug("Skipping issue not in ready for release state", "issueID", issueID, "currentState", issueDetails.StateName, "requiredState", readyForReleaseStateName)
			return nil
		}

		// Update issue state to Released
		if !dryRun {
			if err := l.updateIssueState(ctx, issueID, releasedStateID); err != nil {
				return fmt.Errorf("update issue state: %w", err)
			}
		} else {
			logger.Info("Would update issue state", "issueID", issueID, "releasedStateID", releasedStateID)
		}
		logger.Info("Moved issue to desired state", "issueID", issueID, "stateID", releasedStateID)
>>>>>>> be70d94c9 (fix(linear-sync): look up team per issue instead of using global default (#3495))
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

