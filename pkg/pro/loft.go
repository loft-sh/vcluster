package pro

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/google/go-github/v53/github"
	"github.com/loft-sh/vcluster/pkg/util/cliconfig"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/samber/lo"
)

var (
	LoftBinaryName = "loft"
	LoftConfigName = "config.json"
)

// LoftBinaryFilePath returns the path to the loft binary for the given version
func LoftBinaryFilePath(version string) (string, error) {
	dir, err := LoftWorkingDirectory(version)
	if err != nil {
		return "", fmt.Errorf("failed to open vcluster pro configuration file from, unable to detect working directory: %w", err)
	}

	return filepath.Join(dir, LoftBinaryName), nil
}

// LoftConfigFilePath returns the path to the loft config file for the given version
func LoftConfigFilePath(version string) (string, error) {
	dir, err := LoftWorkingDirectory(version)
	if err != nil {
		return "", fmt.Errorf("failed to open vcluster pro configuration file from, unable to detect working directory: %w", err)
	}

	return filepath.Join(dir, LoftConfigName), nil
}

// LoftConfigFilePath returns the path to the loft config file for the given version
func LoftWorkingDirectory(version string) (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", fmt.Errorf("failed to open vcluster pro configuration file from, unable to detect $HOME directory, falling back to default configuration, following error occurred: %w", err)
	}

	return filepath.Join(home, cliconfig.VclusterFolder, VclusterProFolder, BinariesFolder, version), nil
}

// downloadBinary downloads the loft binary from the given url to the given file path
func downloadBinary(ctx context.Context, filePath, url string) error {
	err := os.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory %s, following error occurred: %w", filepath.Dir(filePath), err)
	}

	out, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s, following error occurred: %w", filePath, err)
	}
	defer out.Close()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed create new request with context: %w", err)
	}

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("failed to download loft binary from %s, following error occurred: %w", url, err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write loft binary to %s, following error occurred: %w", filePath, err)
	}

	err = os.Chmod(filePath, 0755)
	if err != nil {
		return fmt.Errorf("failed to make loft binary executable, following error occurred: %w", err)
	}

	return nil
}

// downloadLoftBinary downloads the loft binary for the given version if it does not exist yet.
//
// Returns the path to the loft binary and the version tag
func downloadLoftBinary(ctx context.Context, version string) (string, error) {
	filePath, err := LoftBinaryFilePath(version)
	if err != nil {
		return "", fmt.Errorf("failed to get loft binary file path: %v", err)
	}

	_, err = os.Stat(filePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("failed to stat loft binary: %w", err)
	}
	if err == nil {
		return filePath, nil
	}

	client := github.NewClient(nil)

	release, _, err := client.Repositories.GetLatestRelease(ctx, "loft-sh", "loft")
	if err != nil {
		return "", fmt.Errorf("failed to get latest release: %w", err)
	}

	asset, found := lo.Find(release.Assets, func(asset *github.ReleaseAsset) bool {
		return fmt.Sprintf("loft-%s-%s", runtime.GOOS, runtime.GOARCH) == *asset.Name
	})

	if !found {
		return "", fmt.Errorf("failed to find loft binary for tag %s", version)
	}

	// download binary
	err = downloadBinary(ctx, filePath, asset.GetBrowserDownloadURL())
	if err != nil {
		return "", fmt.Errorf("failed to download loft binary: %w", err)
	}

	return filePath, nil
}

// downloadLatestLoftBinary downloads the latest loft binary if it does not exist yet.
//
// Returns the path to the loft binary and the version tag
func downloadLatestLoftBinary(ctx context.Context) (string, string, error) {
	client := github.NewClient(nil)

	release, _, err := client.Repositories.GetLatestRelease(ctx, "loft-sh", "loft")
	if err != nil {
		return "", "", fmt.Errorf("failed to get latest release: %w", err)
	}

	tagName := release.GetTagName()

	if tagName == "" {
		return "", "", fmt.Errorf("failed to get latest release tag name")
	}

	binaryPath, err := downloadLoftBinary(ctx, tagName)
	if err != nil {
		return "", "", fmt.Errorf("failed to download loft binary: %w", err)
	}

	return binaryPath, tagName, err
}

// LatestLoftBinary returns the path to the latest loft binary if it exists
//
// Returns the path to the loft binary and the version tag
func LatestLoftBinary(ctx context.Context) (string, string, error) {
	proConfig, err := GetConfig()
	if err != nil {
		return "", "", fmt.Errorf("failed to get pro config: %w", err)
	}

	filePath, err := LoftBinaryFilePath(proConfig.LatestVersion)
	if err != nil {
		return "", "", fmt.Errorf("failed to get loft binary file path: %w", err)
	}

	_, err = os.Stat(filePath)

	if time.Since(proConfig.LatestCheckAt).Hours() > 24 || os.Getenv("PRO_FORCE_UPDATE") == "true" || errors.Is(err, os.ErrNotExist) {
		_, version, err := downloadLatestLoftBinary(ctx)
		if err != nil {
			return "", "", fmt.Errorf("failed to download latest loft binary: %v", err)
		}

		proConfig.LatestVersion = version
		proConfig.LatestCheckAt = time.Now()

		err = WriteConfig(proConfig)
		if err != nil {
			return "", "", fmt.Errorf("failed to write pro config: %w", err)
		}
	}

	return filePath, proConfig.LatestVersion, nil
}

// LoftBinary returns the path to the loft binary for the given version
//
// Returns the path to the loft binary and the version tag
func LoftBinary(ctx context.Context, version string) (string, error) {
	return downloadLoftBinary(ctx, version)
}
