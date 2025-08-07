package pullrequests

import (
	"context"
	"fmt"
	"time"

	"github.com/shurcooL/githubv4"
)

const PageSize = 100

type PullRequest struct {
	Merged      bool
	Body        string
	HeadRefName string
	Author      struct {
		Login string
	}
	Number   int
	MergedAt *githubv4.DateTime
}

// FetchAllPRsBetween fetches all merged PRs between the given tags
// It uses the GitHub GraphQL API to fetch the PRs.
func FetchAllPRsBetween(ctx context.Context, client *githubv4.Client, owner, repo, previousTag, currentTag string) ([]PullRequest, error) {
	var query struct {
		Repository struct {
			Ref struct {
				Compare struct {
					Commits struct {
						PageInfo struct {
							EndCursor   githubv4.String
							HasNextPage bool
						}
						Nodes []struct {
							AssociatedPullRequests struct {
								PageInfo struct {
									EndCursor   githubv4.String
									HasNextPage bool
								}
								Nodes []PullRequest
							} `graphql:"associatedPullRequests(first: $pageSize)"`
						}
					} `graphql:"commits(first: $pageSize, after: $cursor)"`
				} `graphql:"compare(headRef: $currTag)"`
			} `graphql:"ref(qualifiedName: $prevTag)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	var cursor *githubv4.String
	pullRequestsByNumber := map[int]PullRequest{}

	// Paginate through the Commits
	for {
		if err := client.Query(ctx, &query, map[string]interface{}{
			"owner":    githubv4.String(owner),
			"repo":     githubv4.String(repo),
			"prevTag":  githubv4.String(previousTag),
			"currTag":  githubv4.String(currentTag),
			"pageSize": githubv4.Int(PageSize),
			"cursor":   cursor,
		}); err != nil {
			return nil, fmt.Errorf("query repository: %w", err)
		}

		cursor = &query.Repository.Ref.Compare.Commits.PageInfo.EndCursor

		for _, commit := range query.Repository.Ref.Compare.Commits.Nodes {
			for _, pr := range commit.AssociatedPullRequests.Nodes {
				if !pr.Merged {
					continue
				}

				if _, ok := pullRequestsByNumber[pr.Number]; ok {
					continue
				}

				pullRequestsByNumber[pr.Number] = pr
			}
		}

		if !query.Repository.Ref.Compare.Commits.PageInfo.HasNextPage {
			break
		}
	}

	var pullRequests []PullRequest
	for _, pr := range pullRequestsByNumber {
		pullRequests = append(pullRequests, pr)
	}
	return pullRequests, nil
}

// FetchPRsForRelease fetches PRs that were actually included in a specific release
// by filtering out PRs that were merged after the release was published.
// This prevents PRs that were merged after the release tag was created from being included.
func FetchPRsForRelease(ctx context.Context, client *githubv4.Client, owner, repo, previousTag, releaseTag string, releasePublishedAt time.Time) ([]PullRequest, error) {
	// Fetch all PRs between the tags (now includes MergedAt)
	prs, err := FetchAllPRsBetween(ctx, client, owner, repo, previousTag, releaseTag)
	if err != nil {
		return nil, fmt.Errorf("fetch all PRs between tags: %w", err)
	}

	// Filter based on merge time directly from the PR data
	var filtered []PullRequest
	for _, pr := range prs {
		// Time-based filtering: exclude PRs merged after release was published
		if pr.MergedAt != nil && pr.MergedAt.After(releasePublishedAt) {
			continue
		}

		// Include PR if it has a valid merge time before the release
		if pr.MergedAt != nil {
			filtered = append(filtered, pr)
		}
	}

	return filtered, nil
}
