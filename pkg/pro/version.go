package pro

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	"github.com/google/go-github/v53/github"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
)

var (
	MinimumVersionTag = "v3.3.0-alpha.24"
	MinimumVersion    = semver.MustParse(strings.TrimPrefix(MinimumVersionTag, "v"))
)

// LatestCompatibleVersion returns the latest compatible version of vCluster.Pro
func LatestCompatibleVersion(ctx context.Context) (string, error) {
	proConfig, err := GetConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get pro config: %w", err)
	}

	if time.Since(proConfig.LatestCheckAt).Hours() > 24 || os.Getenv("PRO_FORCE_UPDATE") == "true" || errors.Is(err, os.ErrNotExist) {
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

		proConfig.LatestVersion = version
		proConfig.LatestCheckAt = time.Now()

		err = WriteConfig(proConfig)
		if err != nil {
			return "", fmt.Errorf("failed to write pro config: %w", err)
		}
	}

	return proConfig.LatestVersion, nil
}
