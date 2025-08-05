package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"

	pullrequests "github.com/loft-sh/changelog/pull-requests"
	"github.com/loft-sh/changelog/releases"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

var (
	ErrMissingGitHubToken = errors.New("github token must be set")
	ErrMissingLinearToken = errors.New("linear token must be set")
	ErrMissingReleaseTag  = errors.New("release tag must be set")
)

var LoggerKey = struct{ name string }{"logger"}

func main() {
	if err := run(context.Background(), io.Writer(os.Stderr), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(
	ctx context.Context,
	stderr io.Writer,
	args []string,
) error {
	flagset := flag.NewFlagSet(args[0], flag.ExitOnError)
	var (
		owner                    = flagset.String("owner", "loft-sh", "The GitHub owner of the repository")
		repo                     = flagset.String("repo", "vcluster", "The GitHub repository to generate the changelog for")
		githubToken              = flagset.String("token", "", "The GitHub token to use for authentication")
		previousTag              = flagset.String("previous-tag", "", "The previous tag to generate the changelog for (if not set, the last stable release will be used)")
		releaseTag               = flagset.String("release-tag", "", "The tag of the new release")
		debug                    = flagset.Bool("debug", false, "Enable debug logging")
		linearToken              = flagset.String("linear-token", "", "The Linear token to use for authentication")
		releasedStateName        = flagset.String("released-state-name", "Released", "The name of the state to use for the released state")
		readyForReleaseStateName = flagset.String("ready-for-release-state-name", "Ready for Release", "The name of the state that indicates an issue is ready to be released")
		linearTeamName           = flagset.String("linear-team-name", "vCluster / Platform", "The name of the team to use for the linear team")
		dryRun                   = flagset.Bool("dry-run", false, "Do not actually move issues to the released state")
		strictFiltering          = flagset.Bool("strict-filtering", true, "Only include PRs that were actually merged before the release was published (recommended to avoid false positives)")
	)
	if err := flagset.Parse(args[1:]); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	if *githubToken == "" {
		*githubToken = os.Getenv("GITHUB_TOKEN")
	}

	if *linearToken == "" {
		*linearToken = os.Getenv("LINEAR_TOKEN")
	}

	if *githubToken == "" {
		return ErrMissingGitHubToken
	}

	if *releaseTag == "" {
		return ErrMissingReleaseTag
	}

	if *linearToken == "" {
		return ErrMissingLinearToken
	}

	leveler := slog.LevelVar{}
	leveler.Set(slog.LevelInfo)
	if *debug {
		leveler.Set(slog.LevelDebug)
	}

	logger := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{
		Level: &leveler,
	}))

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, os.Kill)
	defer stop()

	ctx = context.WithValue(ctx, LoggerKey, logger)

	httpClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(
		&oauth2.Token{
			AccessToken: *githubToken,
		},
	))

	gqlClient := githubv4.NewClient(httpClient)

	var stableTag string

	if *previousTag != "" {
		release, err := releases.FetchReleaseByTag(ctx, gqlClient, *owner, *repo, *previousTag)
		if err != nil {
			return fmt.Errorf("fetch release by tag: %w", err)
		}

		stableTag = release.TagName
	} else {
		if prevRelease, err := releases.LastStableReleaseBeforeTag(ctx, gqlClient, *owner, *repo, *releaseTag); err != nil {
			return fmt.Errorf("get last stable release before tag: %w", err)
		} else if prevRelease != "" {
			stableTag = prevRelease
		} else {
			stableTag, _, err = releases.LastStableRelease(ctx, gqlClient, *owner, *repo)
			if err != nil {
				return fmt.Errorf("get last stable release: %w", err)
			}
		}
	}

	if stableTag == "" {
		return errors.New("no stable release found")
	}

	logger.Info("Last stable release", "stableTag", stableTag)

	currentRelease, err := releases.FetchReleaseByTag(ctx, gqlClient, *owner, *repo, *releaseTag)
	if err != nil {
		return fmt.Errorf("fetch release by tag: %w", err)
	}

	if currentRelease.TagName != *releaseTag {
		return fmt.Errorf("release not found: %s", *releaseTag)
	}

	prs, err := pullrequests.FetchAllPRsBetween(ctx, gqlClient, *owner, *repo, stableTag, *releaseTag)
	if err != nil {
		return fmt.Errorf("fetch all PRs until: %w", err)
	}

	var pullRequests []LinearPullRequest
	if *strictFiltering {
		// Filter PRs to only include those that were actually part of this release
		filteredPRs, err := pullrequests.FetchPRsForRelease(ctx, gqlClient, *owner, *repo, stableTag, *releaseTag, currentRelease.PublishedAt.Time)
		if err != nil {
			return fmt.Errorf("filter PRs for release: %w", err)
		}
		pullRequests = NewLinearPullRequests(filteredPRs)
		logger.Info("Found merged pull requests for release", "total", len(prs), "filtered", len(pullRequests), "previous", stableTag, "current", *releaseTag)
	} else {
		// Use all PRs between tags (original behavior)
		pullRequests = NewLinearPullRequests(prs)
		logger.Info("Found merged pull requests between releases", "count", len(pullRequests), "previous", stableTag, "current", *releaseTag)
	}

	releasedIssues := []string{}

	for _, pr := range pullRequests {
		if issueIDs := pr.IssueIDs(); len(issueIDs) > 0 {
			for _, issueID := range issueIDs {
				releasedIssues = append(releasedIssues, issueID)
				logger.Debug("Found issue in pull request", "issueID", issueID, "pr", pr.Number)
			}
		}
	}

	logger.Info("Found issues in pull requests", "count", len(releasedIssues))

	linearClient := NewLinearClient(ctx, *linearToken)

	releasedStateID, err := linearClient.WorkflowStateID(ctx, *releasedStateName, *linearTeamName)
	if err != nil {
		return fmt.Errorf("get released workflow ID: %w", err)
	}

	logger.Debug("Found released workflow ID", "workflowID", releasedStateID)

	readyForReleaseStateID, err := linearClient.WorkflowStateID(ctx, *readyForReleaseStateName, *linearTeamName)
	if err != nil {
		return fmt.Errorf("get ready for release workflow ID: %w", err)
	}

	logger.Debug("Found ready for release workflow ID", "workflowID", readyForReleaseStateID)

	currentReleaseDateStr := currentRelease.PublishedAt.Format("2006-01-02")

	releasedCount := 0
	skippedCount := 0

	for _, issueID := range releasedIssues {
		if err := linearClient.MoveIssueToState(ctx, *dryRun, issueID, releasedStateID, *readyForReleaseStateName, currentRelease.TagName, currentReleaseDateStr); err != nil {
			logger.Error("Failed to move issue to state", "issueID", issueID, "error", err)
			skippedCount++
		} else {
			releasedCount++
		}
	}

	logger.Info("Linear sync completed", "processed", len(releasedIssues), "released", releasedCount, "skipped", skippedCount)

	return nil
}
