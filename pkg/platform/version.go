package platform

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/google/go-github/v53/github"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/log"
	"github.com/samber/lo"
)

var (
	MinimumVersionTag = "v4.0.0-alpha.18"
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

type matchedVersion struct {
	Object  storagev1.VersionAccessor
	Version semver.Version
}

func GetLatestVersion(versions storagev1.VersionsAccessor) storagev1.VersionAccessor {
	// find the latest version
	var latestVersion *matchedVersion
	for _, version := range versions.GetVersions() {
		parsedVersion, err := semver.Parse(strings.TrimPrefix(version.GetVersion(), "v"))
		if err != nil {
			continue
		}

		// latest available version
		if latestVersion == nil || latestVersion.Version.LT(parsedVersion) {
			latestVersion = &matchedVersion{
				Object:  version,
				Version: parsedVersion,
			}
		}
	}
	if latestVersion == nil {
		return nil
	}

	return latestVersion.Object
}

func GetLatestMatchedVersion(versions storagev1.VersionsAccessor, versionPattern string) (latestVersion storagev1.VersionAccessor, latestMatchedVersion storagev1.VersionAccessor, err error) {
	// parse version
	splittedVersion := strings.Split(strings.ToLower(strings.TrimPrefix(versionPattern, "v")), ".")
	if len(splittedVersion) != 3 {
		return nil, nil, fmt.Errorf("couldn't parse version %s, expected version in format: 0.0.0", versionPattern)
	}

	// find latest version that matches our defined version
	var latestVersionObj *matchedVersion
	var latestMatchedVersionObj *matchedVersion
	for _, version := range versions.GetVersions() {
		parsedVersion, err := semver.Parse(strings.TrimPrefix(version.GetVersion(), "v"))
		if err != nil {
			continue
		}

		// does the version match our restrictions?
		if (splittedVersion[0] == "x" || splittedVersion[0] == "X" || strconv.FormatUint(parsedVersion.Major, 10) == splittedVersion[0]) &&
			(splittedVersion[1] == "x" || splittedVersion[1] == "X" || strconv.FormatUint(parsedVersion.Minor, 10) == splittedVersion[1]) &&
			(splittedVersion[2] == "x" || splittedVersion[2] == "X" || strconv.FormatUint(parsedVersion.Patch, 10) == splittedVersion[2]) {
			if latestMatchedVersionObj == nil || latestMatchedVersionObj.Version.LT(parsedVersion) {
				latestMatchedVersionObj = &matchedVersion{
					Object:  version,
					Version: parsedVersion,
				}
			}
		}

		// latest available version
		if latestVersionObj == nil || latestVersionObj.Version.LT(parsedVersion) {
			latestVersionObj = &matchedVersion{
				Object:  version,
				Version: parsedVersion,
			}
		}
	}

	if latestVersionObj != nil {
		latestVersion = latestVersionObj.Object
	}
	if latestMatchedVersionObj != nil {
		latestMatchedVersion = latestMatchedVersionObj.Object
	}

	return latestVersion, latestMatchedVersion, nil
}
