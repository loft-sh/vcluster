package pro

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/loft-sh/vcluster/pkg/util/cliconfig"
	homedir "github.com/mitchellh/go-homedir"
)

func LoftBinaryName(version string) string {
	return fmt.Sprintf("%s-loft-%s-%s", version, runtime.GOOS, runtime.GOARCH)
}

func LoftBinaryFilePath(version string) (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", fmt.Errorf("failed to open vcluster pro configuration file from, unable to detect $HOME directory, falling back to default configuration, following error occurred: %w", err)
	}

	return filepath.Join(home, cliconfig.VclusterFolder, VclusterProFolder, BinariesFolder, LoftBinaryName(version)), nil
}

func DownloadLoftBinary(ctx context.Context, filePath, url string) error {
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
