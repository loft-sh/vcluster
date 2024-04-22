package helmdownloader

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/go-logr/logr"
	"github.com/loft-sh/log"
	"github.com/loft-sh/utils/pkg/downloader"
	"github.com/loft-sh/utils/pkg/downloader/commands"
	"github.com/loft-sh/vcluster/pkg/constants"
)

// GetHelmBinaryPath checks for helm binary and downloads if it's not present.
func GetHelmBinaryPath(ctx context.Context, log log.BaseLogger) (string, error) {
	logger := logr.New(log.LogrLogSink())

	// test for helm
	helmExecutablePath, err := exec.LookPath("helm")
	if err != nil {
		helmExecutablePath, err = downloader.NewDownloader(commands.NewHelmV3Command(), logger, constants.VClusterFolder).EnsureCommand(ctx)
		if err != nil {
			return "", fmt.Errorf("error while installing helm: %w", err)
		}
	}
	return helmExecutablePath, nil
}
