package pro

import (
	"context"
	"fmt"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/google/go-github/v53/github"
	"github.com/loft-sh/log"
)

var (
	MinimumVersionTag = "v3.3.0-alpha.24"
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

	version := MinimumVersionTag

	ghVersion, err := semver.Parse(strings.TrimPrefix(tagName, "v"))
	if err != nil {
		log.GetInstance().Warnf("failed to parse latest release tag name, falling back to %s: %v", MinimumVersionTag, err)
	} else if ghVersion.GTE(MinimumVersion) {
		version = tagName
	}

	return version, nil
}
