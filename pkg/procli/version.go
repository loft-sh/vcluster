package procli

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/google/go-github/v53/github"
	"github.com/loft-sh/log"
	"github.com/samber/lo"
)

var (
	MinimumVersionTag = "v3.3.0-alpha.26"
	MinimumVersion    = semver.MustParse(strings.TrimPrefix(MinimumVersionTag, "v"))
)

// LatestCompatibleVersion returns the latest compatible version of vCluster.Pro
func LatestCompatibleVersion(ctx context.Context) (string, error) {
	client := github.NewClient(nil)

	release, _, err := client.Repositories.GetLatestRelease(ctx, "loft-sh", "loft")
	if err != nil {
		return "", fmt.Errorf("failed to get latest release: %w", err)
	}

	tagName := release.GetTagName()

	if tagName == "" {
		return "", fmt.Errorf("failed to get latest release tag name")
	}

	ghVersion, err := semver.Parse(strings.TrimPrefix(tagName, "v"))
	if err != nil {
		log.GetInstance().Warnf("failed to parse latest release tag name, falling back to %s: %v", MinimumVersionTag, err)
		return MinimumVersionTag, nil
	}

	if ghVersion.GTE(MinimumVersion) {
		return tagName, nil
	}

	releases, _, err := client.Repositories.ListReleases(ctx, "loft-sh", "loft", &github.ListOptions{})
	if err != nil {
		log.GetInstance().Warnf("failed to fetch releases from Github, falling back to %s: %v", MinimumVersionTag, err)
		return MinimumVersionTag, nil
	}

	eligibleReleases := lo.FilterMap(releases, func(release *github.RepositoryRelease, _ int) (semver.Version, bool) {
		tagName := release.GetTagName()
		if tagName == "" {
			return semver.Version{}, false
		}

		ghVersion, err := semver.Parse(strings.TrimPrefix(tagName, "v"))
		if err != nil {
			return semver.Version{}, false
		}

		return ghVersion, ghVersion.GTE(MinimumVersion)
	})

	sort.Slice(eligibleReleases, func(i, j int) bool {
		return eligibleReleases[i].LT(eligibleReleases[j])
	})

	if len(eligibleReleases) > 0 {
		return "v" + eligibleReleases[len(eligibleReleases)-1].String(), nil
	}

	return MinimumVersionTag, nil
}
