package lifecycle

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"time"
)

// helmV3Version is the helm 3 the specs use to create a release the way users
// on helm 3 did. It is downloaded rather than taken from PATH, so the spec
// still runs when CI moves its own helm to helm 4.
const helmV3Version = "v3.22.1"

// helmSHA256 pins the published sha256 of each helm archive the specs download,
// by version and platform
// (https://get.helm.sh/helm-<version>-<os>-<arch>.tar.gz.sha256sum). The helm 4
// spec downloads the version Dockerfile.cli.release ships, so a helm bump there
// must add its checksums here or the spec fails.
var helmSHA256 = map[string]map[string]string{
	"v3.22.1": {
		"linux-amd64":  "896cf63684a42a723d61c980369d6e588330927deb13676ca0d85006dd6da0ea",
		"linux-arm64":  "4c29e286b279d170fa048ddfc3e4e83dcd069ca9c52102766ba672d7149e694d",
		"darwin-amd64": "cf850a81b908f8eef0331e6b18776da2767bcac639ca84472311dad20008495c",
		"darwin-arm64": "e73da8a0b0d5193774fdff17ce6319c7436bb22ee229b45fd64f3c40368c753a",
	},
	"v4.3.2": {
		"linux-amd64":  "d1c0278d46beab79582d7e50f5b3dc206944ba8e577cde618513fd07daa53983",
		"linux-arm64":  "df31cb61680ef728f82c770e5280dd565617bee6b039d37e85c677288be4bbb0",
		"darwin-amd64": "9e0068e9ae18c441d27388190ee7f53ae65eba44bfe910b40bdac44c4b51fa71",
		"darwin-arm64": "6f3421de3a8cc2ac578c8cfb33b8edde290dde2871bc1bab12e24f2e5caeaa90",
	},
}

var helmVersionArg = regexp.MustCompile(`(?m)^ARG HELM_VERSION="(v[^"]+)"`)

// cliImageHelmVersion returns HELM_VERSION from Dockerfile.cli.release: the helm
// the vcluster-cli image ships, which this spec tests against.
func cliImageHelmVersion() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("cannot locate the source file to find Dockerfile.cli.release")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "..", "Dockerfile.cli.release")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	match := helmVersionArg.FindSubmatch(data)
	if match == nil {
		return "", fmt.Errorf("%s has no ARG HELM_VERSION", path)
	}
	return string(match[1]), nil
}

var (
	helmDirsMu sync.Mutex
	helmDirs   = map[string]string{}
)

// cachedHelmDir returns a directory holding the given helm release as dir/helm.
// It downloads each version once per test process, so the specs share one
// download. The directories live until the process exits.
func cachedHelmDir(ctx context.Context, version string) (string, error) {
	helmDirsMu.Lock()
	defer helmDirsMu.Unlock()
	if dir, ok := helmDirs[version]; ok {
		return dir, nil
	}
	dir, err := os.MkdirTemp("", "vcluster-e2e-helm-"+version+"-")
	if err != nil {
		return "", err
	}
	if err := downloadHelm(ctx, version, dir); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	helmDirs[version] = dir
	return dir, nil
}

// helmDownloadAttempts and helmDownloadBackoff bound the retries, so a short
// network blip on get.helm.sh does not fail the spec.
const (
	helmDownloadAttempts = 3
	helmDownloadBackoff  = 5 * time.Second
)

// downloadHelm downloads the helm release for this platform, checks it against
// helmSHA256 and writes the binary to dir/helm. It retries failed downloads but
// not a checksum mismatch.
func downloadHelm(ctx context.Context, version, dir string) error {
	platform := runtime.GOOS + "-" + runtime.GOARCH
	want, ok := helmSHA256[version][platform]
	if !ok {
		return fmt.Errorf("no pinned checksum for helm %s on %s: add it to helmSHA256", version, platform)
	}
	url := fmt.Sprintf("https://get.helm.sh/helm-%s-%s.tar.gz", version, platform)

	var archive []byte
	var err error
	for attempt := 1; ; attempt++ {
		archive, err = fetch(ctx, url)
		if err == nil || attempt == helmDownloadAttempts {
			break
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("%w (last error: %w)", ctx.Err(), err)
		case <-time.After(time.Duration(attempt) * helmDownloadBackoff):
		}
	}
	if err != nil {
		return fmt.Errorf("after %d attempts: %w", helmDownloadAttempts, err)
	}

	sum := sha256.Sum256(archive)
	if got := hex.EncodeToString(sum[:]); got != want {
		return fmt.Errorf("checksum of %s is %s, want %s", url, got, want)
	}
	return extractHelm(archive, platform, url, dir)
}

func fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: %s", url, resp.Status)
	}
	archive, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", url, err)
	}
	return archive, nil
}

func extractHelm(archive []byte, platform, url, dir string) error {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("%s has no %s/helm", url, platform)
		}
		if err != nil {
			return err
		}
		if header.Name != platform+"/helm" {
			continue
		}
		out, err := os.OpenFile(filepath.Join(dir, "helm"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return err
		}
		return out.Close()
	}
}
