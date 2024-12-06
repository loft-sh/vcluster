package upgrade

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strings"

	"github.com/blang/semver"
	"github.com/google/go-github/v30/github"
	"github.com/pkg/errors"
	gitconfig "github.com/tcnksm/go-gitconfig"
	"golang.org/x/oauth2"
)

func fetchReleaseByTag(ctx context.Context, owner, repo, tag string) (*github.RepositoryRelease, error) {
	var (
		token string
		hc    *http.Client

		release *github.RepositoryRelease
	)

	if os.Getenv("GITHUB_TOKEN") != "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		token, _ = gitconfig.GithubToken()
	}

	if token == "" {
		hc = http.DefaultClient
	} else {
		src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		hc = oauth2.NewClient(ctx, src)
	}

	client := github.NewClient(hc)

	// Fetch the release by tag
	release, _, err := client.Repositories.GetReleaseByTag(ctx, owner, repo, tag)
	if err != nil {
		return release, fmt.Errorf("error fetching release by tag: %w", err)
	}

	return release, nil
}

func findAssetFromRelease(rel *github.RepositoryRelease) (*github.ReleaseAsset, semver.Version, bool, error) {
	// Generate candidates
	suffixes := make([]string, 0, 2*7*2)
	for _, sep := range []rune{'_', '-'} {
		for _, ext := range []string{".zip", ".tar.gz", ".tgz", ".gzip", ".gz", ".tar.xz", ".xz", ""} {
			suffix := fmt.Sprintf("%s%c%s%s", runtime.GOOS, sep, runtime.GOARCH, ext)
			suffixes = append(suffixes, suffix)
			if runtime.GOOS == "windows" {
				suffix = fmt.Sprintf("%s%c%s.exe%s", runtime.GOOS, sep, runtime.GOARCH, ext)
				suffixes = append(suffixes, suffix)
			}
		}
	}

	verText := rel.GetTagName()
	indices := reVersion.FindStringIndex(verText)
	if indices == nil {
		return nil, semver.Version{}, false, fmt.Errorf("skip version not adopting semver %s", verText)
	}
	if indices[0] > 0 {
		// Strip prefix of version
		verText = verText[indices[0]:]
	}

	// If semver cannot parse the version text, it means that the text is not adopting
	// the semantic versioning. So it should be skipped.
	ver, err := semver.Make(verText)
	if err != nil {
		return nil, semver.Version{}, false, fmt.Errorf("failed to parse a semantic version %s", verText)
	}

	filterRe, err := regexp.Compile("vcluster")
	if err != nil {
		return nil, semver.Version{}, false, errors.New("failed to compile regexp")
	}

	for _, asset := range rel.Assets {
		name := asset.GetName()

		// Skipping asset not matching filter
		if !filterRe.MatchString(name) {
			continue
		}

		for _, s := range suffixes {
			if strings.HasSuffix(name, s) { // require version, arch etc
				// default: assume single artifact
				return asset, ver, true, nil
			}
		}
	}

	return nil, semver.Version{}, false, fmt.Errorf("no suitable asset was found in release %s", rel.GetTagName())
}
