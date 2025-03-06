package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"sort"

	"github.com/google/go-github/v59/github"
	"github.com/loft-sh/changelog/log"
	pullrequests "github.com/loft-sh/changelog/pull-requests"
	"github.com/loft-sh/changelog/releases"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

var ErrMissingToken = errors.New("github token must be set")

func main() {
	if err := run(context.Background(), os.Stdout, os.Stderr, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(
	ctx context.Context,
	stdout, stderr io.Writer,
	args []string,
) error {
	flagset := flag.NewFlagSet(args[0], flag.ExitOnError)
	var (
		owner       = flagset.String("owner", "loft-sh", "The GitHub owner of the repository")
		repo        = flagset.String("repo", "vcluster", "The GitHub repository to generate the changelog for")
		githubToken = flagset.String("token", "", "The GitHub token to use for authentication")
		previousTag = flagset.String("previous-tag", "", "The previous tag to generate the changelog for (if not set, the last stable release will be used)")
		releaseTag  = flagset.String("release-tag", "", "The tag of the release to generate the changelog for")
		updateNotes = flagset.Bool("update-notes", true, "Update the release notes of the release with the generated ones")
		overwrite   = flagset.Bool("overwrite", false, "Overwrite the release notes with the generated ones")
		debug       = flagset.Bool("debug", false, "Enable debug logging")
	)
	if err := flagset.Parse(args[1:]); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	if *githubToken == "" {
		*githubToken = os.Getenv("GITHUB_TOKEN")
	}

	if *githubToken == "" {
		return ErrMissingToken
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

	ctx = context.WithValue(ctx, log.LoggerKey, logger)

	httpClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(
		&oauth2.Token{
			AccessToken: *githubToken,
		},
	))

	client := github.NewClient(httpClient)
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

	var currentRelease releases.Release
	if *releaseTag != "" {
		var err error
		currentRelease, err = releases.FetchReleaseByTag(ctx, gqlClient, *owner, *repo, *releaseTag)
		if err != nil {
			return fmt.Errorf("fetch release by tag: %w", err)
		}

		if currentRelease.TagName != *releaseTag {
			return fmt.Errorf("release not found: %s", *releaseTag)
		}
	}

	pullRequests, err := pullrequests.FetchAllPRsBetween(ctx, gqlClient, *owner, *repo, stableTag, *releaseTag)
	if err != nil {
		return fmt.Errorf("fetch all PRs until: %w", err)
	}

	logger.Info("Found merged pull requests between releases", "count", len(pullRequests), "previous", stableTag, "current", *releaseTag)

	notes := []Note{}
	for _, pr := range pullRequests {
		notes = append(notes, NewNotesFromPullRequest(pr)...)
	}
	sort.Slice(notes, SortNotes(notes))

	buffer := bytes.Buffer{}

	for _, note := range notes {
		if _, err := buffer.Write([]byte(note.String())); err != nil {
			return fmt.Errorf("write note: %w", err)
		}
	}

	if *releaseTag != "" && *updateNotes {
		if currentRelease.Description == "" || *overwrite {
			if err := releases.UpdateReleaseNotes(ctx, client, *owner, *repo, currentRelease.DatabaseId, buffer.String()); err != nil {
				return fmt.Errorf("update release notes: %w", err)
			}
			logger.Info("Updated release notes", "releaseTag", *releaseTag)
		} else {
			logger.Warn("Release notes already exist for tag, skipping update", "releaseTag", *releaseTag)
		}
	}

	if _, err := stdout.Write(buffer.Bytes()); err != nil {
		return fmt.Errorf("write changelog: %w", err)
	}

	return nil
}
