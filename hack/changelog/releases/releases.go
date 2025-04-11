package releases

import (
	"context"
	"fmt"
	"strings"

	"github.com/blang/semver"
	"github.com/google/go-github/v59/github"
	"github.com/shurcooL/githubv4"
)

const PageSize = 100

// LastStableRelease returns the last stable release for the given repository.
// It returns the tag name, id and the creation time of the release.
func LastStableRelease(ctx context.Context, client *githubv4.Client, owner, repo string) (string, int, error) {
	var query struct {
		Repository struct {
			LatestRelease struct {
				CreatedAt  githubv4.DateTime
				TagName    string
				DatabaseId int
			}
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	if err := client.Query(ctx, &query, map[string]interface{}{
		"owner": githubv4.String(owner),
		"repo":  githubv4.String(repo),
	}); err != nil {
		return "", 0, fmt.Errorf("query latest release: %w", err)
	}

	return query.Repository.LatestRelease.TagName, query.Repository.LatestRelease.DatabaseId, nil
}

func LastStableReleaseBeforeTag(ctx context.Context, client *githubv4.Client, owner, repo, tag string) (string, error) {
	sanitizedTag, _ := strings.CutPrefix(tag, "v")
	tagSemver, err := semver.ParseTolerant(sanitizedTag)
	if err != nil {
		return "", fmt.Errorf("failed to parse tag: %w", err)
	}

	return LatestStableSemverRange(ctx, client, owner, repo, "<"+tagSemver.String())
}

func LatestStableSemverRange(ctx context.Context, client *githubv4.Client, owner, repo, tagRangeExpr string) (string, error) {
	tagRange, err := semver.ParseRange(tagRangeExpr)
	if err != nil {
		// Ignore bad ranges for now.
		return "", fmt.Errorf("failed to parse tag: %w", err)
	}

	var query struct {
		Repository struct {
			Releases struct {
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
				Nodes []struct {
					TagName      string
					IsPrerelease bool
				}
			} `graphql:"releases(first: $pageSize, after: $cursor, orderBy: { direction: DESC, field: CREATED_AT})"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	var cursor *githubv4.String

	// Paginate through the Releases
	for {
		if err := client.Query(ctx, &query, map[string]interface{}{
			"owner":    githubv4.String(owner),
			"repo":     githubv4.String(repo),
			"pageSize": githubv4.Int(PageSize),
			"cursor":   cursor,
		}); err != nil {
			return "", fmt.Errorf("query repository: %w", err)
		}

		cursor = &query.Repository.Releases.PageInfo.EndCursor

		for _, release := range query.Repository.Releases.Nodes {
			releaseSemver, err := semver.ParseTolerant(release.TagName)
			if err != nil {
				continue
			}

			if len(releaseSemver.Pre) > 0 {
				continue
			}

			if release.IsPrerelease {
				continue
			}

			if tagRange(releaseSemver) {
				return release.TagName, nil
			}
		}

		if !query.Repository.Releases.PageInfo.HasNextPage {
			break
		}
	}

	return "", nil
}

func LatestRelease(ctx context.Context, client *githubv4.Client, owner, repo string) (string, error) {
	var query struct {
		Repository struct {
			Releases struct {
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
				Nodes []struct {
					TagName      string
					IsPrerelease bool
				}
			} `graphql:"releases(first: $pageSize, orderBy: { direction: DESC, field: CREATED_AT})"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	// Get the latest out of the top page
	if err := client.Query(ctx, &query, map[string]interface{}{
		"owner":    githubv4.String(owner),
		"repo":     githubv4.String(repo),
		"pageSize": githubv4.Int(PageSize),
	}); err != nil {
		return "", fmt.Errorf("query repository: %w", err)
	}

	latestTag := ""
	latestSemver := semver.MustParse("0.0.0")
	for _, release := range query.Repository.Releases.Nodes {
		releaseSemver, err := semver.ParseTolerant(release.TagName)
		if err != nil {
			continue
		}

		if releaseSemver.Compare(latestSemver) == 1 {
			latestSemver = releaseSemver
			latestTag = release.TagName
		} else {
			continue
		}
	}

	return latestTag, nil
}

type Release struct {
	PublishedAt githubv4.DateTime
	Description string
	Name        string
	TagName     string
	DatabaseId  int64
}

// FetchReleaseByTag fetches a release by its tag name.
// It returns the release or an error if the release could not be found.
func FetchReleaseByTag(ctx context.Context, client *githubv4.Client, owner, repo, tag string) (Release, error) {
	var query struct {
		Repository struct {
			Release Release `graphql:"release(tagName: $tag)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	if err := client.Query(ctx, &query, map[string]interface{}{
		"owner": githubv4.String(owner),
		"repo":  githubv4.String(repo),
		"tag":   githubv4.String(tag),
	}); err != nil {
		return Release{}, fmt.Errorf("query release by tag: %w", err)
	}

	return query.Repository.Release, nil
}

// UpdateReleaseNotes updates the release notes of the given release.
// It returns an error if the release notes could not be updated.
func UpdateReleaseNotes(ctx context.Context, client *github.Client, owner, repo string, releaseId int64, notes string) error {
	_, _, err := client.Repositories.EditRelease(ctx, owner, repo, releaseId, &github.RepositoryRelease{
		Body: &notes,
	})
	if err != nil {
		return fmt.Errorf("update release notes: %w", err)
	}

	return nil
}
